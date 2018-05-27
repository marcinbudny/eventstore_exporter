package main

import (
	"github.com/prometheus/client_golang/prometheus"
	jp "github.com/buger/jsonparser"
)

const (
	namespace = "eventstore"
	subsystem = ""
)

type exporter struct {
	up                          			prometheus.Gauge
	processCPU								prometheus.Gauge
	processCPUScaled						prometheus.Gauge
	processMemoryBytes						prometheus.Gauge
	diskIoReadBytes							prometheus.Counter
	diskIoWrittenBytes						prometheus.Counter
	diskIoReadOps							prometheus.Counter
	diskIoWriteOps							prometheus.Counter
	uptimeSeconds							prometheus.Counter
	tcpSentBytes							prometheus.Counter
	tcpReceivedBytes						prometheus.Counter
	tcpConnections							prometheus.Gauge
	queueLength								prometheus.Gauge
	queueItemsProcessed						prometheus.Gauge
	projectionRunning						prometheus.Gauge
	projectionProgress						prometheus.Gauge
	projectionEventsProcessedAfterRestart	prometheus.Counter
	clusterMembers							prometheus.Gauge
	clusterMemberAlive						prometheus.Gauge
	clusterMemberIsMaster					prometheus.Gauge
}

func newExporter() *exporter {
	return &exporter {
		up:                         			createGauge("up", "Whether the EventStore scrape was successful"),
		processCPU:								createGauge("process_cpu", "Process CPU usage, 0 - number of cores"),
		processCPUScaled:						createGauge("process_cpu_scaled", "Process CPU usage scaled to number of cores, 0 - 1, 1 = full load on all cores"),
		processMemoryBytes:						createGauge("process_memory_bytes", "Process memory usage"),
		diskIoReadBytes:						createCounter("disk_io_read_bytes", "Total number of disk IO read bytes"),
		diskIoWrittenBytes:						createCounter("disk_io_written_bytes", "Total number of disk IO written bytes"),
		diskIoReadOps:							createCounter("disk_io_read_ops", "Total number of disk IO read operations"),
		diskIoWriteOps:							createCounter("disk_io_write_ops", "Total number of disk IO write operations"),
		uptimeSeconds:							createCounter("uptime_seconds", "Total uptime seconds"),
		tcpSentBytes:							createCounter("tcp_sent_bytes", "TCP sent bytes"),
		tcpReceivedBytes:						createCounter("tcp_received_bytes", "TCP received bytes"),
		tcpConnections:							createGauge("tcp_connections", "Current number of TCP connections"),
		queueLength:							createGauge("queue_length", "Queue length"),
		queueItemsProcessed:					createGauge("queue_items_processed_total", "Total number items processed by queue"),
		projectionRunning:						createGauge("projection_running", "If 1, projection is in 'Running' state"),
		projectionProgress:						createGauge("projection_progress", "Projection progress 0 - 1, where 1 = projection progress at 100%"),
		projectionEventsProcessedAfterRestart:	createCounter("projection_events_processed_after_restart", "Projection event processed count"),
		clusterMembers:							createGauge("cluster_member_count", "Current count of cluster members"),
		clusterMemberAlive:						createGauge("cluster_member_alive", "If 1, cluster member is alive, as seen from current cluster member"),
		clusterMemberIsMaster:					createGauge("cluster_member_is_master", "If 1, current cluster member is the master"),
	}
}

func (e *exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.up.Desc()
	ch <- e.processCPU.Desc()
	ch <- e.processMemoryBytes.Desc()
	ch <- e.diskIoReadBytes.Desc()
	ch <- e.diskIoWrittenBytes.Desc()
	ch <- e.diskIoReadOps.Desc()
	ch <- e.diskIoWriteOps.Desc()
	ch <- e.uptimeSeconds.Desc()
	ch <- e.tcpSentBytes.Desc()
	ch <- e.tcpReceivedBytes.Desc()
	ch <- e.tcpConnections.Desc()
	ch <- e.queueLength.Desc()
	ch <- e.queueItemsProcessed.Desc()
	ch <- e.projectionRunning.Desc()
	ch <- e.projectionProgress.Desc()
	ch <- e.projectionEventsProcessedAfterRestart.Desc()
	
	if(isInClusterMode()) {
		ch <- e.clusterMembers.Desc()
		ch <- e.clusterMemberAlive.Desc()
		ch <- e.clusterMemberIsMaster.Desc()
	}
}

func (e *exporter) Collect(ch chan<- prometheus.Metric) {
	log.Info("Running scrape")

	if stats, err := getStats(); err != nil {
		log.WithError(err).Error("Error while getting data from EventStore")

		e.up.Set(0)
		ch <- e.up
	} else {
		e.up.Set(1)
		ch <- e.up

		e.processCPU.Set(getProcessCPU(stats))
		ch <- e.processCPU

		e.processCPUScaled.Set(getProcessCPUScaled(stats))
		ch <- e.processCPUScaled

		e.diskIoReadBytes.Set(getDiskIoReadBytes(stats))
		ch <- e.diskIoReadBytes
		
		e.diskIoWrittenBytes.Set(getDiskIoWrittenBytes(stats))
		ch <- e.diskIoWrittenBytes

		e.diskIoReadOps.Set(getDiskIoReadOps(stats))
		ch <- e.diskIoReadOps

		e.diskIoWriteOps.Set(getDiskIoWriteOps(stats))
		ch <- e.diskIoWriteOps

		e.tcpConnections.Set(getTcpConnections(stats))
		ch <- e.tcpConnections

		e.tcpReceivedBytes.Set(getTcpReceivedBytes(stats))
		ch <- e.tcpReceivedBytes

		e.tcpSentBytes.Set(getTcpSentBytes(stats))
		ch <- e.tcpSentBytes

	}
}

func getProcessCPU(stats *stats) float64 {
	value, _ := jp.GetFloat(stats.serverStats, "proc", "cpu") 
	return value / 100.0
}

func getProcessCPUScaled(stats *stats) float64 {
	value, _ := jp.GetFloat(stats.serverStats, "proc", "cpuScaled") 
	return value / 100.0
}

func getProcessMemory(stats *stats) float64 {
	value, _ := jp.GetFloat(stats.serverStats, "proc", "mem")
	return value
}

func getDiskIoReadBytes(stats *stats) float64 {
	value, _ := jp.GetFloat(stats.serverStats, "proc", "diskIo", "readBytes")
	return value
}

func getDiskIoWrittenBytes(stats *stats) float64 {
	value, _ := jp.GetFloat(stats.serverStats, "proc", "diskIo", "writtenBytes")
	return value
}

func getDiskIoReadOps(stats *stats) float64 {
	value, _ := jp.GetFloat(stats.serverStats, "proc", "diskIo", "readOps")
	return value
}

func getDiskIoWriteOps(stats *stats) float64 {
	value, _ := jp.GetFloat(stats.serverStats, "proc", "diskIo", "writeOps")
	return value
}

func getTcpSentBytes(stats *stats) float64 {
	value, _ := jp.GetFloat(stats.serverStats, "proc", "tcp", "sentBytesTotal")
	return value
}

func getTcpReceivedBytes(stats *stats) float64 {
	value, _ := jp.GetFloat(stats.serverStats, "proc", "tcp", "receivedBytesTotal")
	return value
}

func getTcpConnections(stats *stats) float64 {
	value, _ := jp.GetFloat(stats.serverStats, "proc", "tcp", "connections")
	return value
}



func createGauge(name string, help string) prometheus.Gauge {
	return prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      name,
		Help:      help,
	})
}

func createDatabaseGaugeVec(name string, help string) *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      name,
		Help:      help,
	}, []string{"database"})
}

func createCounter(name string, help string) prometheus.Counter {
	return prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      name,
		Help:      help,
	})
}

func createDatabaseCounterVec(name string, help string) *prometheus.CounterVec {
	return prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      name,
		Help:      help,
	}, []string{"database"})
}

func isInClusterMode() bool {
	return clusterMode == "cluster"
}