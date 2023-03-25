package client

import (
	"context"
)

type EsInfo struct {
	EsVersion   EventStoreVersion `json:"esVersion"`
	MemberState string            `json:"state"`
	Features    Features          `json:"features"`
}

type Features struct {
	Projections    bool `json:"projections"`
	UserManagement bool `json:"userManagement"`
	AtomPub        bool `json:"atomPub"`
}

func (client *EventStoreStatsClient) GetEsInfo(ctx context.Context) (*EsInfo, error) {
	info, err := esHTTPGetAndParse[EsInfo](ctx, client, "/info", false)
	if err != nil {
		return nil, err
	}

	return &info, nil
}
