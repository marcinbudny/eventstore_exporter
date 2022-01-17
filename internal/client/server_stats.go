package client

import jp "github.com/buger/jsonparser"

type getServerStatsResult struct {
	process  *ProcessStats
	diskIo   *DiskIoStats
	tcpStats *TcpStats
	queues   []QueueStats
	drives   []DriveStats

	err error
}

type ProcessStats struct {
	Cpu         float64
	MemoryBytes float64
}

type DiskIoStats struct {
	ReadBytes    float64
	WrittenBytes float64
	ReadOps      float64
	WriteOps     float64
}

type TcpStats struct {
	SentBytes     float64
	ReceivedBytes float64
	Connections   float64
}

type QueueStats struct {
	Name           string
	Length         float64
	ItemsProcessed float64
}

type DriveStats struct {
	Name           string
	TotalBytes     float64
	AvailableBytes float64
}

func (esClient *EventStoreStatsClient) getServerStats() <-chan getServerStatsResult {
	stats := make(chan getServerStatsResult, 1)
	go func() {
		if serverJson, err := esClient.esHttpGet("/stats", false); err == nil {
			stats <- getServerStatsResult{
				process: &ProcessStats{
					Cpu:         getFloat(serverJson, "proc", "cpu") / 100.0,
					MemoryBytes: getFloat(serverJson, "proc", "mem"),
				},
				diskIo: &DiskIoStats{
					ReadBytes:    getFloat(serverJson, "proc", "diskIo", "readBytes"),
					WrittenBytes: getFloat(serverJson, "proc", "diskIo", "writtenBytes"),
					ReadOps:      getFloat(serverJson, "proc", "diskIo", "readOps"),
					WriteOps:     getFloat(serverJson, "proc", "diskIo", "writeOps"),
				},
				tcpStats: &TcpStats{
					SentBytes:     getFloat(serverJson, "proc", "tcp", "sentBytesTotal"),
					ReceivedBytes: getFloat(serverJson, "proc", "tcp", "receivedBytesTotal"),
					Connections:   getFloat(serverJson, "proc", "tcp", "connections"),
				},
				queues: getQueueStats(serverJson),
				drives: getDriveStats(serverJson),
			}
		} else {
			stats <- getServerStatsResult{err: err}
		}
	}()

	return stats
}

func getQueueStats(serverStats []byte) []QueueStats {
	queues := []QueueStats{}

	jp.ObjectEach(serverStats, func(key []byte, jsonValue []byte, dataType jp.ValueType, offset int) error {
		queues = append(queues, QueueStats{
			Name:           string(key),
			Length:         getFloat(jsonValue, "length"),
			ItemsProcessed: getFloat(jsonValue, "totalItemsProcessed"),
		})

		return nil
	}, "es", "queue")

	return queues
}

func getDriveStats(serverStats []byte) []DriveStats {
	drives := []DriveStats{}

	jp.ObjectEach(serverStats, func(key []byte, jsonValue []byte, dataType jp.ValueType, offset int) error {
		drives = append(drives, DriveStats{
			Name:           string(key),
			TotalBytes:     getFloat(jsonValue, "totalBytes"),
			AvailableBytes: getFloat(jsonValue, "availableBytes"),
		})

		return nil
	}, "sys", "drive")

	return drives
}
