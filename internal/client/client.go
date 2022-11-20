package client

import (
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
	EsVersion     EventStoreVersion
	Process       *ProcessStats
	DiskIo        *DiskIoStats
	Tcp           *TcpStats
	Cluster       *ClusterStats
	Queues        []QueueStats
	Drives        []DriveStats
	Projections   []ProjectionStats
	Subscriptions []SubscriptionStats
	Streams       []StreamStats
}

func New(config *config.Config) *EventStoreStatsClient {
	esClient := &EventStoreStatsClient{}
	esClient.config = config

	if config.InsecureSkipVerify {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		esClient.httpClient = http.Client{
			Timeout:   config.Timeout,
			Transport: tr,
		}
	} else {
		esClient.httpClient = http.Client{
			Timeout: config.Timeout,
		}
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

func (client *EventStoreStatsClient) GetStats() (*Stats, error) {
	// TODO: support cancellation on error
	group := &errgroup.Group{}

	stats := &Stats{}

	group.Go(func() error {
		if esVersion, err := client.getEsVersion(); err != nil {
			return err
		} else {
			stats.EsVersion = esVersion
			return nil
		}
	})

	group.Go(func() error {
		if serverStats, err := client.getServerStats(); err != nil {
			return err
		} else {
			stats.Process = serverStats.process
			stats.DiskIo = serverStats.diskIo
			stats.Tcp = serverStats.tcpStats
			stats.Queues = serverStats.queues
			stats.Drives = serverStats.drives
			return nil
		}
	})

	group.Go(func() error {
		if projectionStats, err := client.getProjectionStats(); err != nil {
			return err
		} else {
			stats.Projections = projectionStats
			return nil
		}
	})

	group.Go(func() error {
		if subscriptionStats, err := client.getSubscriptionStats(); err != nil {
			return err
		} else {
			stats.Subscriptions = subscriptionStats
			return nil
		}
	})

	group.Go(func() error {
		if streamStats, err := client.getStreamStats(); err != nil {
			return err
		} else {
			stats.Streams = streamStats
			return nil
		}
	})

	group.Go(func() error {
		if clusterStats, err := client.getClusterStats(); err != nil {
			return err
		} else {
			stats.Cluster = clusterStats
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
