package collector

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/marcinbudny/eventstore_exporter/internal/client"
	"github.com/marcinbudny/eventstore_exporter/internal/config"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

type Collector struct {
	config *config.Config
	client *client.EventStoreStatsClient

	up                 *prometheus.Desc
	processCPU         *prometheus.Desc
	processMemoryBytes *prometheus.Desc
	diskIoReadBytes    *prometheus.Desc
	diskIoWrittenBytes *prometheus.Desc
	diskIoReadOps      *prometheus.Desc
	diskIoWriteOps     *prometheus.Desc
	uptimeSeconds      *prometheus.Desc
	tcpSentBytes       *prometheus.Desc
	tcpReceivedBytes   *prometheus.Desc
	tcpConnections     *prometheus.Desc

	tcpConnectionSentBytes            *prometheus.Desc
	tcpConnectionReceivedBytes        *prometheus.Desc
	tcpConnectionPendingSendBytes     *prometheus.Desc
	tcpConnectionPendingReceivedBytes *prometheus.Desc

	queueLength         *prometheus.Desc
	queueItemsProcessed *prometheus.Desc

	driveTotalBytes     *prometheus.Desc
	driveAvailableBytes *prometheus.Desc

	projectionRunning                     *prometheus.Desc
	projectionStatus                      *prometheus.Desc
	projectionProgress                    *prometheus.Desc
	projectionEventsProcessedAfterRestart *prometheus.Desc

	clusterMemberAlive             *prometheus.Desc
	clusterMemberIsClone           *prometheus.Desc
	clusterMemberIsLeader          *prometheus.Desc
	clusterMemberIsFollower        *prometheus.Desc
	clusterMemberIsReadonlyReplica *prometheus.Desc

	subscriptionTotalItemsProcessed                 *prometheus.Desc
	subscriptionLastProcessedEventNumber            *prometheus.Desc
	subscriptionLastKnownEventNumber                *prometheus.Desc
	subscriptionLastCheckpointedEventCommitPosition *prometheus.Desc
	subscriptionLastKnownEventCommitPosition        *prometheus.Desc
	subscriptionConnectionCount                     *prometheus.Desc
	subscriptionTotalInFlightMessages               *prometheus.Desc
	subscriptionTotalNumberOfParkedMessages         *prometheus.Desc
	subscriptionOldestParkedMessage                 *prometheus.Desc

	streamLastCommitPosition *prometheus.Desc
	streamLastEventNumber    *prometheus.Desc
}

func NewCollector(config *config.Config, client *client.EventStoreStatsClient) *Collector {
	return &Collector{
		config: config,
		client: client,

		up:                 prometheus.NewDesc("eventstore_up", "Whether the EventStore scrape was successful", nil, nil),
		processCPU:         prometheus.NewDesc("eventstore_process_cpu", "Process CPU usage, 0 - number of cores", nil, nil),
		processMemoryBytes: prometheus.NewDesc("eventstore_process_memory_bytes", "Process memory usage, as reported by EventStore", nil, nil),
		diskIoReadBytes:    prometheus.NewDesc("eventstore_disk_io_read_bytes", "Total number of disk IO read bytes", nil, nil),
		diskIoWrittenBytes: prometheus.NewDesc("eventstore_disk_io_written_bytes", "Total number of disk IO written bytes", nil, nil),
		diskIoReadOps:      prometheus.NewDesc("eventstore_disk_io_read_ops", "Total number of disk IO read operations", nil, nil),
		diskIoWriteOps:     prometheus.NewDesc("eventstore_disk_io_write_ops", "Total number of disk IO write operations", nil, nil),
		uptimeSeconds:      prometheus.NewDesc("eventstore_uptime_seconds", "Total uptime seconds", nil, nil),
		tcpSentBytes:       prometheus.NewDesc("eventstore_tcp_sent_bytes", "TCP sent bytes", nil, nil),
		tcpReceivedBytes:   prometheus.NewDesc("eventstore_tcp_received_bytes", "TCP received bytes", nil, nil),
		tcpConnections:     prometheus.NewDesc("eventstore_tcp_connections", "Current number of TCP connections", nil, nil),

		tcpConnectionSentBytes:            prometheus.NewDesc("eventstore_tcp_connection_sent_bytes", "TCP connection total sent bytes", []string{"id", "client_name", "remote_endpoint", "local_endpoint", "external", "ssl"}, nil),
		tcpConnectionReceivedBytes:        prometheus.NewDesc("eventstore_tcp_connection_received_bytes", "TCP connection total received bytes", []string{"id", "client_name", "remote_endpoint", "local_endpoint", "external", "ssl"}, nil),
		tcpConnectionPendingSendBytes:     prometheus.NewDesc("eventstore_tcp_connection_pending_send_bytes", "TCP connection pending send bytes", []string{"id", "client_name", "remote_endpoint", "local_endpoint", "external", "ssl"}, nil),
		tcpConnectionPendingReceivedBytes: prometheus.NewDesc("eventstore_tcp_connection_pending_received_bytes", "TCP connection pending received bytes", []string{"id", "client_name", "remote_endpoint", "local_endpoint", "external", "ssl"}, nil),

		queueLength:         prometheus.NewDesc("eventstore_queue_length", "Queue length", []string{"queue"}, nil),
		queueItemsProcessed: prometheus.NewDesc("eventstore_queue_items_processed_total", "Total number items processed by queue", []string{"queue"}, nil),

		driveTotalBytes:     prometheus.NewDesc("eventstore_drive_total_bytes", "Drive total size in bytes", []string{"drive"}, nil),
		driveAvailableBytes: prometheus.NewDesc("eventstore_drive_available_bytes", "Drive available bytes", []string{"drive"}, nil),

		projectionRunning:                     prometheus.NewDesc("eventstore_projection_running", "If 1, projection is in 'Running' state", []string{"projection"}, nil),
		projectionStatus:                      prometheus.NewDesc("eventstore_projection_status", "If 1, projection is in specified state", []string{"projection", "status"}, nil),
		projectionProgress:                    prometheus.NewDesc("eventstore_projection_progress", "Projection progress 0 - 1, where 1 = projection progress at 100%", []string{"projection"}, nil),
		projectionEventsProcessedAfterRestart: prometheus.NewDesc("eventstore_projection_events_processed_after_restart_total", "Projection event processed count after restart", []string{"projection"}, nil),

		clusterMemberAlive:             prometheus.NewDesc("eventstore_cluster_member_alive", "If 1, cluster member is alive, as seen from current cluster member", []string{"member"}, nil),
		clusterMemberIsClone:           prometheus.NewDesc("eventstore_cluster_member_is_clone", "If 1, current cluster member is a clone", nil, nil),
		clusterMemberIsLeader:          prometheus.NewDesc("eventstore_cluster_member_is_leader", "If 1, current cluster member is the leader", nil, nil),
		clusterMemberIsFollower:        prometheus.NewDesc("eventstore_cluster_member_is_follower", "If 1, current cluster member is a follower", nil, nil),
		clusterMemberIsReadonlyReplica: prometheus.NewDesc("eventstore_cluster_member_is_readonly_replica", "If 1, current cluster member is a readonly replica", nil, nil),

		subscriptionTotalItemsProcessed:                 prometheus.NewDesc("eventstore_subscription_items_processed_total", "Total items processed by subscription", []string{"event_stream_id", "group_name"}, nil),
		subscriptionLastProcessedEventNumber:            prometheus.NewDesc("eventstore_subscription_last_processed_event_number", "Last event number processed by subscription (streams other than $all)", []string{"event_stream_id", "group_name"}, nil),
		subscriptionLastKnownEventNumber:                prometheus.NewDesc("eventstore_subscription_last_known_event_number", "Last known event number in subscription (streams other than $all)", []string{"event_stream_id", "group_name"}, nil),
		subscriptionLastCheckpointedEventCommitPosition: prometheus.NewDesc("eventstore_subscription_last_checkpointed_event_commit_position", "Last checkpointed event's commit position ($all stream only)", []string{"event_stream_id", "group_name"}, nil),
		subscriptionLastKnownEventCommitPosition:        prometheus.NewDesc("eventstore_subscription_last_known_event_commit_position", "Last known event's commit position ($all stream only)", []string{"event_stream_id", "group_name"}, nil),
		subscriptionConnectionCount:                     prometheus.NewDesc("eventstore_subscription_connections", "Number of connections to subscription", []string{"event_stream_id", "group_name"}, nil),
		subscriptionTotalInFlightMessages:               prometheus.NewDesc("eventstore_subscription_messages_in_flight", "Number of messages in flight for subscription", []string{"event_stream_id", "group_name"}, nil),
		subscriptionTotalNumberOfParkedMessages:         prometheus.NewDesc("eventstore_subscription_parked_messages", "Number of parked messages for subscription", []string{"event_stream_id", "group_name"}, nil),
		subscriptionOldestParkedMessage:                 prometheus.NewDesc("eventstore_subscription_oldest_parked_message_age_seconds", "Oldest parked message age for subscription in seconds", []string{"event_stream_id", "group_name"}, nil),

		streamLastEventNumber:    prometheus.NewDesc("eventstore_stream_last_event_number", "Last event number in a stream (streams other than $all)", []string{"event_stream_id"}, nil),
		streamLastCommitPosition: prometheus.NewDesc("eventstore_stream_last_commit_position", "Last commit position in a stream ($all stream only)", []string{"event_stream_id"}, nil),
	}
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.up
	ch <- c.processCPU
	ch <- c.processMemoryBytes
	ch <- c.diskIoReadBytes
	ch <- c.diskIoWrittenBytes
	ch <- c.diskIoReadOps
	ch <- c.diskIoWriteOps
	ch <- c.uptimeSeconds
	ch <- c.tcpSentBytes
	ch <- c.tcpReceivedBytes
	ch <- c.tcpConnections

	if c.config.EnableTcpConnectionStats {
		ch <- c.tcpConnectionSentBytes
		ch <- c.tcpConnectionReceivedBytes
		ch <- c.tcpConnectionPendingSendBytes
		ch <- c.tcpConnectionPendingReceivedBytes
	}

	ch <- c.queueLength
	ch <- c.queueItemsProcessed

	ch <- c.driveTotalBytes
	ch <- c.driveAvailableBytes

	ch <- c.projectionRunning
	ch <- c.projectionStatus
	ch <- c.projectionProgress
	ch <- c.projectionEventsProcessedAfterRestart

	if c.config.IsInClusterMode() {
		ch <- c.clusterMemberAlive
		ch <- c.clusterMemberIsClone
		ch <- c.clusterMemberIsLeader
		ch <- c.clusterMemberIsFollower
		ch <- c.clusterMemberIsReadonlyReplica
	}

	ch <- c.subscriptionTotalItemsProcessed
	ch <- c.subscriptionLastProcessedEventNumber
	ch <- c.subscriptionLastKnownEventNumber
	ch <- c.subscriptionLastCheckpointedEventCommitPosition
	ch <- c.subscriptionLastKnownEventCommitPosition
	ch <- c.subscriptionConnectionCount
	ch <- c.subscriptionTotalInFlightMessages
	ch <- c.subscriptionTotalNumberOfParkedMessages
	ch <- c.subscriptionOldestParkedMessage
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	log.Info("Running scrape")

	// context is not passed to the collector, so we need to create a new one
	// https://groups.google.com/g/prometheus-developers/c/a8k4CXhGdPI
	ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
	defer cancel()

	if stats, err := c.client.GetStats(ctx); err != nil {
		log.WithError(err).Error("Error while getting data from EventStore")

		ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 0)
	} else {
		ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 1)

		c.collectFromStats(ch, stats)
	}
}

func (c *Collector) collectFromStats(ch chan<- prometheus.Metric, stats *client.Stats) {
	c.collectFromServerStats(ch, stats)
	c.collectFromTcpConnectionStats(ch, stats.TcpConnections)
	c.collectFromQueueStats(ch, stats.Server.Es.Queues)
	c.collectFromDriveStats(ch, stats.Server.System.Drives)
	c.collectFromProjectionStats(ch, stats.Projections)
	c.collectFromSubscriptionStats(ch, stats.Subscriptions)
	c.collectFromStreamStats(ch, stats.Streams)
	c.collectFromClusterStats(ch, stats)
}

func (c *Collector) collectFromServerStats(ch chan<- prometheus.Metric, stats *client.Stats) {
	ch <- prometheus.MustNewConstMetric(c.processCPU, prometheus.GaugeValue, stats.Server.Process.Cpu/100.0) // scale to 0-[num of cores]
	ch <- prometheus.MustNewConstMetric(c.processMemoryBytes, prometheus.GaugeValue, float64(stats.Server.Process.MemoryBytes))
	ch <- prometheus.MustNewConstMetric(c.diskIoReadBytes, prometheus.GaugeValue, float64(stats.Server.Process.DiskIo.ReadBytes))
	ch <- prometheus.MustNewConstMetric(c.diskIoWrittenBytes, prometheus.GaugeValue, float64(stats.Server.Process.DiskIo.WrittenBytes))
	ch <- prometheus.MustNewConstMetric(c.diskIoReadOps, prometheus.GaugeValue, float64(stats.Server.Process.DiskIo.ReadOps))
	ch <- prometheus.MustNewConstMetric(c.diskIoWriteOps, prometheus.GaugeValue, float64(stats.Server.Process.DiskIo.WriteOps))
	ch <- prometheus.MustNewConstMetric(c.tcpSentBytes, prometheus.GaugeValue, float64(stats.Server.Process.Tcp.SentBytes))
	ch <- prometheus.MustNewConstMetric(c.tcpReceivedBytes, prometheus.GaugeValue, float64(stats.Server.Process.Tcp.ReceivedBytes))
	ch <- prometheus.MustNewConstMetric(c.tcpConnections, prometheus.GaugeValue, float64(stats.Server.Process.Tcp.Connections))
}

func (c *Collector) collectFromTcpConnectionStats(ch chan<- prometheus.Metric, stats []client.TcpConnectionStats) {
	for _, tcpConn := range stats {
		id := tcpConn.ConnectionId
		clientName := tcpConn.ClientConnectionName
		remoteEndPoint := tcpConn.RemoteEndPoint
		localEndPoint := tcpConn.LocalEndPoint
		external := strconv.FormatBool(tcpConn.IsExternalConnection)
		ssl := strconv.FormatBool(tcpConn.IsSslConnection)

		labels := []string{id, clientName, remoteEndPoint, localEndPoint, external, ssl}

		ch <- prometheus.MustNewConstMetric(c.tcpConnectionSentBytes, prometheus.CounterValue, float64(tcpConn.TotalBytesSent), labels...)
		ch <- prometheus.MustNewConstMetric(c.tcpConnectionReceivedBytes, prometheus.CounterValue, float64(tcpConn.TotalBytesReceived), labels...)
		ch <- prometheus.MustNewConstMetric(c.tcpConnectionPendingSendBytes, prometheus.GaugeValue, float64(tcpConn.PendingSendBytes), labels...)
		ch <- prometheus.MustNewConstMetric(c.tcpConnectionPendingReceivedBytes, prometheus.GaugeValue, float64(tcpConn.PendingReceivedBytes), labels...)
	}
}

func (c *Collector) collectFromQueueStats(ch chan<- prometheus.Metric, stats map[string]client.QueueStats) {
	for _, queue := range stats {
		ch <- prometheus.MustNewConstMetric(c.queueLength, prometheus.GaugeValue, float64(queue.Length), queue.QueueName)
		ch <- prometheus.MustNewConstMetric(c.queueItemsProcessed, prometheus.CounterValue, float64(queue.ItemsProcessed), queue.QueueName)
	}
}

func (c *Collector) collectFromDriveStats(ch chan<- prometheus.Metric, stats map[string]client.DriveStats) {
	for driveName, drive := range stats {
		ch <- prometheus.MustNewConstMetric(c.driveTotalBytes, prometheus.GaugeValue, float64(drive.TotalBytes), driveName)
		ch <- prometheus.MustNewConstMetric(c.driveAvailableBytes, prometheus.GaugeValue, float64(drive.AvailableBytes), driveName)
	}
}

func (c *Collector) collectFromProjectionStats(ch chan<- prometheus.Metric, stats []client.ProjectionStats) {
	for _, projection := range stats {
		running := 0.0
		stopped := 0.0
		faulted := 0.0

		if projection.Status == "Running" {
			running = 1.0
		} else if projection.Status == "Stopped" {
			stopped = 1.0
		} else if strings.Contains(projection.Status, "Faulted") {
			faulted = 1.0
		}

		ch <- prometheus.MustNewConstMetric(c.projectionRunning, prometheus.GaugeValue, running, projection.EffectiveName)
		ch <- prometheus.MustNewConstMetric(c.projectionStatus, prometheus.GaugeValue, running, projection.EffectiveName, "Running")
		ch <- prometheus.MustNewConstMetric(c.projectionStatus, prometheus.GaugeValue, stopped, projection.EffectiveName, "Stopped")
		ch <- prometheus.MustNewConstMetric(c.projectionStatus, prometheus.GaugeValue, faulted, projection.EffectiveName, "Faulted")
		ch <- prometheus.MustNewConstMetric(c.projectionProgress, prometheus.GaugeValue, projection.Progress/100.0, projection.EffectiveName) // scale to 0-1
		ch <- prometheus.MustNewConstMetric(c.projectionEventsProcessedAfterRestart, prometheus.CounterValue, float64(projection.EventsProcessedAfterRestart), projection.EffectiveName)
	}
}

func (c *Collector) collectFromSubscriptionStats(ch chan<- prometheus.Metric, stats []client.SubscriptionStats) {
	for _, subscription := range stats {
		ch <- prometheus.MustNewConstMetric(c.subscriptionTotalItemsProcessed, prometheus.CounterValue, float64(subscription.TotalItemsProcessed), subscription.EventStreamID, subscription.GroupName)
		ch <- prometheus.MustNewConstMetric(c.subscriptionConnectionCount, prometheus.GaugeValue, float64(subscription.ConnectionCount), subscription.EventStreamID, subscription.GroupName)
		ch <- prometheus.MustNewConstMetric(c.subscriptionTotalInFlightMessages, prometheus.GaugeValue, float64(subscription.TotalInFlightMessages), subscription.EventStreamID, subscription.GroupName)
		ch <- prometheus.MustNewConstMetric(c.subscriptionTotalNumberOfParkedMessages, prometheus.GaugeValue, float64(subscription.TotalNumberOfParkedMessages), subscription.EventStreamID, subscription.GroupName)
		ch <- prometheus.MustNewConstMetric(c.subscriptionOldestParkedMessage, prometheus.GaugeValue, subscription.OldestParkedMessageAgeInSeconds, subscription.EventStreamID, subscription.GroupName)

		if subscription.EventStreamID == "$all" {
			lastCheckpointedEventPosition, _, err := client.EventPosition(subscription.LastCheckpointedEventPosition).ParseCommitPreparePosition()
			if err != nil {
				log.WithError(err).Warnf("Error while parsing last checkpointed event position of $all stream subscription group %s", subscription.GroupName)
			}
			lastKnownEventPosition, _, err := client.EventPosition(subscription.LastKnownEventPosition).ParseCommitPreparePosition()
			if err != nil {
				log.WithError(err).Errorf("Error while parsing last known event position of $all stream subscription group %s", subscription.GroupName)
			}
			ch <- prometheus.MustNewConstMetric(c.subscriptionLastCheckpointedEventCommitPosition, prometheus.GaugeValue, float64(lastCheckpointedEventPosition), subscription.EventStreamID, subscription.GroupName)
			ch <- prometheus.MustNewConstMetric(c.subscriptionLastKnownEventCommitPosition, prometheus.GaugeValue, float64(lastKnownEventPosition), subscription.EventStreamID, subscription.GroupName)
		} else {
			ch <- prometheus.MustNewConstMetric(c.subscriptionLastProcessedEventNumber, prometheus.GaugeValue, float64(subscription.LastProcessedEventNumber), subscription.EventStreamID, subscription.GroupName)
			ch <- prometheus.MustNewConstMetric(c.subscriptionLastKnownEventNumber, prometheus.GaugeValue, float64(subscription.LastKnownEventNumber), subscription.EventStreamID, subscription.GroupName)
		}

	}
}

func (c *Collector) collectFromStreamStats(ch chan<- prometheus.Metric, stats []client.StreamStats) {
	for _, stream := range stats {
		if stream.EventStreamID == "$all" {
			ch <- prometheus.MustNewConstMetric(c.streamLastCommitPosition, prometheus.GaugeValue, float64(stream.LastCommitPosition), stream.EventStreamID)
		} else {
			ch <- prometheus.MustNewConstMetric(c.streamLastEventNumber, prometheus.GaugeValue, float64(stream.LastEventNumber), stream.EventStreamID)
		}
	}
}

func (c *Collector) collectFromClusterStats(ch chan<- prometheus.Metric, stats *client.Stats) {
	if c.config.IsInClusterMode() {

		isLeader := 0.0
		if stats.Info.MemberState == client.MemberStateLeader {
			isLeader = 1.0
		}

		isFollower := 0.0
		if stats.Info.MemberState == client.MemberStateFollower {
			isFollower = 1.0
		}

		isReadOnlyReplica := 0.0
		if stats.Info.MemberState == client.MemberStateReadOnlyReplica {
			isReadOnlyReplica = 1.0
		}

		isClone := 0.0
		if stats.Info.MemberState == client.MemberStateClone {
			isClone = 1.0
		}

		ch <- prometheus.MustNewConstMetric(c.clusterMemberIsLeader, prometheus.GaugeValue, isLeader)
		ch <- prometheus.MustNewConstMetric(c.clusterMemberIsFollower, prometheus.GaugeValue, isFollower)
		ch <- prometheus.MustNewConstMetric(c.clusterMemberIsReadonlyReplica, prometheus.GaugeValue, isReadOnlyReplica)
		ch <- prometheus.MustNewConstMetric(c.clusterMemberIsClone, prometheus.GaugeValue, isClone)

		for _, member := range stats.ClusterMembers {
			isAlive := 0.0
			if member.IsAlive {
				isAlive = 1.0
			}

			memberName := fmt.Sprintf("%s:%d", member.HttpEndpointIp, member.HttpEndpointPort)

			ch <- prometheus.MustNewConstMetric(c.clusterMemberAlive, prometheus.GaugeValue, isAlive, memberName)
		}
	}
}
