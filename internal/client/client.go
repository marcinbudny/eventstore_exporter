package client

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"time"

	"github.com/EventStore/EventStore-Client-Go/v2/esdb"
	"github.com/marcinbudny/eventstore_exporter/internal/config"
	log "github.com/sirupsen/logrus"
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

	url, err := url.Parse(client.config.EventStoreURL)

	if err != nil {
		return nil, err
	}

	esConfig := &esdb.Configuration{
		Address:                     url.Host,
		DisableTLS:                  url.Scheme != "https",
		SkipCertificateVerification: client.config.InsecureSkipVerify,
		DiscoveryInterval:           100,
		GossipTimeout:               5,
		MaxDiscoverAttempts:         10,
		KeepAliveInterval:           10 * time.Second,
		KeepAliveTimeout:            10 * time.Second,
	}

	if client.config.EventStoreUser != "" && client.config.EventStorePassword != "" {
		esConfig.Username = client.config.EventStoreUser
		esConfig.Password = client.config.EventStorePassword
	}

	return esdb.NewClient(esConfig)
}

func (client *EventStoreStatsClient) GetStats() (*Stats, error) {
	esVersionChan := client.getEsVersion()
	serverStatsChan := client.getServerStats()
	projectionStatsChan := client.getProjectionStats()
	subscriptionStatsChan := client.getSubscriptionStats()
	streamStatsChan := client.getStreamStats()

	var clusterStatsChan <-chan getClusterStatsResult
	if client.config.IsInClusterMode() {
		clusterStatsChan = client.getClusterStats()
	}

	esVersionResult := <-esVersionChan
	serverStatsResult := <-serverStatsChan
	projectionStatsResult := <-projectionStatsChan
	subscriptionsStatsResult := <-subscriptionStatsChan
	streamStatsResult := <-streamStatsChan

	var clusterStatsResult getClusterStatsResult
	if client.config.IsInClusterMode() {
		clusterStatsResult = <-clusterStatsChan
	}

	if esVersionResult.err != nil {
		return nil, esVersionResult.err
	}
	if serverStatsResult.err != nil {
		return nil, serverStatsResult.err
	}
	if projectionStatsResult.err != nil {
		return nil, projectionStatsResult.err
	}
	if subscriptionsStatsResult.err != nil {
		return nil, subscriptionsStatsResult.err
	}
	if streamStatsResult.err != nil {
		return nil, streamStatsResult.err
	}
	if client.config.IsInClusterMode() && clusterStatsResult.err != nil {
		return nil, clusterStatsResult.err
	}

	var stats = &Stats{
		EsVersion:     esVersionResult.esVersion,
		Process:       serverStatsResult.process,
		DiskIo:        serverStatsResult.diskIo,
		Tcp:           serverStatsResult.tcpStats,
		Queues:        serverStatsResult.queues,
		Drives:        serverStatsResult.drives,
		Projections:   projectionStatsResult.projections,
		Subscriptions: subscriptionsStatsResult.subscriptions,
		Streams:       streamStatsResult.streams,
	}

	if client.config.IsInClusterMode() {
		stats.Cluster = clusterStatsResult.cluster
	}

	return stats, nil
}
