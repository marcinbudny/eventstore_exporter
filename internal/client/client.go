package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/EventStore/EventStore-Client-Go/v3/esdb"
	"github.com/marcinbudny/eventstore_exporter/internal/config"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type EventStoreStatsClient struct {
	httpClient http.Client
	config     *config.Config
}

type Stats struct {
	Info           *EsInfo
	Server         *ServerStats
	ClusterMembers []MemberStats
	Projections    []ProjectionStats
	Subscriptions  []SubscriptionStats
	Streams        []StreamStats
	TCPConnections []TCPConnectionStats
}

func New(config *config.Config) *EventStoreStatsClient {
	esClient := &EventStoreStatsClient{}
	esClient.config = config

	if config.InsecureSkipVerify {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // nolint: gosec
		}
		esClient.httpClient = http.Client{
			Transport: tr,
		}
	} else {
		esClient.httpClient = http.Client{}
	}

	return esClient
}

func (client *EventStoreStatsClient) getGrpcClient() (*esdb.Client, error) {
	log.Debug("Creating ES grpc client")

	esURL, err := url.Parse(client.config.EventStoreURL)

	if err != nil {
		return nil, err
	}

	esConfig := &esdb.Configuration{
		Address:                     esURL.Host,
		DisableTLS:                  esURL.Scheme != "https",
		SkipCertificateVerification: client.config.InsecureSkipVerify,
		DiscoveryInterval:           100,
		GossipTimeout:               5,
		MaxDiscoverAttempts:         10,
		KeepAliveInterval:           10 * time.Second,
		KeepAliveTimeout:            10 * time.Second,
		Logger:                      loggerAdapter,
	}

	if client.config.EventStoreUser != "" && client.config.EventStorePassword != "" {
		esConfig.Username = client.config.EventStoreUser
		esConfig.Password = client.config.EventStorePassword
	}

	return esdb.NewClient(esConfig)
}

func (client *EventStoreStatsClient) GetStats(ctx context.Context) (*Stats, error) {
	// TODO: support cancellation on error
	group := &errgroup.Group{}

	stats := &Stats{}

	group.Go(func() error {
		info, err := client.GetEsInfo(ctx)
		if err != nil {
			return fmt.Errorf("error while getting ES Info: %w", err)
		}

		stats.Info = info
		return nil
	})

	group.Go(func() error {
		serverStats, err := client.getServerStats(ctx)
		if err != nil {
			return fmt.Errorf("error while getting server stats: %w", err)
		}

		stats.Server = serverStats
		return nil
	})

	group.Go(func() error {
		projectionStats, err := client.getProjectionStats(ctx)
		if err != nil {
			return fmt.Errorf("error while getting projection stats: %w", err)
		}

		stats.Projections = projectionStats
		return nil
	})

	group.Go(func() error {
		subscriptionStats, err := client.getSubscriptionStats(ctx)
		if err != nil {
			return fmt.Errorf("error while getting subscription stats: %w", err)
		}

		stats.Subscriptions = subscriptionStats
		return nil
	})

	group.Go(func() error {
		streamStats, err := client.getStreamStats(ctx)
		if err != nil {
			return fmt.Errorf("error while getting stream stats: %w", err)
		}

		stats.Streams = streamStats
		return nil
	})

	group.Go(func() error {
		clusterStats, err := client.getClusterStats(ctx)
		if err != nil {
			return fmt.Errorf("error while getting cluster stats: %w", err)
		}

		stats.ClusterMembers = clusterStats
		return nil
	})

	group.Go(func() error {
		tcpConnectionStats, err := client.getTCPConnectionStats(ctx)
		if err != nil {
			return fmt.Errorf("error while getting tcp connection stats: %w", err)
		}

		stats.TCPConnections = tcpConnectionStats
		return nil
	})

	if err := group.Wait(); err != nil {
		return nil, err
	}

	return stats, nil
}

func loggerAdapter(level esdb.LogLevel, format string, args ...interface{}) {
	mappedLevel := log.InfoLevel

	switch level {
	case "debug":
		mappedLevel = log.DebugLevel
	case "warn":
		mappedLevel = log.WarnLevel
	case "error":
		mappedLevel = log.ErrorLevel
	}

	if log.IsLevelEnabled(mappedLevel) {
		log.StandardLogger().WithField("context", "esdb_client").Log(mappedLevel, fmt.Sprintf(format, args...))
	}
}
