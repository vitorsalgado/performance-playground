package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/vitorsalgado/ad-tech-performance/internal/openrtb"
	"github.com/vitorsalgado/ad-tech-performance/internal/testcert"
)

// Config is the configuration for the DSP.
type Config struct {
	// Latency is the latency to add to the /bid endpoint.
	Latency time.Duration
}

// LatencyByHostname maps each DSP replica hostname to its configured latency.
// Used when running with docker-compose deploy.replicas; latencies cycle 0, 5ms, 10ms, 1s, 500ms.
var LatencyByHostname = map[string]time.Duration{
	"adtech_dsp_1": 0, "adtech_dsp_2": 5 * time.Millisecond, "adtech_dsp_3": 10 * time.Millisecond, "adtech_dsp_4": 1 * time.Second, "adtech_dsp_5": 500 * time.Millisecond,
	"adtech_dsp_6": 0, "adtech_dsp_7": 5 * time.Millisecond, "adtech_dsp_8": 10 * time.Millisecond, "adtech_dsp_9": 1 * time.Second, "adtech_dsp_10": 500 * time.Millisecond,
	"adtech_dsp_11": 0, "adtech_dsp_12": 5 * time.Millisecond, "adtech_dsp_13": 10 * time.Millisecond, "adtech_dsp_14": 1 * time.Second, "adtech_dsp_15": 500 * time.Millisecond,
	"adtech_dsp_16": 0, "adtech_dsp_17": 5 * time.Millisecond, "adtech_dsp_18": 10 * time.Millisecond, "adtech_dsp_19": 1 * time.Second, "adtech_dsp_20": 500 * time.Millisecond,
	"adtech_dsp_21": 0, "adtech_dsp_22": 5 * time.Millisecond, "adtech_dsp_23": 10 * time.Millisecond, "adtech_dsp_24": 1 * time.Second, "adtech_dsp_25": 500 * time.Millisecond,
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	config := Config{}

	hostname, err := os.Hostname()
	if err != nil {
		logger.Error("error getting hostname", slog.Any("error", err))
		os.Exit(1)
	}

	if latency, ok := LatencyByHostname[hostname]; ok {
		config.Latency = latency
		logger.Info("latency from hostname map", slog.String("hostname", hostname), slog.Duration("latency", latency))
	} else {
		config.Latency = 0
		logger.Info("hostname not in latency map, using 0", slog.String("hostname", hostname))
	}

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	mux := http.NewServeMux()
	server := &http.Server{Addr: ":8080", Handler: mux, BaseContext: func(l net.Listener) context.Context { return rootCtx }}

	cert, err := tls.X509KeyPair(testcert.LocalhostCert, testcert.LocalhostKey)
	if err != nil {
		logger.Error("error creating TLS certificate", slog.Any("error", err))
		os.Exit(1)
	}

	server.TLSConfig = &tls.Config{Certificates: []tls.Certificate{cert}}

	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("pong")) })

	// Prometheus metrics collector
	// VictoriaMetrics will scrape metrics through this endpoint.
	mux.Handle("/metrics", promhttp.Handler())

	// Bid endpoint
	// /bid is the main endpoint for the DSP and will be used for performance testing.
	// --

	mux.HandleFunc("/bid", func(w http.ResponseWriter, r *http.Request) {
		if config.Latency > 0 {
			time.Sleep(config.Latency)
		}

		bid := &openrtb.BidResponse{
			ID:      "123",
			SeatBid: []openrtb.SeatBid{{Bid: []openrtb.Bid{{ID: "123", Price: 1.0, ImpID: "123"}}}},
		}

		if err := json.NewEncoder(w).Encode(bid); err != nil {
			logger.Error("error encoding bid", slog.Any("error", err))
		}
	})

	// Starting the HTTP server

	// Graceful shutdown
	go func() {
		<-rootCtx.Done()
		stop()

		c, fn := context.WithTimeout(context.Background(), 5*time.Second)
		defer fn()

		if err := server.Shutdown(c); err != nil {
			logger.Error("error during shutdown", slog.Any("error", err))
		}
	}()

	logger.Info("starting")

	if err := server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
		logger.Error("server error", slog.Any("error", err))
	}
}
