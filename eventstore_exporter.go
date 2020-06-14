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

	timeout            time.Duration
	port               uint
	verbose            bool
	insecureSkipVerify bool

	eventStoreURL      string
	eventStoreUser     string
	eventStorePassword string
	clusterMode        string
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
	prometheus.MustRegister(NewCollector())

	http.Handle("/metrics", promhttp.Handler())
}

func readAndValidateConfig() {
	flag.StringVar(&eventStoreURL, "eventstore-url", "http://localhost:2113", "EventStore URL")
	flag.StringVar(&eventStoreUser, "eventstore-user", "", "EventStore User")
	flag.StringVar(&eventStorePassword, "eventstore-password", "", "EventStore Password")
	flag.UintVar(&port, "port", 9448, "Port to expose scraping endpoint on")
	flag.DurationVar(&timeout, "timeout", time.Second*10, "Timeout when calling EventStore")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	flag.StringVar(&clusterMode, "cluster-mode", "cluster", "Cluster mode: `cluster` or `single`. In single mode, calls to cluster status endpoints are skipped")
	flag.BoolVar(&insecureSkipVerify, "insecure-skip-verify", false, "Skip TLS certificatte verification for EventStore HTTP client")

	flag.Parse()

	if clusterMode != "cluster" && clusterMode != "single" {
		log.Fatalf("Unknown cluster mode %v, use 'cluster' or 'single'", clusterMode)
	}

	if (eventStoreUser != "" && eventStorePassword == "") || (eventStoreUser == "" && eventStorePassword != "") {
		log.Fatal("EventStore user and password should both be specified, or should both be empty")
	}

	log.WithFields(logrus.Fields{
		"eventStoreURL":      eventStoreURL,
		"eventStoreUser":     eventStoreUser,
		"port":               port,
		"timeout":            timeout,
		"verbose":            verbose,
		"clusterMode":        clusterMode,
		"insecureSkipVerify": insecureSkipVerify,
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

	readAndValidateConfig()
	setupLogger()

	initializeClient()

	serveLandingPage()
	serveMetrics()

	startHTTPServer()
}
