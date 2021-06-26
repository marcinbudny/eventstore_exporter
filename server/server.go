package server

import (
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/marcinbudny/eventstore_exporter/collector"
	"github.com/marcinbudny/eventstore_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	log.Fatal(http.ListenAndServe(listenAddr, server.mux))
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
