package config

import (
	"errors"
	"fmt"
	"strings"
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
	EnableParkedMessagesStats bool
	Streams                   []string
	StreamsSeparator          string
	EnableTCPConnectionStats  bool
}

func Load(args []string, suppressOutput bool) (*Config, error) {
	config := &Config{}

	fs := flag.NewFlagSet("flagset", flag.ContinueOnError)
	fs.String(flag.DefaultConfigFlagname, "", "Path to config file")
	fs.StringVar(&config.EventStoreURL, "eventstore-url", "http://localhost:2113", "EventStore URL")
	fs.StringVar(&config.EventStoreUser, "eventstore-user", "", "EventStore User")
	fs.StringVar(&config.EventStorePassword, "eventstore-password", "", "EventStore Password")
	fs.UintVar(&config.Port, "port", 9448, "Port to expose scraping endpoint on")
	fs.DurationVar(&config.Timeout, "timeout", time.Second*10, "Timeout for the scrape operation")
	fs.BoolVar(&config.Verbose, "verbose", false, "Enable verbose logging")
	fs.BoolVar(&config.InsecureSkipVerify, "insecure-skip-verify", false, "Skip TLS certificatte verification for EventStore HTTP client")
	fs.BoolVar(&config.EnableParkedMessagesStats, "enable-parked-messages-stats", false, "Enable parked messages stats scraping")
	streamsString := fs.String("streams", "", "List of streams to get metrics for")
	fs.StringVar(&config.StreamsSeparator, "streams-separator", ",", "Separator for streams list (default: ',')")
	fs.BoolVar(&config.EnableTCPConnectionStats, "enable-tcp-connection-stats", false, "Enable TCP connection stats scraping")

	if suppressOutput {
		fs.Usage = func() {}
	}

	err := fs.Parse(args)
	if err != nil {
		return nil, err
	}

	config.Streams = parseStreamList(streamsString, config.StreamsSeparator)

	err = config.validate()
	if err != nil {
		return nil, err
	}

	return config, nil
}

func (config *Config) validate() error {
	if (config.EventStoreUser != "" && config.EventStorePassword == "") || (config.EventStoreUser == "" && config.EventStorePassword != "") {
		return errors.New("EventStore user and password should both be specified, or should both be empty")
	}

	if len(config.StreamsSeparator) != 1 {
		return fmt.Errorf("streams separator should be a single character, got %s", config.StreamsSeparator)
	}

	return nil
}

func parseStreamList(streamsString *string, streamsSeparator string) []string {
	if streamsString == nil || *streamsString == "" {
		return []string{}
	}

	return strings.Split(*streamsString, streamsSeparator)
}
