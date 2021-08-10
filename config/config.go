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

func createFlagSet(config *Config) *flag.FlagSet {
	fs := flag.NewFlagSet("flagset", flag.ContinueOnError)
	fs.String(flag.DefaultConfigFlagname, "", "Path to config file")
	fs.StringVar(&config.EventStoreURL, "eventstore-url", "http://localhost:2113", "EventStore URL")
	fs.StringVar(&config.EventStoreUser, "eventstore-user", "", "EventStore User")
	fs.StringVar(&config.EventStorePassword, "eventstore-password", "", "EventStore Password")
	fs.UintVar(&config.Port, "port", 9448, "Port to expose scraping endpoint on")
	fs.DurationVar(&config.Timeout, "timeout", time.Second*10, "Timeout when calling EventStore")
	fs.BoolVar(&config.Verbose, "verbose", false, "Enable verbose logging")
	fs.StringVar(&config.ClusterMode, "cluster-mode", "cluster", "Cluster mode: `cluster` or `single`. In single mode, calls to cluster status endpoints are skipped")
	fs.BoolVar(&config.InsecureSkipVerify, "insecure-skip-verify", false, "Skip TLS certificatte verification for EventStore HTTP client")
	fs.BoolVar(&config.EnableParkedMessagesStats, "enable-parked-messages-stats", false, "Enable parked messages stats scraping")

	return fs
}

func Load(args []string, suppressOutput bool) (*Config, error) {
	config := &Config{}
	fs := createFlagSet(config)

	if suppressOutput {
		fs.Usage = func() {}
	}

	if err := fs.Parse(args); err == nil {
		if validationErr := config.validate(); validationErr == nil {
			return config, nil
		} else {
			return nil, validationErr
		}
	} else {
		return nil, err
	}
}

func (config *Config) validate() error {
	if config.ClusterMode != "cluster" && config.ClusterMode != "single" {
		return fmt.Errorf("unknown cluster mode %v, use 'cluster' or 'single'", config.ClusterMode)
	}

	if (config.EventStoreUser != "" && config.EventStorePassword == "") || (config.EventStoreUser == "" && config.EventStorePassword != "") {
		return errors.New("EventStore user and password should both be specified, or should both be empty")
	}

	return nil
}

func (config *Config) IsInClusterMode() bool {
	return config.ClusterMode == "cluster"
}
