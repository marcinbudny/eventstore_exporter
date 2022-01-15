package client

import (
	"crypto/tls"
	"net/http"

	"github.com/marcinbudny/eventstore_exporter/internal/config"
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

func (client *EventStoreStatsClient) GetStats() (*Stats, error) {

	esVersionResult := <-client.getEsVersion()
	if esVersionResult.err != nil {
		return nil, esVersionResult.err
	}

	serverStatsChan := client.getServerStats()
	projectionStatsChan := client.getProjectionStats()
	subscriptionStatsChan := client.getSubscriptionStats(esVersionResult.esVersion)

	var clusterStatsChan <-chan getClusterStatsResult
	if client.config.IsInClusterMode() {
		clusterStatsChan = client.getClusterStats()
	}

	serverStatsResult := <-serverStatsChan
	projectionStatsResult := <-projectionStatsChan
	subscriptionsStatsResult := <-subscriptionStatsChan

	var clusterStatsResult getClusterStatsResult
	if client.config.IsInClusterMode() {
		clusterStatsResult = <-clusterStatsChan
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
	}

	if client.config.IsInClusterMode() {
		stats.Cluster = clusterStatsResult.cluster
	}

	return stats, nil
}
