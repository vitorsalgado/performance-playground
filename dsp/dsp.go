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

	"github.com/vitorsalgado/ad-tech-performance/internal/environ"
	"github.com/vitorsalgado/ad-tech-performance/internal/openrtb"
	"github.com/vitorsalgado/ad-tech-performance/internal/testcert"
)

const latencyQueryParam = "latency"

// Config is the configuration for the DSP.
type Config struct {
	// Latency is the latency to add to the /bid endpoint.
	Latency time.Duration
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	var config = Config{}
	var err error
	config.Latency, err = environ.GetDuration("DSP_LATENCY", 0)
	if err != nil {
		logger.Error("error parsing DSP_LATENCY", slog.Any("error", err))
		os.Exit(1)
	}
	if config.Latency > 0 {
		logger.Info("latency from env", slog.Duration("latency", config.Latency))
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
		latency := config.Latency
		if s := r.URL.Query().Get(latencyQueryParam); s != "" {
			if d, err := time.ParseDuration(s); err == nil && d >= 0 {
				latency = d
			}
		}
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
