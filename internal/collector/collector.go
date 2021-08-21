package collector

import (
	"fmt"

	jp "github.com/buger/jsonparser"
	"github.com/marcinbudny/eventstore_exporter/internal/client"
	"github.com/marcinbudny/eventstore_exporter/internal/config"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

// Collector struct
type Collector struct {
	config *config.Config
	client *client.EventStoreStatsClient

	up                 *prometheus.Desc
	processCPU         *prometheus.Desc
	processCPUScaled   *prometheus.Desc
	processMemoryBytes *prometheus.Desc
	diskIoReadBytes    *prometheus.Desc
	diskIoWrittenBytes *prometheus.Desc
	diskIoReadOps      *prometheus.Desc
	diskIoWriteOps     *prometheus.Desc
	uptimeSeconds      *prometheus.Desc
	tcpSentBytes       *prometheus.Desc
	tcpReceivedBytes   *prometheus.Desc
	tcpConnections     *prometheus.Desc

	queueLength         *prometheus.Desc
	queueItemsProcessed *prometheus.Desc

	driveTotalBytes     *prometheus.Desc
	driveAvailableBytes *prometheus.Desc

	projectionRunning                     *prometheus.Desc
	projectionProgress                    *prometheus.Desc
	projectionEventsProcessedAfterRestart *prometheus.Desc

	clusterMemberAlive             *prometheus.Desc
	clusterMemberIsMaster          *prometheus.Desc
	clusterMemberIsSlave           *prometheus.Desc
	clusterMemberIsClone           *prometheus.Desc
	clusterMemberIsLeader          *prometheus.Desc
	clusterMemberIsFollower        *prometheus.Desc
	clusterMemberIsReadonlyReplica *prometheus.Desc

	subscriptionTotalItemsProcessed         *prometheus.Desc
	subscriptionLastProcessedEventNumber    *prometheus.Desc
	subscriptionLastKnownEventNumber        *prometheus.Desc
	subscriptionConnectionCount             *prometheus.Desc
	subscriptionTotalInFlightMessages       *prometheus.Desc
	subscriptionTotalNumberOfParkedMessages *prometheus.Desc
	subscriptionOldestParkedMessage         *prometheus.Desc
}

// NewCollector function
func NewCollector(config *config.Config, client *client.EventStoreStatsClient) *Collector {
	return &Collector{
		config: config,
		client: client,

		up:                 prometheus.NewDesc("eventstore_up", "Whether the EventStore scrape was successful", nil, nil),
		processCPU:         prometheus.NewDesc("eventstore_process_cpu", "Process CPU usage, 0 - number of cores", nil, nil),
		processCPUScaled:   prometheus.NewDesc("eventstore_process_cpu_scaled", "Process CPU usage scaled to number of cores, 0 - 1, 1 = full load on all cores (available only on versions < 20.6)", nil, nil),
		processMemoryBytes: prometheus.NewDesc("eventstore_process_memory_bytes", "Process memory usage, as reported by EventStore", nil, nil),
		diskIoReadBytes:    prometheus.NewDesc("eventstore_disk_io_read_bytes", "Total number of disk IO read bytes", nil, nil),
		diskIoWrittenBytes: prometheus.NewDesc("eventstore_disk_io_written_bytes", "Total number of disk IO written bytes", nil, nil),
		diskIoReadOps:      prometheus.NewDesc("eventstore_disk_io_read_ops", "Total number of disk IO read operations", nil, nil),
		diskIoWriteOps:     prometheus.NewDesc("eventstore_disk_io_write_ops", "Total number of disk IO write operations", nil, nil),
		uptimeSeconds:      prometheus.NewDesc("eventstore_uptime_seconds", "Total uptime seconds", nil, nil),
		tcpSentBytes:       prometheus.NewDesc("eventstore_tcp_sent_bytes", "TCP sent bytes", nil, nil),
		tcpReceivedBytes:   prometheus.NewDesc("eventstore_tcp_received_bytes", "TCP received bytes", nil, nil),
		tcpConnections:     prometheus.NewDesc("eventstore_tcp_connections", "Current number of TCP connections", nil, nil),

		queueLength:         prometheus.NewDesc("eventstore_queue_length", "Queue length", []string{"queue"}, nil),
		queueItemsProcessed: prometheus.NewDesc("eventstore_queue_items_processed_total", "Total number items processed by queue", []string{"queue"}, nil),

		driveTotalBytes:     prometheus.NewDesc("eventstore_drive_total_bytes", "Drive total size in bytes", []string{"drive"}, nil),
		driveAvailableBytes: prometheus.NewDesc("eventstore_drive_available_bytes", "Drive available bytes", []string{"drive"}, nil),

		projectionRunning:                     prometheus.NewDesc("eventstore_projection_running", "If 1, projection is in 'Running' state", []string{"projection"}, nil),
		projectionProgress:                    prometheus.NewDesc("eventstore_projection_progress", "Projection progress 0 - 1, where 1 = projection progress at 100%", []string{"projection"}, nil),
		projectionEventsProcessedAfterRestart: prometheus.NewDesc("eventstore_projection_events_processed_after_restart_total", "Projection event processed count after restart", []string{"projection"}, nil),

		clusterMemberAlive:             prometheus.NewDesc("eventstore_cluster_member_alive", "If 1, cluster member is alive, as seen from current cluster member", []string{"member"}, nil),
		clusterMemberIsMaster:          prometheus.NewDesc("eventstore_cluster_member_is_master", "If 1, current cluster member is the master (only versions < 20.6)", nil, nil),
		clusterMemberIsSlave:           prometheus.NewDesc("eventstore_cluster_member_is_slave", "If 1, current cluster member is a slave (only versions < 20.6)", nil, nil),
		clusterMemberIsClone:           prometheus.NewDesc("eventstore_cluster_member_is_clone", "If 1, current cluster member is a clone", nil, nil),
		clusterMemberIsLeader:          prometheus.NewDesc("eventstore_cluster_member_is_leader", "If 1, current cluster member is the leader (only versions >= 20.6)", nil, nil),
		clusterMemberIsFollower:        prometheus.NewDesc("eventstore_cluster_member_is_follower", "If 1, current cluster member is a follower (only versions >= 20.6)", nil, nil),
		clusterMemberIsReadonlyReplica: prometheus.NewDesc("eventstore_cluster_member_is_readonly_replica", "If 1, current cluster member is a readonly replica (only versions >= 20.6)", nil, nil),

		subscriptionTotalItemsProcessed:         prometheus.NewDesc("eventstore_subscription_items_processed_total", "Total items processed by subscription", []string{"event_stream_id", "group_name"}, nil),
		subscriptionLastProcessedEventNumber:    prometheus.NewDesc("eventstore_subscription_last_processed_event_number", "Last event number processed by subscription", []string{"event_stream_id", "group_name"}, nil),
		subscriptionLastKnownEventNumber:        prometheus.NewDesc("eventstore_subscription_last_known_event_number", "Last known event number in subscription", []string{"event_stream_id", "group_name"}, nil),
		subscriptionConnectionCount:             prometheus.NewDesc("eventstore_subscription_connections", "Number of connections to subscription", []string{"event_stream_id", "group_name"}, nil),
		subscriptionTotalInFlightMessages:       prometheus.NewDesc("eventstore_subscription_messages_in_flight", "Number of messages in flight for subscription", []string{"event_stream_id", "group_name"}, nil),
		subscriptionTotalNumberOfParkedMessages: prometheus.NewDesc("eventstore_subscription_parked_messages", "Number of parked messages for subscription", []string{"event_stream_id", "group_name"}, nil),
		subscriptionOldestParkedMessage:         prometheus.NewDesc("eventstore_subscription_oldest_parked_message_age_seconds", "Oldest parked message age for subscription in seconds", []string{"event_stream_id", "group_name"}, nil),
	}
}

// Describe function
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.up
	ch <- c.processCPU
	ch <- c.processCPUScaled
	ch <- c.processMemoryBytes
	ch <- c.diskIoReadBytes
	ch <- c.diskIoWrittenBytes
	ch <- c.diskIoReadOps
	ch <- c.diskIoWriteOps
	ch <- c.uptimeSeconds
	ch <- c.tcpSentBytes
	ch <- c.tcpReceivedBytes
	ch <- c.tcpConnections

	ch <- c.queueLength
	ch <- c.queueItemsProcessed

	ch <- c.driveTotalBytes
	ch <- c.driveAvailableBytes

	ch <- c.projectionRunning
	ch <- c.projectionProgress
	ch <- c.projectionEventsProcessedAfterRestart

	if c.config.IsInClusterMode() {
		ch <- c.clusterMemberAlive
		ch <- c.clusterMemberIsMaster
		ch <- c.clusterMemberIsSlave
		ch <- c.clusterMemberIsClone
		ch <- c.clusterMemberIsLeader
		ch <- c.clusterMemberIsFollower
		ch <- c.clusterMemberIsReadonlyReplica
	}

	ch <- c.subscriptionTotalItemsProcessed
	ch <- c.subscriptionLastProcessedEventNumber
	ch <- c.subscriptionLastKnownEventNumber
	ch <- c.subscriptionConnectionCount
	ch <- c.subscriptionTotalInFlightMessages
	ch <- c.subscriptionTotalNumberOfParkedMessages
	ch <- c.subscriptionOldestParkedMessage
}

// Collect function
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	log.Info("Running scrape")

	if stats, err := c.client.GetStats(); err != nil {
		log.WithError(err).Error("Error while getting data from EventStore")

		ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 0)
	} else {
		ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 1)

		ch <- prometheus.MustNewConstMetric(c.processCPU, prometheus.GaugeValue, getProcessCPU(stats))

		if stats.EsVersion.ReportsCpuScaled() {
			ch <- prometheus.MustNewConstMetric(c.processCPUScaled, prometheus.GaugeValue, getProcessCPUScaled(stats))
		}

		ch <- prometheus.MustNewConstMetric(c.processMemoryBytes, prometheus.GaugeValue, getProcessMemory(stats))
		ch <- prometheus.MustNewConstMetric(c.diskIoReadBytes, prometheus.GaugeValue, getDiskIoReadBytes(stats))
		ch <- prometheus.MustNewConstMetric(c.diskIoWrittenBytes, prometheus.GaugeValue, getDiskIoWrittenBytes(stats))
		ch <- prometheus.MustNewConstMetric(c.diskIoReadOps, prometheus.GaugeValue, getDiskIoReadOps(stats))
		ch <- prometheus.MustNewConstMetric(c.diskIoWriteOps, prometheus.GaugeValue, getDiskIoWriteOps(stats))
		ch <- prometheus.MustNewConstMetric(c.tcpSentBytes, prometheus.GaugeValue, getTCPSentBytes(stats))
		ch <- prometheus.MustNewConstMetric(c.tcpReceivedBytes, prometheus.GaugeValue, getTCPReceivedBytes(stats))
		ch <- prometheus.MustNewConstMetric(c.tcpConnections, prometheus.GaugeValue, getTCPConnections(stats))

		if stats.EsVersion.UsesLeaderFollowerNomenclature() {
			ch <- prometheus.MustNewConstMetric(c.clusterMemberIsLeader, prometheus.GaugeValue, getIs("leader", stats))
			ch <- prometheus.MustNewConstMetric(c.clusterMemberIsFollower, prometheus.GaugeValue, getIs("follower", stats))
			ch <- prometheus.MustNewConstMetric(c.clusterMemberIsReadonlyReplica, prometheus.GaugeValue, getIs("readonlyreplica", stats))
		} else {
			ch <- prometheus.MustNewConstMetric(c.clusterMemberIsMaster, prometheus.GaugeValue, getIs("master", stats))
			ch <- prometheus.MustNewConstMetric(c.clusterMemberIsSlave, prometheus.GaugeValue, getIs("slave", stats))
		}
		ch <- prometheus.MustNewConstMetric(c.clusterMemberIsClone, prometheus.GaugeValue, getIs("clone", stats))

		collectPerQueueMetric(stats, c.queueLength, getQueueLength, ch)
		collectPerQueueMetric(stats, c.queueItemsProcessed, getQueueItemsProcessed, ch)

		collectPerDriveMetric(stats, c.driveTotalBytes, getDriveTotalBytes, ch)
		collectPerDriveMetric(stats, c.driveAvailableBytes, getDriveAvailableBytes, ch)

		collectPerProjectionMetric(stats, c.projectionRunning, getProjectionIsRunning, ch)
		collectPerProjectionMetric(stats, c.projectionProgress, getProjectionProgress, ch)
		collectPerProjectionMetric(stats, c.projectionEventsProcessedAfterRestart, getProjectionEventsProcessedAfterRestart, ch)

		collectPerSubscriptionMetric(stats, c.subscriptionTotalItemsProcessed, getSubscriptionTotalItemsProcessed, ch)
		collectPerSubscriptionMetric(stats, c.subscriptionConnectionCount, getSubscriptionConnectionCount, ch)
		collectPerSubscriptionMetric(stats, c.subscriptionLastKnownEventNumber, getSubscriptionLastKnownEventNumber, ch)
		collectPerSubscriptionMetric(stats, c.subscriptionLastProcessedEventNumber, getSubscriptionLastProcessedEventNumber, ch)
		collectPerSubscriptionMetric(stats, c.subscriptionTotalInFlightMessages, getSubscriptionTotalInFlightMessages, ch)
		collectParkedMessagesPerSubscriptionMetric(stats.ParkedMessagesStats, c.subscriptionTotalNumberOfParkedMessages, ch)
		collectOldestParkedMessagePerSubscriptionMetric(stats.ParkedMessagesStats, c.subscriptionOldestParkedMessage, ch)

		if c.config.IsInClusterMode() {
			collectPerMemberMetric(stats, c.clusterMemberAlive, getMemberIsAlive, ch)
		}
	}
}

func collectPerMemberMetric(stats *client.Stats, desc *prometheus.Desc, collectFunc func([]byte) (prometheus.ValueType, float64), ch chan<- prometheus.Metric) {

	httpEndPointNomenclature := stats.EsVersion.UsesHttpEndPointNomenclature()

	jp.ArrayEach(stats.GossipStats, func(jsonValue []byte, dataType jp.ValueType, offset int, err error) {
		ip := ""
		port := int64(0)
		if httpEndPointNomenclature {
			ip, _ = jp.GetString(jsonValue, "httpEndPointIp")
			port, _ = jp.GetInt(jsonValue, "httpEndPointPort")

		} else {
			ip, _ = jp.GetString(jsonValue, "externalHttpIp")
			port, _ = jp.GetInt(jsonValue, "externalHttpPort")

		}

		memberName := fmt.Sprintf("%s:%d", ip, port)
		valueType, value := collectFunc(jsonValue)
		ch <- prometheus.MustNewConstMetric(desc, valueType, value, memberName)
	}, "members")

}

func getMemberIsAlive(member []byte) (prometheus.ValueType, float64) {
	alive, _ := jp.GetBoolean(member, "isAlive")
	if alive {
		return prometheus.GaugeValue, 1
	}
	return prometheus.GaugeValue, 0
}

func collectPerProjectionMetric(stats *client.Stats, desc *prometheus.Desc, collectFunc func([]byte) (prometheus.ValueType, float64), ch chan<- prometheus.Metric) {
	jp.ArrayEach(stats.ProjectionStats, func(jsonValue []byte, dataType jp.ValueType, offset int, err error) {
		projectionName, _ := jp.GetString(jsonValue, "effectiveName")
		valueType, value := collectFunc(jsonValue)
		ch <- prometheus.MustNewConstMetric(desc, valueType, value, projectionName)
	}, "projections")
}

func getProjectionIsRunning(projection []byte) (prometheus.ValueType, float64) {
	status, _ := jp.GetString(projection, "status")
	if status == "Running" {
		return prometheus.GaugeValue, 1
	}
	return prometheus.GaugeValue, 0
}

func getProjectionProgress(projection []byte) (prometheus.ValueType, float64) {
	progress, _ := jp.GetFloat(projection, "progress")
	return prometheus.GaugeValue, progress / 100.0 // scale to 0-1
}

func getProjectionEventsProcessedAfterRestart(projection []byte) (prometheus.ValueType, float64) {
	processed, _ := jp.GetFloat(projection, "eventsProcessedAfterRestart")
	return prometheus.CounterValue, processed
}

func collectPerQueueMetric(stats *client.Stats, desc *prometheus.Desc, collectFunc func([]byte) (prometheus.ValueType, float64), ch chan<- prometheus.Metric) {
	jp.ObjectEach(stats.ServerStats, func(key []byte, jsonValue []byte, dataType jp.ValueType, offset int) error {
		queueName := string(key)
		valueType, value := collectFunc(jsonValue)
		ch <- prometheus.MustNewConstMetric(desc, valueType, value, queueName)
		return nil
	}, "es", "queue")
}

func getQueueLength(queue []byte) (prometheus.ValueType, float64) {
	value, _ := jp.GetFloat(queue, "length")
	return prometheus.GaugeValue, value
}

func getQueueItemsProcessed(queue []byte) (prometheus.ValueType, float64) {
	value, _ := jp.GetFloat(queue, "totalItemsProcessed")
	return prometheus.CounterValue, value
}

func collectPerDriveMetric(stats *client.Stats, desc *prometheus.Desc, collectFunc func([]byte) (prometheus.ValueType, float64), ch chan<- prometheus.Metric) {
	jp.ObjectEach(stats.ServerStats, func(key []byte, jsonValue []byte, dataType jp.ValueType, offset int) error {
		drive := string(key)
		valueType, value := collectFunc(jsonValue)
		ch <- prometheus.MustNewConstMetric(desc, valueType, value, drive)
		return nil
	}, "sys", "drive")

}

func getDriveTotalBytes(drive []byte) (prometheus.ValueType, float64) {
	value, _ := jp.GetFloat(drive, "totalBytes")
	return prometheus.GaugeValue, value
}

func getDriveAvailableBytes(drive []byte) (prometheus.ValueType, float64) {
	value, _ := jp.GetFloat(drive, "availableBytes")
	return prometheus.GaugeValue, value
}

func collectPerSubscriptionMetric(stats *client.Stats, desc *prometheus.Desc, collectFunc func([]byte) (prometheus.ValueType, float64), ch chan<- prometheus.Metric) {
	jp.ArrayEach(stats.SubscriptionsStats, func(jsonValue []byte, dataType jp.ValueType, offset int, err error) {
		eventStreamID, _ := jp.GetString(jsonValue, "eventStreamId")
		groupName, _ := jp.GetString(jsonValue, "groupName")
		valueType, value := collectFunc(jsonValue)
		ch <- prometheus.MustNewConstMetric(desc, valueType, value, eventStreamID, groupName)
	})
}

func collectParkedMessagesPerSubscriptionMetric(stats []client.ParkedMessagesStats, desc *prometheus.Desc, ch chan<- prometheus.Metric) {
	for _, stat := range stats {
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, stat.TotalNumberOfParkedMessages, stat.EventStreamID, stat.GroupName)
	}
}

func collectOldestParkedMessagePerSubscriptionMetric(stats []client.ParkedMessagesStats, desc *prometheus.Desc, ch chan<- prometheus.Metric) {
	for _, stat := range stats {
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, stat.OldestParkedMessageAgeInSeconds, stat.EventStreamID, stat.GroupName)
	}
}

func getSubscriptionTotalItemsProcessed(subscription []byte) (prometheus.ValueType, float64) {
	value, _ := jp.GetFloat(subscription, "totalItemsProcessed")
	return prometheus.CounterValue, value
}

func getSubscriptionConnectionCount(subscription []byte) (prometheus.ValueType, float64) {
	value, _ := jp.GetFloat(subscription, "connectionCount")
	return prometheus.GaugeValue, value
}

func getSubscriptionLastProcessedEventNumber(subscription []byte) (prometheus.ValueType, float64) {
	value, _ := jp.GetFloat(subscription, "lastProcessedEventNumber")
	return prometheus.GaugeValue, value
}

func getSubscriptionLastKnownEventNumber(subscription []byte) (prometheus.ValueType, float64) {
	value, _ := jp.GetFloat(subscription, "lastKnownEventNumber")
	return prometheus.GaugeValue, value
}

func getSubscriptionTotalInFlightMessages(subscription []byte) (prometheus.ValueType, float64) {
	value, _ := jp.GetFloat(subscription, "totalInFlightMessages")
	return prometheus.GaugeValue, value
}

func getProcessCPU(stats *client.Stats) float64 {
	value, _ := jp.GetFloat(stats.ServerStats, "proc", "cpu")
	return value / 100.0
}

func getProcessCPUScaled(stats *client.Stats) float64 {
	value, _ := jp.GetFloat(stats.ServerStats, "proc", "cpuScaled")
	return value / 100.0
}

func getProcessMemory(stats *client.Stats) float64 {
	value, _ := jp.GetFloat(stats.ServerStats, "proc", "mem")
	return value
}

func getDiskIoReadBytes(stats *client.Stats) float64 {
	value, _ := jp.GetFloat(stats.ServerStats, "proc", "diskIo", "readBytes")
	return value
}

func getDiskIoWrittenBytes(stats *client.Stats) float64 {
	value, _ := jp.GetFloat(stats.ServerStats, "proc", "diskIo", "writtenBytes")
	return value
}

func getDiskIoReadOps(stats *client.Stats) float64 {
	value, _ := jp.GetFloat(stats.ServerStats, "proc", "diskIo", "readOps")
	return value
}

func getDiskIoWriteOps(stats *client.Stats) float64 {
	value, _ := jp.GetFloat(stats.ServerStats, "proc", "diskIo", "writeOps")
	return value
}

func getTCPSentBytes(stats *client.Stats) float64 {
	value, _ := jp.GetFloat(stats.ServerStats, "proc", "tcp", "sentBytesTotal")
	return value
}

func getTCPReceivedBytes(stats *client.Stats) float64 {
	value, _ := jp.GetFloat(stats.ServerStats, "proc", "tcp", "receivedBytesTotal")
	return value
}

func getTCPConnections(stats *client.Stats) float64 {
	value, _ := jp.GetFloat(stats.ServerStats, "proc", "tcp", "connections")
	return value
}

func getIs(status string, stats *client.Stats) float64 {
	value, _ := jp.GetString(stats.Info, "state")
	if value == status {
		return 1
	}
	return 0
}
