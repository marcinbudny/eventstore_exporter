package client

import jp "github.com/buger/jsonparser"

type getServerStatsResult struct {
	process  *ProcessStats
	diskIo   *DiskIoStats
	tcpStats *TcpStats
	queues   []QueueStats
	drives   []DriveStats
}

type ProcessStats struct {
	Cpu         float64
	MemoryBytes int64
}

type DiskIoStats struct {
	ReadBytes    int64
	WrittenBytes int64
	ReadOps      int64
	WriteOps     int64
}

type TcpStats struct {
	SentBytes     int64
	ReceivedBytes int64
	Connections   int64
}

type QueueStats struct {
	Name           string
	Length         int64
	ItemsProcessed int64
}

type DriveStats struct {
	Name           string
	TotalBytes     int64
	AvailableBytes int64
}

func (client *EventStoreStatsClient) getServerStats() (*getServerStatsResult, error) {
	if serverJson, err := client.esHttpGet("/stats", false); err == nil {
		return &getServerStatsResult{
			process: &ProcessStats{
				Cpu:         getFloat(serverJson, "proc", "cpu") / 100.0,
				MemoryBytes: getInt(serverJson, "proc", "mem"),
			},
			diskIo: &DiskIoStats{
				ReadBytes:    getInt(serverJson, "proc", "diskIo", "readBytes"),
				WrittenBytes: getInt(serverJson, "proc", "diskIo", "writtenBytes"),
				ReadOps:      getInt(serverJson, "proc", "diskIo", "readOps"),
				WriteOps:     getInt(serverJson, "proc", "diskIo", "writeOps"),
			},
			tcpStats: &TcpStats{
				SentBytes:     getInt(serverJson, "proc", "tcp", "sentBytesTotal"),
				ReceivedBytes: getInt(serverJson, "proc", "tcp", "receivedBytesTotal"),
				Connections:   getInt(serverJson, "proc", "tcp", "connections"),
			},
			queues: getQueueStats(serverJson),
			drives: getDriveStats(serverJson),
		}, nil
	} else {
		return nil, err
	}
}

func getQueueStats(serverStats []byte) []QueueStats {
	queues := []QueueStats{}

	jp.ObjectEach(serverStats, func(key []byte, jsonValue []byte, dataType jp.ValueType, offset int) error {
		queues = append(queues, QueueStats{
			Name:           string(key),
			Length:         getInt(jsonValue, "length"),
			ItemsProcessed: getInt(jsonValue, "totalItemsProcessed"),
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
			TotalBytes:     getInt(jsonValue, "totalBytes"),
			AvailableBytes: getInt(jsonValue, "availableBytes"),
		})

		return nil
	}, "sys", "drive")

	return drives
}
