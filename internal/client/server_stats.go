package client

import (
	"context"
)

type ServerStats struct {
	Process ProcessStats `json:"proc"`
	System  SystemStats  `json:"sys"`
	Es      EsStats      `json:"es"`
}

type ProcessStats struct {
	CPU         float64     `json:"cpu"`
	MemoryBytes int64       `json:"mem"`
	DiskIo      DiskIoStats `json:"diskIo"`
	TCP         TCPStats    `json:"tcp"`
}

type DiskIoStats struct {
	ReadBytes    int64 `json:"readBytes"`
	WrittenBytes int64 `json:"writtenBytes"`
	ReadOps      int64 `json:"readOps"`
	WriteOps     int64 `json:"writeOps"`
}

type TCPStats struct {
	SentBytes     int64 `json:"sentBytesTotal"`
	ReceivedBytes int64 `json:"receivedBytesTotal"`
	Connections   int64 `json:"connections"`
}

type SystemStats struct {
	Drives map[string]DriveStats `json:"drive"`
}

type DriveStats struct {
	TotalBytes     int64 `json:"totalBytes"`
	AvailableBytes int64 `json:"availableBytes"`
}

type EsStats struct {
	Queues map[string]QueueStats `json:"queue"`
}

type QueueStats struct {
	QueueName      string `json:"queueName"`
	Length         int64  `json:"length"`
	ItemsProcessed int64  `json:"totalItemsProcessed"`
}

func (client *EventStoreStatsClient) getServerStats(ctx context.Context) (*ServerStats, error) {
	stats, err := esHTTPGetAndParse[ServerStats](ctx, client, "/stats", false)
	if err != nil {
		return nil, err
	}

	return &stats, nil
}
