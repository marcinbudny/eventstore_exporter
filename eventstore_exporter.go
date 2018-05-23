package main

import (
	"fmt"
	"net/http"
	"time"
	
	"github.com/namsral/flag"

	"github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	log = logrus.New()

	timeout time.Duration
	port    uint
	verbose bool

	eventStoreURL  string
)

func serveLandingPage() {
	var landingPage = []byte(`<html>
		<head><title>EventStore exporter for Prometheus</title></head>
		<body>
		<h1>EventStore exporter for Prometheus</h1>
		<p><a href='/metrics'>Metrics</a></p>
		</body>
		</html>
		`)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write(landingPage) // nolint: errcheck
	})
}

func serveMetrics() {
	prometheus.MustRegister(newExporter())

	http.Handle("/metrics", promhttp.Handler())
}

func readConfig() {
	flag.StringVar(&eventStoreURL, "eventstore-url", "http://localhost:2113", "EventStore URL")
	flag.UintVar(&port, "port", 9999, "Port to expose scraping endpoint on")
	flag.DurationVar(&timeout, "timeout", time.Second*10, "Timeout when calling EventStore")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")

	flag.Parse()

	log.WithFields(logrus.Fields{
		"eventStoreURL": eventStoreURL,
		"port":       port,
		"timeout":    timeout,
		"verbose":    verbose,
	}).Infof("EventStore exporter configured")
}

func setupLogger() {
	if verbose {
		log.Level = logrus.DebugLevel
	}
}

func startHTTPServer() {
	listenAddr := fmt.Sprintf(":%d", port)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}

func main() {

	readConfig()
	setupLogger()

	initializeClient()

	serveLandingPage()
	serveMetrics()

	startHTTPServer()
}

