package client

import "context"

type TcpConnectionStats struct {
	RemoteEndPoint       string `json:"remoteEndPoint"`
	LocalEndPoint        string `json:"localEndPoint"`
	ConnectionId         string `json:"connectionId"`
	ClientConnectionName string `json:"clientConnectionName"`
	TotalBytesSent       int64  `json:"totalBytesSent"`
	TotalBytesReceived   int64  `json:"totalBytesReceived"`
	PendingSendBytes     int64  `json:"pendingSendBytes"`
	PendingReceivedBytes int64  `json:"pendingReceivedBytes"`
	IsExternalConnection bool   `json:"isExternalConnection"`
	IsSslConnection      bool   `json:"isSslConnection"`
}

func (client *EventStoreStatsClient) getTcpConnectionStats(ctx context.Context) ([]TcpConnectionStats, error) {
	if !client.config.EnableTcpConnectionStats {
		return []TcpConnectionStats{}, nil
	}

	stats, err := esHttpGetAndParse[[]TcpConnectionStats](ctx, client, "/stats/tcp", false)
	if err != nil {
		return nil, err
	}

	return stats, nil
}
