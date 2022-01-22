package collector

import (
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

	queueLength         *prometheus.Desc
	queueItemsProcessed *prometheus.Desc

	driveTotalBytes     *prometheus.Desc
	driveAvailableBytes *prometheus.Desc

	projectionRunning                     *prometheus.Desc
	projectionProgress                    *prometheus.Desc
	projectionEventsProcessedAfterRestart *prometheus.Desc

	clusterMemberAlive             *prometheus.Desc
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

	streamLastPosition *prometheus.Desc
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

		queueLength:         prometheus.NewDesc("eventstore_queue_length", "Queue length", []string{"queue"}, nil),
		queueItemsProcessed: prometheus.NewDesc("eventstore_queue_items_processed_total", "Total number items processed by queue", []string{"queue"}, nil),

		driveTotalBytes:     prometheus.NewDesc("eventstore_drive_total_bytes", "Drive total size in bytes", []string{"drive"}, nil),
		driveAvailableBytes: prometheus.NewDesc("eventstore_drive_available_bytes", "Drive available bytes", []string{"drive"}, nil),

		projectionRunning:                     prometheus.NewDesc("eventstore_projection_running", "If 1, projection is in 'Running' state", []string{"projection"}, nil),
		projectionProgress:                    prometheus.NewDesc("eventstore_projection_progress", "Projection progress 0 - 1, where 1 = projection progress at 100%", []string{"projection"}, nil),
		projectionEventsProcessedAfterRestart: prometheus.NewDesc("eventstore_projection_events_processed_after_restart_total", "Projection event processed count after restart", []string{"projection"}, nil),

		clusterMemberAlive:             prometheus.NewDesc("eventstore_cluster_member_alive", "If 1, cluster member is alive, as seen from current cluster member", []string{"member"}, nil),
		clusterMemberIsClone:           prometheus.NewDesc("eventstore_cluster_member_is_clone", "If 1, current cluster member is a clone", nil, nil),
		clusterMemberIsLeader:          prometheus.NewDesc("eventstore_cluster_member_is_leader", "If 1, current cluster member is the leader", nil, nil),
		clusterMemberIsFollower:        prometheus.NewDesc("eventstore_cluster_member_is_follower", "If 1, current cluster member is a follower", nil, nil),
		clusterMemberIsReadonlyReplica: prometheus.NewDesc("eventstore_cluster_member_is_readonly_replica", "If 1, current cluster member is a readonly replica", nil, nil),

		subscriptionTotalItemsProcessed:         prometheus.NewDesc("eventstore_subscription_items_processed_total", "Total items processed by subscription", []string{"event_stream_id", "group_name"}, nil),
		subscriptionLastProcessedEventNumber:    prometheus.NewDesc("eventstore_subscription_last_processed_event_number", "Last event number processed by subscription", []string{"event_stream_id", "group_name"}, nil),
		subscriptionLastKnownEventNumber:        prometheus.NewDesc("eventstore_subscription_last_known_event_number", "Last known event number in subscription", []string{"event_stream_id", "group_name"}, nil),
		subscriptionConnectionCount:             prometheus.NewDesc("eventstore_subscription_connections", "Number of connections to subscription", []string{"event_stream_id", "group_name"}, nil),
		subscriptionTotalInFlightMessages:       prometheus.NewDesc("eventstore_subscription_messages_in_flight", "Number of messages in flight for subscription", []string{"event_stream_id", "group_name"}, nil),
		subscriptionTotalNumberOfParkedMessages: prometheus.NewDesc("eventstore_subscription_parked_messages", "Number of parked messages for subscription", []string{"event_stream_id", "group_name"}, nil),
		subscriptionOldestParkedMessage:         prometheus.NewDesc("eventstore_subscription_oldest_parked_message_age_seconds", "Oldest parked message age for subscription in seconds", []string{"event_stream_id", "group_name"}, nil),

		streamLastPosition: prometheus.NewDesc("eventstore_stream_last_event_position", "Last event number in a stream or last commit position in case of $all stream", []string{"event_stream_id"}, nil),
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

	ch <- c.queueLength
	ch <- c.queueItemsProcessed

	ch <- c.driveTotalBytes
	ch <- c.driveAvailableBytes

	ch <- c.projectionRunning
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
	ch <- c.subscriptionConnectionCount
	ch <- c.subscriptionTotalInFlightMessages
	ch <- c.subscriptionTotalNumberOfParkedMessages
	ch <- c.subscriptionOldestParkedMessage
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	log.Info("Running scrape")

	if stats, err := c.client.GetStats(); err != nil {
		log.WithError(err).Error("Error while getting data from EventStore")

		ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 0)
	} else {
		ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 1)

		ch <- prometheus.MustNewConstMetric(c.processCPU, prometheus.GaugeValue, stats.Process.Cpu)

		ch <- prometheus.MustNewConstMetric(c.processMemoryBytes, prometheus.GaugeValue, float64(stats.Process.MemoryBytes))
		ch <- prometheus.MustNewConstMetric(c.diskIoReadBytes, prometheus.GaugeValue, float64(stats.DiskIo.ReadBytes))
		ch <- prometheus.MustNewConstMetric(c.diskIoWrittenBytes, prometheus.GaugeValue, float64(stats.DiskIo.WrittenBytes))
		ch <- prometheus.MustNewConstMetric(c.diskIoReadOps, prometheus.GaugeValue, float64(stats.DiskIo.ReadOps))
		ch <- prometheus.MustNewConstMetric(c.diskIoWriteOps, prometheus.GaugeValue, float64(stats.DiskIo.WriteOps))
		ch <- prometheus.MustNewConstMetric(c.tcpSentBytes, prometheus.GaugeValue, float64(stats.Tcp.SentBytes))
		ch <- prometheus.MustNewConstMetric(c.tcpReceivedBytes, prometheus.GaugeValue, float64(stats.Tcp.ReceivedBytes))
		ch <- prometheus.MustNewConstMetric(c.tcpConnections, prometheus.GaugeValue, float64(stats.Tcp.Connections))

		for _, queue := range stats.Queues {
			ch <- prometheus.MustNewConstMetric(c.queueLength, prometheus.GaugeValue, float64(queue.Length), queue.Name)
			ch <- prometheus.MustNewConstMetric(c.queueItemsProcessed, prometheus.CounterValue, float64(queue.ItemsProcessed), queue.Name)
		}

		for _, drive := range stats.Drives {
			ch <- prometheus.MustNewConstMetric(c.driveTotalBytes, prometheus.GaugeValue, float64(drive.TotalBytes), drive.Name)
			ch <- prometheus.MustNewConstMetric(c.driveAvailableBytes, prometheus.GaugeValue, float64(drive.AvailableBytes), drive.Name)
		}

		for _, projection := range stats.Projections {
			running := 0.0
			if projection.Running {
				running = 1.0
			}
			ch <- prometheus.MustNewConstMetric(c.projectionRunning, prometheus.GaugeValue, running, projection.Name)
			ch <- prometheus.MustNewConstMetric(c.projectionProgress, prometheus.GaugeValue, projection.Progress, projection.Name)
			ch <- prometheus.MustNewConstMetric(c.projectionEventsProcessedAfterRestart, prometheus.CounterValue, float64(projection.EventsProcessedAfterRestart), projection.Name)
		}

		for _, subscription := range stats.Subscriptions {
			ch <- prometheus.MustNewConstMetric(c.subscriptionTotalItemsProcessed, prometheus.CounterValue, float64(subscription.TotalItemsProcessed), subscription.EventStreamID, subscription.GroupName)
			ch <- prometheus.MustNewConstMetric(c.subscriptionLastProcessedEventNumber, prometheus.GaugeValue, float64(subscription.LastProcessedEventNumber), subscription.EventStreamID, subscription.GroupName)
			ch <- prometheus.MustNewConstMetric(c.subscriptionLastKnownEventNumber, prometheus.GaugeValue, float64(subscription.LastKnownEventNumber), subscription.EventStreamID, subscription.GroupName)
			ch <- prometheus.MustNewConstMetric(c.subscriptionConnectionCount, prometheus.GaugeValue, float64(subscription.ConnectionCount), subscription.EventStreamID, subscription.GroupName)
			ch <- prometheus.MustNewConstMetric(c.subscriptionTotalInFlightMessages, prometheus.GaugeValue, float64(subscription.TotalInFlightMessages), subscription.EventStreamID, subscription.GroupName)
			ch <- prometheus.MustNewConstMetric(c.subscriptionTotalNumberOfParkedMessages, prometheus.GaugeValue, float64(subscription.TotalNumberOfParkedMessages), subscription.EventStreamID, subscription.GroupName)
			ch <- prometheus.MustNewConstMetric(c.subscriptionOldestParkedMessage, prometheus.GaugeValue, subscription.OldestParkedMessageAgeInSeconds, subscription.EventStreamID, subscription.GroupName)
		}

		for _, stream := range stats.Streams {
			ch <- prometheus.MustNewConstMetric(c.streamLastPosition, prometheus.GaugeValue, float64(stream.LastPosition), stream.EventStreamID)
		}

		if c.config.IsInClusterMode() {
			isLeader := 0.0
			if stats.Cluster.CurrentNodeMemberType == client.Leader {
				isLeader = 1.0
			}

			isFollower := 0.0
			if stats.Cluster.CurrentNodeMemberType == client.Follower {
				isFollower = 1.0
			}

			isReadOnlyReplica := 0.0
			if stats.Cluster.CurrentNodeMemberType == client.ReadOnlyReplica {
				isReadOnlyReplica = 1.0
			}

			isClone := 0.0
			if stats.Cluster.CurrentNodeMemberType == client.Clone {
				isClone = 1.0
			}

			ch <- prometheus.MustNewConstMetric(c.clusterMemberIsLeader, prometheus.GaugeValue, isLeader)
			ch <- prometheus.MustNewConstMetric(c.clusterMemberIsFollower, prometheus.GaugeValue, isFollower)
			ch <- prometheus.MustNewConstMetric(c.clusterMemberIsReadonlyReplica, prometheus.GaugeValue, isReadOnlyReplica)
			ch <- prometheus.MustNewConstMetric(c.clusterMemberIsClone, prometheus.GaugeValue, isClone)

			for _, member := range stats.Cluster.Members {
				isAlive := 0.0
				if member.IsAlive {
					isAlive = 1.0
				}

				ch <- prometheus.MustNewConstMetric(c.clusterMemberAlive, prometheus.GaugeValue, isAlive, member.MemberName)
			}
		}

	}
}
