package main

import (
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/marcinbudny/eventstore_exporter/internal/client"
	"github.com/marcinbudny/eventstore_exporter/internal/collector"
	"github.com/marcinbudny/eventstore_exporter/internal/config"
	"github.com/marcinbudny/eventstore_exporter/internal/server"
)

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

func main() {

	config := readAndValidateConfig()
	setupLogger(config)

	client := client.New(config)
	collector := collector.NewCollector(config, client)

	exporterServer := server.NewExporterServer(config, collector)
	exporterServer.ListenAndServe()
}
