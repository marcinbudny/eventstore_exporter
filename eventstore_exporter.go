package main

import (
	"os"

	"github.com/marcinbudny/eventstore_exporter/internal/client"
	"github.com/marcinbudny/eventstore_exporter/internal/collector"
	"github.com/marcinbudny/eventstore_exporter/internal/config"
	"github.com/marcinbudny/eventstore_exporter/internal/server"
	log "github.com/sirupsen/logrus"
)

func readAndValidateConfig() *config.Config {
	config, err := config.Load(os.Args[1:], false)
	if err != nil {
		log.Fatal(err)
		return nil
	}

	password := config.EventStorePassword
	if password != "" {
		password = "**REDACTED**" // nolint:gosec
	}
	log.WithFields(log.Fields{
		"eventStoreURL":             config.EventStoreURL,
		"eventStoreUser":            config.EventStoreUser,
		"eventStorePassword":        password,
		"port":                      config.Port,
		"timeout":                   config.Timeout,
		"verbose":                   config.Verbose,
		"insecureSkipVerify":        config.InsecureSkipVerify,
		"enableParkedMessagesStats": config.EnableParkedMessagesStats,
		"streams":                   config.Streams,
	}).Infof("EventStore exporter configured")

	return config
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
