package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/namsral/flag"
)

type Config struct {
	Timeout            time.Duration
	Port               uint
	Verbose            bool
	InsecureSkipVerify bool

	EventStoreURL             string
	EventStoreUser            string
	EventStorePassword        string
	ClusterMode               string
	EnableParkedMessagesStats bool
}

func Load() (*Config, error) {
	config := &Config{}

	flag.StringVar(&config.EventStoreURL, "eventstore-url", "http://localhost:2113", "EventStore URL")
	flag.StringVar(&config.EventStoreUser, "eventstore-user", "", "EventStore User")
	flag.StringVar(&config.EventStorePassword, "eventstore-password", "", "EventStore Password")
	flag.UintVar(&config.Port, "port", 9448, "Port to expose scraping endpoint on")
	flag.DurationVar(&config.Timeout, "timeout", time.Second*10, "Timeout when calling EventStore")
	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose logging")
	flag.StringVar(&config.ClusterMode, "cluster-mode", "cluster", "Cluster mode: `cluster` or `single`. In single mode, calls to cluster status endpoints are skipped")
	flag.BoolVar(&config.InsecureSkipVerify, "insecure-skip-verify", false, "Skip TLS certificatte verification for EventStore HTTP client")
	flag.BoolVar(&config.EnableParkedMessagesStats, "enable-parked-messages-stats", false, "Enable parked messages stats scraping")

	flag.Parse()

	return config, config.validate()
}

func (config *Config) validate() error {
	if config.ClusterMode != "cluster" && config.ClusterMode != "single" {
		return fmt.Errorf("Unknown cluster mode %v, use 'cluster' or 'single'", config.ClusterMode)
	}

	if (config.EventStoreUser != "" && config.EventStorePassword == "") || (config.EventStoreUser == "" && config.EventStorePassword != "") {
		return errors.New("EventStore user and password should both be specified, or should both be empty")
	}

	return nil
}

func (config *Config) IsInClusterMode() bool {
	return config.ClusterMode == "cluster"
}
