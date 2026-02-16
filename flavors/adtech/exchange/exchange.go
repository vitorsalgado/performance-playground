package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/pprof"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/errgroup"

	"perftest/internal/environ"
	"perftest/internal/openrtb"
)

// Models
// --

type App struct {
	ID        int        `json:"id"`
	Name      string     `json:"name"`
	Publisher *Publisher `json:"publisher"`
}

type Publisher struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Apps holds a map of applications for quick lookup.
type Apps struct {
	Apps map[int]*App
}

// DSP represents a DSP with its endpoint and optional latency.
type DSP struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Endpoint string `json:"endpoint"`
	Latency  string `json:"latency"`
}

// DSPs holds a map of DSPs for quick lookup.
type DSPs struct {
	DSPs []*DSP
}

// Cache
// The cache holds application data that can be served quickly without hitting a database.
// This is a simplified example with only application data to test the performance overhead of the whole
// caching process.
// Ideally, the cache entries should be big.
// --

// CacheLoadFunc represents a function that loads cache data.
type CacheLoadFunc func(state *State, logger *slog.Logger) error

type State struct {
	Apps atomic.Pointer[Apps]
	DSPs atomic.Pointer[DSPs]
}

// Cache manages the in-memory cache objects needed by the application.
type Cache struct {
	state  *State
	plan   map[string]CacheLoadFunc
	logger *slog.Logger
	done   chan struct{}
}

// NewCache creates a new cache with the given logger and plan.
func NewCache(logger *slog.Logger, plan map[string]CacheLoadFunc) *Cache {
	return &Cache{state: &State{}, plan: plan, logger: logger, done: make(chan struct{})}
}

// Start starts the cache loading process.
// The cache will periodically reload the data from the underlying data source.
func (c *Cache) Start(ctx context.Context, interval time.Duration) {
	go c.worker(ctx, interval)

	c.logger.Info("cache: started", slog.Duration("interval", interval))
}

func (c *Cache) worker(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.done:
			return
		case <-ticker.C:
			c.Load(ctx)
		}
	}
}

// Stop stops the cache loading process.
func (c *Cache) Stop() {
	close(c.done)
}

// Load loads all cache data in parallel.
func (c *Cache) Load(ctx context.Context) error {
	group, ctx := errgroup.WithContext(ctx)

	for name, action := range c.plan {
		group.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if err := action(c.state, c.logger); err != nil {
				c.logger.Error("cache: error loading", slog.String("name", name), slog.Any("error", err))
				return err
			}

			c.logger.Info("cache: loaded", slog.String("name", name))

			return nil
		})
	}

	return group.Wait()
}

// CacheLoadApps loads the apps from the given path.
func CacheLoadApps(path string) CacheLoadFunc {
	return func(state *State, logger *slog.Logger) error {
		f, err := os.Open(path)
		if err != nil {
			return err
		}

		defer f.Close()

		var apps []App
		if err = json.NewDecoder(f).Decode(&apps); err != nil {
			return err
		}

		appMap := make(map[int]*App, len(apps))
		for i := range apps {
			app := apps[i]
			appMap[app.ID] = &app
		}

		state.Apps.Store(&Apps{Apps: appMap})

		logger.Info("cache: loaded apps", slog.Int("count", len(apps)))

		return nil
	}
}

// CacheLoadDSPs loads the DSPs from the given path.
func CacheLoadDSPs(path string) CacheLoadFunc {
	return func(state *State, logger *slog.Logger) error {
		f, err := os.Open(path)
		if err != nil {
			return err
		}

		defer f.Close()

		var dsps []*DSP
		if err = json.NewDecoder(f).Decode(&dsps); err != nil {
			return err
		}

		state.DSPs.Store(&DSPs{DSPs: dsps})

		for _, dsp := range dsps {
			gDSPConfigInfo.WithLabelValues(strconv.Itoa(dsp.ID)).Set(1)
		}

		logger.Info("cache: loaded dsps", slog.Int("count", len(dsps)))

		return nil
	}
}

// DSP IO
// The DSP IO is responsible for handling the DSP requests and responses.
// --

// In represents the input to execute a DSP request.
type In struct {
	ID         int
	DSPID      int
	BidRequest *http.Request
	Responder  chan<- Out
	Timestamp  time.Time
}

// Out represents the response of a DSP request.
type Out struct {
	ID          int
	DSPID       int
	BidResponse openrtb.BidResponse
	Err         error
}

// DSPIO represents the actual DSP IO handler.
type DSPIO struct {
	logger    *slog.Logger
	transport *http.Transport
	pool      int
	input     chan In
	done      chan struct{}
}

// NewDSPIO creates a new DSP IO handler.
func NewDSPIO(logger *slog.Logger, transport *http.Transport, pool int) *DSPIO {
	return &DSPIO{
		logger:    logger,
		transport: transport,
		pool:      pool,
		input:     make(chan In),
		done:      make(chan struct{}),
	}
}

// Start starts the DSP IO background workers.
// The workers will execute the DSP requests in the background.
func (d *DSPIO) Start(ctx context.Context) {
	for range d.pool {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-d.done:
					return
				case in := <-d.input:
					d.Execute(in)
				}
			}
		}()
	}
}

// Stop stops the DSP IO background workers.
func (d *DSPIO) Stop() {
	close(d.done)
}

// Enqueue enqueues a DSP request to be executed by the background workers.
func (d *DSPIO) Enqueue(in In) {
	d.logger.Info("dspio: enqueued request", slog.Int("dsp_id", in.DSPID), slog.Int("id", in.ID))

	mDSPRequestTotal.
		WithLabelValues(strconv.Itoa(in.DSPID)).
		Inc()

	select {
	case d.input <- in:
		return
	default:
	}

	mDSPRequestDropped.
		WithLabelValues(strconv.Itoa(in.DSPID)).
		Inc()

	in.Responder <- Out{
		ID:    in.ID,
		DSPID: in.DSPID,
		Err:   errors.New("dspio: queue is full"),
	}
}

// Execute executes the DSP request.
func (d *DSPIO) Execute(in In) {
	rateDSPConcurrency.Inc()
	defer rateDSPConcurrency.Dec()

	d.logger.Info("dspio: executing request", slog.Int("dsp_id", in.DSPID), slog.Int("id", in.ID))

	start := time.Now()
	res, err := d.transport.RoundTrip(in.BidRequest)
	elapsed := time.Since(start).Seconds()
	dspIDStr := strconv.Itoa(in.DSPID)

	hDSPRequestDuration.WithLabelValues(dspIDStr).Observe(elapsed)

	if err != nil {
		d.logger.Info("dspio: response error", slog.Int("dsp_id", in.DSPID), slog.Int("id", in.ID), slog.Any("error", err))
		mDSPRequestError.WithLabelValues(dspIDStr).Inc()
		in.Responder <- Out{ID: in.ID, DSPID: in.DSPID, Err: err}
		return
	}

	var bidResponse openrtb.BidResponse
	if err = json.NewDecoder(res.Body).Decode(&bidResponse); err != nil {
		d.logger.Info("dspio: response decode error", slog.Int("dsp_id", in.DSPID), slog.Int("id", in.ID), slog.Any("error", err))
		mDSPRequestError.WithLabelValues(dspIDStr).Inc()
		in.Responder <- Out{ID: in.ID, DSPID: in.DSPID, Err: err}
		return
	}

	d.logger.Info("dspio: success", slog.Int("dsp_id", in.DSPID), slog.Int("id", in.ID))

	in.Responder <- Out{
		ID:          in.ID,
		DSPID:       in.DSPID,
		BidResponse: bidResponse,
		Err:         nil,
	}
}

// Metrics
// --
// DSP IO metrics.
var rateDSPConcurrency = prometheus.NewGauge(prometheus.GaugeOpts{Name: "dspio_concurrency_rate"})
var mDSPRequestTotal = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "dspio_request_total"}, []string{"dsp_id"})
var mDSPRequestDropped = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "dspio_request_dropped_total"}, []string{"dsp_id"})
var mDSPRequestError = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "dspio_request_error_total"}, []string{"dsp_id"})
var mDSPConnDialTotal = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "dspio_conn_dial_total"}, []string{"host"})
var hDSPRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "dspio_request_duration_seconds",
	Help:    "Time spent waiting for DSP bid response.",
	Buckets: prometheus.ExponentialBuckets(0.001, 2, 14), // 1ms to ~8s
}, []string{"dsp_id"})

// Ad request metrics.
var counterTotalAdRequest = prometheus.NewCounter(prometheus.CounterOpts{Name: "ad_request_total"})
var mTotalAdRequestPerPubAndApp = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "ad_request_per_pub_and_app_total"}, []string{"pub_id", "app_id"})

// DSP exchange metrics.
var mDSPBeforePerPub = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "dsp_before_per_pub_total"}, []string{"dsp_id", "pub_id"})
var mDSPAfterPerPub = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "dsp_after_per_pub_total"}, []string{"dsp_id", "pub_id"})

// Config info: always-exposed metrics so dashboard variables (e.g. dsp_id) have options before traffic.
var gDSPConfigInfo = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Name: "exchange_dsp_config_info",
	Help: "Configured DSPs (1 per dsp_id). Used for dashboard label_values so dsp_id variable is populated.",
}, []string{"dsp_id"})

// Main application logic.
// --

func init() {
	prometheus.MustRegister(
		rateDSPConcurrency,
		mDSPRequestTotal,
		mDSPRequestDropped,
		mDSPRequestError,
		mDSPConnDialTotal,
		hDSPRequestDuration,
		counterTotalAdRequest,
		mTotalAdRequestPerPubAndApp,
		mDSPBeforePerPub,
		mDSPAfterPerPub,
		gDSPConfigInfo,
	)
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	mux := http.NewServeMux()
	server := &http.Server{Addr: ":8080", Handler: mux, BaseContext: func(l net.Listener) context.Context { return rootCtx }}

	// Cache
	// --
	cacheUpdateInterval, err := environ.GetDuration("EXCHANGE_CACHE_UPDATE_INTERVAL", 1*time.Minute)
	if err != nil {
		logger.Error("main: failed to parse EXCHANGE_CACHE_UPDATE_INTERVAL", slog.Any("error", err))
		os.Exit(1)
	}

	plan := make(map[string]CacheLoadFunc, 2)
	plan["apps"] = CacheLoadApps(os.Getenv("EXCHANGE_APPS_CACHE_PATH"))
	plan["dsps"] = CacheLoadDSPs(os.Getenv("EXCHANGE_DSPS_CACHE_PATH"))

	cache := NewCache(logger, plan)
	if err := cache.Load(rootCtx); err != nil {
		logger.Error("main: failed to load cache", slog.Any("error", err))
		os.Exit(1)
	}
	cache.Start(rootCtx, cacheUpdateInterval)

	// DSP IO
	// --
	maxIdleConns, err := environ.GetInt("EXCHANGE_DSPIO_MAX_IDLE_CONNS", 100)
	if err != nil {
		logger.Error("main: failed to parse EXCHANGE_DSPIO_MAX_IDLE_CONNS", slog.Any("error", err))
		os.Exit(1)
	}
	maxIdleConnsPerHost, err := environ.GetInt("EXCHANGE_DSPIO_MAX_IDLE_CONNS_PER_HOST", 100)
	if err != nil {
		logger.Error("main: failed to parse EXCHANGE_DSPIO_MAX_IDLE_CONNS_PER_HOST", slog.Any("error", err))
		os.Exit(1)
	}
	idleConnTimeout, err := environ.GetDuration("EXCHANGE_DSPIO_IDLE_CONN_TIMEOUT", 15*time.Second)
	if err != nil {
		logger.Error("main: failed to parse EXCHANGE_DSPIO_IDLE_CONN_TIMEOUT", slog.Any("error", err))
		os.Exit(1)
	}
	keepAlive, err := environ.GetDuration("EXCHANGE_DSPIO_KEEP_ALIVE", 30*time.Second)
	if err != nil {
		logger.Error("main: failed to parse EXCHANGE_DSPIO_KEEP_ALIVE", slog.Any("error", err))
		os.Exit(1)
	}
	timeout, err := environ.GetDuration("EXCHANGE_DSPIO_TIMEOUT", 30*time.Second)
	if err != nil {
		logger.Error("main: failed to parse EXCHANGE_DSPIO_TIMEOUT", slog.Any("error", err))
		os.Exit(1)
	}
	pool, err := environ.GetInt("EXCHANGE_DSPIO_POOL", 100)
	if err != nil {
		logger.Error("main: failed to parse EXCHANGE_DSPIO_POOL", slog.Any("error", err))
		os.Exit(1)
	}

	insecureSkipVerify, err := environ.GetBool("EXCHANGE_DSPIO_INSECURE_SKIP_VERIFY", true)
	if err != nil {
		logger.Error("main: failed to parse EXCHANGE_DSPIO_INSECURE_SKIP_VERIFY", slog.Any("error", err))
		os.Exit(1)
	}

	responseHeaderTimeout, err := environ.GetDuration("EXCHANGE_DSPIO_RESPONSE_HEADER_TIMEOUT", 10*time.Second)
	if err != nil {
		logger.Error("main: failed to parse EXCHANGE_DSPIO_RESPONSE_HEADER_TIMEOUT", slog.Any("error", err))
		os.Exit(1)
	}
	expectContinueTimeout, err := environ.GetDuration("EXCHANGE_DSPIO_EXPECT_CONTINUE_TIMEOUT", 1*time.Second)
	if err != nil {
		logger.Error("main: failed to parse EXCHANGE_DSPIO_EXPECT_CONTINUE_TIMEOUT", slog.Any("error", err))
		os.Exit(1)
	}
	forceHTTP2, err := environ.GetBool("EXCHANGE_DSPIO_FORCE_HTTP2", true)
	if err != nil {
		logger.Error("main: failed to parse EXCHANGE_DSPIO_FORCE_HTTP2", slog.Any("error", err))
		os.Exit(1)
	}
	requestTimeout, err := environ.GetDuration("EXCHANGE_DSPIO_REQUEST_TIMEOUT", 500*time.Millisecond)
	if err != nil {
		logger.Error("main: failed to parse EXCHANGE_DSPIO_REQUEST_TIMEOUT", slog.Any("error", err))
		os.Exit(1)
	}

	logger.Info("main: DSP IO transport config",
		slog.Duration("dial_timeout", timeout),
		slog.Duration("keep_alive", keepAlive),
		slog.Duration("idle_conn_timeout", idleConnTimeout),
		slog.Duration("response_header_timeout", responseHeaderTimeout),
		slog.Duration("expect_continue_timeout", expectContinueTimeout),
		slog.Bool("force_http2", forceHTTP2),
		slog.Bool("insecure_skip_verify", insecureSkipVerify),
		slog.Duration("request_timeout", requestTimeout),
	)

	dialer := &net.Dialer{Timeout: timeout, KeepAlive: keepAlive}
	transport := &http.Transport{
		MaxIdleConns:        maxIdleConns,
		MaxIdleConnsPerHost: maxIdleConnsPerHost,
		IdleConnTimeout:     idleConnTimeout,
		TLSClientConfig: &tls.Config{
			ClientSessionCache: tls.NewLRUClientSessionCache(256),
			InsecureSkipVerify: insecureSkipVerify,
		},
		ForceAttemptHTTP2: forceHTTP2,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			c, err := dialer.DialContext(ctx, network, addr)
			if err != nil {
				return c, err
			}

			sep := strings.LastIndex(addr, ":")
			mDSPConnDialTotal.WithLabelValues(addr[:sep]).Inc()

			return c, nil
		},
	}
	dspio := NewDSPIO(logger, transport, pool)
	dspio.Start(rootCtx)

	// HTTP endpoints
	// --
	// Ping/Pong
	// Simple endpoint to check if the server is running.
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("pong")) })
	// Profiling endpoints.
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	mux.Handle("/debug/pprof/block", pprof.Handler("block"))
	// Prometheus metrics collector.
	// VictoriaMetrics will scrape metrics through this endpoint.
	mux.Handle("/metrics", promhttp.Handler())

	// Ad request endpoint.
	// This is the main endpoint that will be used for experimentation.
	mux.HandleFunc("/ad", func(w http.ResponseWriter, r *http.Request) {
		counterTotalAdRequest.Inc()

		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer gz.Close()

		var adRequest openrtb.BidRequest
		if err = json.NewDecoder(gz).Decode(&adRequest); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		apps := cache.state.Apps.Load()
		appid, err := strconv.Atoi(adRequest.App.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		app := apps.Apps[appid]
		if app == nil {
			http.Error(w, "app not found", http.StatusNotFound)
			return
		}

		mTotalAdRequestPerPubAndApp.
			WithLabelValues(strconv.Itoa(app.Publisher.ID), strconv.Itoa(app.ID)).
			Inc()

		dsps := cache.state.DSPs.Load()
		responses := make(chan Out, len(dsps.DSPs))
		ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
		defer cancel()
		// Do not close `responses`: DSP IO workers may still send after we return,
		// and closing here would risk panics ("send on closed channel").

		body, err := json.Marshal(adRequest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		for i, dsp := range dsps.DSPs {
			mDSPBeforePerPub.
				WithLabelValues(strconv.Itoa(dsp.ID), strconv.Itoa(app.Publisher.ID)).
				Inc()

			buf := new(bytes.Buffer)
			gzw := gzip.NewWriter(buf)
			if _, err := gzw.Write(body); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := gzw.Close(); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			bidURL := dsp.Endpoint
			if dsp.Latency != "" {
				u, err := url.Parse(dsp.Endpoint)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				q := u.Query()
				q.Set("latency", dsp.Latency)
				u.RawQuery = q.Encode()
				bidURL = u.String()
			}

			req, err := http.NewRequestWithContext(ctx, http.MethodPost, bidURL, buf)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Content-Encoding", "gzip")

			dspio.Enqueue(In{
				ID:         i,
				DSPID:      dsp.ID,
				BidRequest: req,
				Responder:  responses,
				Timestamp:  time.Now(),
			})

			mDSPAfterPerPub.
				WithLabelValues(strconv.Itoa(dsp.ID), strconv.Itoa(app.Publisher.ID)).
				Inc()
		}

		n := len(dsps.DSPs)
		bidResponses := make([]Out, 0, n)

	loop:
		for range n {
			select {
			case out := <-responses:
				if out.Err == nil {
					bidResponses = append(bidResponses, out)
				} else {
					logger.Error("exchange: error from dsp", slog.Int("dsp_id", out.DSPID), slog.Any("error", out.Err))
				}
			case <-ctx.Done():
				break loop
			}
		}

		var bidResponse openrtb.BidResponse
		if len(bidResponses) > 0 {
			bidResponse = bidResponses[0].BidResponse
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(bidResponse); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	// Starting the HTTP server
	// --
	// Graceful shutdown
	go func() {
		<-rootCtx.Done()
		stop()

		cache.Stop()
		dspio.Stop()

		c, fn := context.WithTimeout(context.Background(), 5*time.Second)
		defer fn()

		if err := server.Shutdown(c); err != nil {
			logger.Error("error during shutdown", slog.Any("error", err))
		}
	}()

	logger.Info("starting")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server error", slog.Any("error", err))
	}
}
