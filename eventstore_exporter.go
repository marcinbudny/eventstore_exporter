package main

import (
	"fmt"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/marcinbudny/eventstore_exporter/client"
	"github.com/marcinbudny/eventstore_exporter/collector"
	"github.com/marcinbudny/eventstore_exporter/config"
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

func serveMetrics(config *config.Config, client *client.EventStoreStatsClient) {
	prometheus.MustRegister(collector.NewCollector(config, client))

	http.Handle("/metrics", promhttp.Handler())
}

func readAndValidateConfig() *config.Config {
	if config, err := config.Load(os.Args[1:], false); err == nil {
		password := config.EventStorePassword
		if password != "" {
			password = "**REDACTED**"
		}
		log.WithFields(log.Fields{
			"eventStoreURL":             config.EventStoreURL,
			"eventStoreUser":            config.EventStoreUser,
			"eventStorePassword":        password,
			"port":                      config.Port,
			"timeout":                   config.Timeout,
			"verbose":                   config.Verbose,
			"clusterMode":               config.ClusterMode,
			"insecureSkipVerify":        config.InsecureSkipVerify,
			"enableParkedMessagesStats": config.EnableParkedMessagesStats,
		}).Infof("EventStore exporter configured")

		return config
	} else {
		log.Fatal(err)
		return nil
	}
}

func setupLogger(config *config.Config) {
	if config.Verbose {
		log.SetLevel(log.DebugLevel)
	}
}

func startHTTPServer(config *config.Config) {
	listenAddr := fmt.Sprintf(":%d", config.Port)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}

func main() {

	config := readAndValidateConfig()
	setupLogger(config)

	client := client.New(config)

	serveLandingPage()
	serveMetrics(config, client)

	startHTTPServer(config)
}
