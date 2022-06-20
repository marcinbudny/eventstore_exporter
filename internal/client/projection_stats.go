package client

import (
	"strings"

	jp "github.com/buger/jsonparser"
)

type getProjectionStatsResult struct {
	projections []ProjectionStats
	err         error
}

type ProjectionStats struct {
	Name                        string
	Running                     bool
	Stopped                     bool
	Faulted                     bool
	Progress                    float64
	EventsProcessedAfterRestart int64
}

func (client *EventStoreStatsClient) getProjectionStats() <-chan getProjectionStatsResult {
	stats := make(chan getProjectionStatsResult, 1)

	go func() {
		if projectionsJson, err := client.esHttpGet("/projections/all-non-transient", true); err == nil {
			stats <- getProjectionStatsResult{
				projections: getProjectionStats(projectionsJson),
			}
		} else {
			stats <- getProjectionStatsResult{err: err}
		}

	}()

	return stats
}

func getProjectionStats(projectionsJson []byte) []ProjectionStats {
	projections := []ProjectionStats{}

	jp.ArrayEach(projectionsJson, func(jsonValue []byte, dataType jp.ValueType, offset int, err error) {
		projections = append(projections, ProjectionStats{
			Name:                        getString(jsonValue, "effectiveName"),
			Running:                     getString(jsonValue, "status") == "Running",
			Stopped:                     getString(jsonValue, "status") == "Stopped",
			Faulted:                     strings.Contains(getString(jsonValue, "status"), "Faulted"),
			Progress:                    getFloat(jsonValue, "progress") / 100.0, // scale to 0-1
			EventsProcessedAfterRestart: getInt(jsonValue, "eventsProcessedAfterRestart"),
		})

	}, "projections")

	return projections
}
