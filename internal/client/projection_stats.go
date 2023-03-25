package client

import (
	"context"
)

type projectionStatsEnvelope struct {
	Projections []ProjectionStats `json:"projections"`
}

type ProjectionStats struct {
	EffectiveName               string  `json:"effectiveName"`
	Status                      string  `json:"status"`
	Progress                    float64 `json:"progress"`
	EventsProcessedAfterRestart int64   `json:"eventsProcessedAfterRestart"`
}

func (client *EventStoreStatsClient) getProjectionStats(ctx context.Context) ([]ProjectionStats, error) {
	envelope, err := esHTTPGetAndParse[projectionStatsEnvelope](ctx, client, "/projections/all-non-transient", true)
	if err != nil {
		return nil, err
	}

	return envelope.Projections, nil
}
