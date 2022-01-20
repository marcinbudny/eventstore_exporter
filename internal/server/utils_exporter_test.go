package server

import (
	"os"
	"time"

	"github.com/marcinbudny/eventstore_exporter/internal/client"
	"github.com/marcinbudny/eventstore_exporter/internal/collector"
	"github.com/marcinbudny/eventstore_exporter/internal/config"
)

func prepareExporterServer() *ExporterServer {
	return prepareExporterServerWithConfig(func(_ *config.Config) {})
}

func prepareExporterServerWithConfig(updateConfig func(*config.Config)) *ExporterServer {
	eventStoreURL := getEventStoreURL()

	clusterMode := "single"
	if os.Getenv("TEST_CLUSTER_MODE") != "" {
		clusterMode = os.Getenv("TEST_CLUSTER_MODE")
	}

	config := &config.Config{
		EventStoreURL:             eventStoreURL,
		EventStoreUser:            "admin",
		EventStorePassword:        "changeit",
		ClusterMode:               clusterMode,
		InsecureSkipVerify:        true,
		Timeout:                   time.Second * 10,
		EnableParkedMessagesStats: true,
	}

	if updateConfig != nil {
		updateConfig(config)
	}

	client := client.New(config)
	collector := collector.NewCollector(config, client)
	return NewExporterServer(config, collector)
}

func prepareExporterServerWithInvalidConnection() *ExporterServer {
	eventStoreURL := "http://does_not_exist"

	config := &config.Config{
		EventStoreURL:      eventStoreURL,
		EventStoreUser:     "admin",
		EventStorePassword: "changeit",
		ClusterMode:        "single",
		InsecureSkipVerify: true,
		Timeout:            time.Second * 10,
	}

	client := client.New(config)
	collector := collector.NewCollector(config, client)
	return NewExporterServer(config, collector)
}
