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
	Latency time.Duration
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	config := Config{}

	latency, err := time.ParseDuration(os.Getenv("DSP_LATENCY"))
	if err != nil {
		logger.Error("error parsing DSP_LATENCY", slog.Any("error", err))
		os.Exit(1)
	}

	config.Latency = latency

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	mux := http.NewServeMux()
	server := &http.Server{Addr: ":8080", Handler: mux, BaseContext: func(l net.Listener) context.Context { return rootCtx }}

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

	logger.Info("starting")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server error", slog.Any("error", err))
	}
}
