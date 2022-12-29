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
	TcpConnections []TcpConnectionStats
}

func New(config *config.Config) *EventStoreStatsClient {
	esClient := &EventStoreStatsClient{}
	esClient.config = config

	if config.InsecureSkipVerify {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
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

	esUrl, err := url.Parse(client.config.EventStoreURL)

	if err != nil {
		return nil, err
	}

	esConfig := &esdb.Configuration{
		Address:                     esUrl.Host,
		DisableTLS:                  esUrl.Scheme != "https",
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
		if info, err := client.GetEsInfo(ctx); err != nil {
			return fmt.Errorf("error while getting ES Info: %w", err)
		} else {
			stats.Info = info
			return nil
		}
	})

	group.Go(func() error {
		if serverStats, err := client.getServerStats(ctx); err != nil {
			return fmt.Errorf("error while getting server stats: %w", err)
		} else {
			stats.Server = serverStats
			return nil
		}
	})

	group.Go(func() error {
		if projectionStats, err := client.getProjectionStats(ctx); err != nil {
			return fmt.Errorf("error while getting projection stats: %w", err)
		} else {
			stats.Projections = projectionStats
			return nil
		}
	})

	group.Go(func() error {
		if subscriptionStats, err := client.getSubscriptionStats(ctx); err != nil {
			return fmt.Errorf("error while getting subscription stats: %w", err)
		} else {
			stats.Subscriptions = subscriptionStats
			return nil
		}
	})

	group.Go(func() error {
		if streamStats, err := client.getStreamStats(ctx); err != nil {
			return fmt.Errorf("error while getting stream stats: %w", err)
		} else {
			stats.Streams = streamStats
			return nil
		}
	})

	group.Go(func() error {
		if clusterStats, err := client.getClusterStats(ctx); err != nil {
			return fmt.Errorf("error while getting cluster stats: %w", err)
		} else {
			stats.ClusterMembers = clusterStats
			return nil
		}
	})

	group.Go(func() error {
		if tcpConnectionStats, err := client.getTcpConnectionStats(ctx); err != nil {
			return fmt.Errorf("error while getting tcp connection stats: %w", err)
		} else {
			stats.TcpConnections = tcpConnectionStats
			return nil
		}
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
