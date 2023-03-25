package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/marcinbudny/eventstore_exporter/internal/collector"
	"github.com/marcinbudny/eventstore_exporter/internal/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

type ExporterServer struct {
	config    *config.Config
	collector *collector.Collector
	mux       *http.ServeMux
}

func NewExporterServer(config *config.Config, collector *collector.Collector) *ExporterServer {
	server := &ExporterServer{
		config:    config,
		collector: collector,
		mux:       http.NewServeMux(),
	}
	server.serveLandingPage()
	server.serveMetrics()

	return server
}

func (server *ExporterServer) ListenAndServe() {
	listenAddr := fmt.Sprintf(":%d", server.config.Port)

	srv := &http.Server{
		Addr:         listenAddr,
		Handler:      server.mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}

func (server *ExporterServer) serveLandingPage() {
	var landingPage = []byte(`<html>
		<head><title>EventStore exporter for Prometheus</title></head>
		<body>
		<h1>EventStore exporter for Prometheus</h1>
		<p><a href='/metrics'>Metrics</a></p>
		</body>
		</html>
		`)

	server.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write(landingPage) // nolint: errcheck
	})
}

func (server *ExporterServer) serveMetrics() {
	registry := prometheus.NewRegistry()
	registry.MustRegister(server.collector)

	server.mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
}
