package client

import "context"

type TCPConnectionStats struct {
	RemoteEndPoint       string `json:"remoteEndPoint"`
	LocalEndPoint        string `json:"localEndPoint"`
	ConnectionID         string `json:"connectionId"`
	ClientConnectionName string `json:"clientConnectionName"`
	TotalBytesSent       int64  `json:"totalBytesSent"`
	TotalBytesReceived   int64  `json:"totalBytesReceived"`
	PendingSendBytes     int64  `json:"pendingSendBytes"`
	PendingReceivedBytes int64  `json:"pendingReceivedBytes"`
	IsExternalConnection bool   `json:"isExternalConnection"`
	IsSslConnection      bool   `json:"isSslConnection"`
}

func (client *EventStoreStatsClient) getTCPConnectionStats(ctx context.Context) ([]TCPConnectionStats, error) {
	if !client.config.EnableTCPConnectionStats {
		return []TCPConnectionStats{}, nil
	}

	stats, err := esHTTPGetAndParse[[]TCPConnectionStats](ctx, client, "/stats/tcp", false)
	if err != nil {
		return nil, err
	}

	return stats, nil
}
