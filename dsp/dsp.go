package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/vitorsalgado/ad-tech-performance/internal/openrtb"
)

// Config is the configuration for the DSP.
type Config struct {
	// Latency is the latency to add to the /bid endpoint.
	// It is a string that represents a duration in milliseconds.
	// Example: "100ms", "1s", "1000ms".
	Latency string `json:"latency"`
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cp := os.Getenv("DSP_CONFIG_PATH")
	f, err := os.Open(cp)
	if err != nil {
		logger.Error("error opening config file", slog.Any("error", err))
		os.Exit(1)
	}
	defer f.Close()

	config := new(Config)
	if err := json.NewDecoder(f).Decode(config); err != nil {
		logger.Error("error decoding config file", slog.Any("error", err))
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	mux := http.NewServeMux()
	server := &http.Server{Addr: ":8080", Handler: mux, BaseContext: func(l net.Listener) context.Context { return ctx }}

	// Graceful shutdown
	go func() {
		<-ctx.Done()
		stop()

		c, fn := context.WithTimeout(context.Background(), 5*time.Second)
		defer fn()

		if err := server.Shutdown(c); err != nil {
			logger.Error("error during shutdown", slog.Any("error", err))
		}
	}()

	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("pong")) })

	// Prometheus metrics collector
	// VictoriaMetrics will scrape metrics through this endpoint.
	registerer := prometheus.NewRegistry()
	registerer.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	mux.Handle("/metrics", promhttp.HandlerFor(registerer, promhttp.HandlerOpts{}))

	// Bid endpoint
	// /bid is the main endpoint for the DSP and will be used for performance testing.
	// --

	// Configuring the /bid endpoint
	latency, err := time.ParseDuration(config.Latency)
	if err != nil {
		logger.Error("error parsing latency", slog.Any("error", err))
		os.Exit(1)
	}

	mux.HandleFunc("/bid", func(w http.ResponseWriter, r *http.Request) {
		if latency > 0 {
			time.Sleep(latency)
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

	logger.Info("starting")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server error", slog.Any("error", err))
	}
}
