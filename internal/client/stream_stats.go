package client

import (
	"context"
	"sync"
	"time"

	"github.com/EventStore/EventStore-Client-Go/v3/esdb"
	log "github.com/sirupsen/logrus"
)

type StreamStats struct {
	EventStreamID      string
	LastCommitPosition int64
	LastEventNumber    int64
}

func (client *EventStoreStatsClient) getStreamStats(ctx context.Context) ([]StreamStats, error) {
	if streamStats, err := getStreamStatsFromEachStream(ctx, client); err == nil {
		return streamStats, nil
	} else {
		return nil, err
	}
}

func getStreamStatsFromEachStream(ctx context.Context, client *EventStoreStatsClient) ([]StreamStats, error) {
	if len(client.config.Streams) == 0 {
		return make([]StreamStats, 0), nil
	}

	grpcClient, err := client.getGrpcClient()
	if err != nil {
		return nil, err
	}
	defer grpcClient.Close()

	streamStats := make(chan StreamStats, len(client.config.Streams))
	var wg sync.WaitGroup

	for _, stream := range client.config.Streams {
		wg.Add(1)

		go func(stream string) {
			defer wg.Done()

			log.WithField("stream", stream).Debug("Getting stream stats")
			if stats, getErr := getSingleStreamStats(ctx, grpcClient, stream, client.config.Timeout); getErr == nil {
				streamStats <- stats
			}

		}(stream)
	}

	wg.Wait()
	close(streamStats)

	return toSlice(streamStats), nil
}

func toSlice(streamStats <-chan StreamStats) []StreamStats {
	streamStatsSlice := make([]StreamStats, 0)
	for s := range streamStats {
		streamStatsSlice = append(streamStatsSlice, s)
	}

	return streamStatsSlice
}

func getSingleStreamStats(ctx context.Context, grpcClient *esdb.Client, stream string, timeout time.Duration) (StreamStats, error) {
	if stream == "$all" {
		return getAllStreamStats(ctx, grpcClient, timeout)
	}

	return getRegularStreamStats(ctx, grpcClient, stream, timeout)
}

func getAllStreamStats(ctx context.Context, grpcClient *esdb.Client, timeout time.Duration) (StreamStats, error) {
	event, err := readSingleEventFromAll(ctx, grpcClient, esdb.ReadAllOptions{Direction: esdb.Backwards, From: esdb.End{}})

	if err != nil {
		log.WithError(err).Error("Error when reading last event from $all stream")

		return StreamStats{}, err
	}

	return StreamStats{
		EventStreamID:      "$all",
		LastCommitPosition: int64(event.Event.Position.Commit),
		LastEventNumber:    -1,
	}, nil

}

func getRegularStreamStats(ctx context.Context, grpcClient *esdb.Client, stream string, timeout time.Duration) (StreamStats, error) {

	event, err := readSingleEvent(ctx, grpcClient, stream, esdb.ReadStreamOptions{Direction: esdb.Backwards, From: esdb.End{}})
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"streamId": stream,
		}).Error("Error when reading last event from stream")

		return StreamStats{}, err
	}

	return StreamStats{
		EventStreamID:      stream,
		LastCommitPosition: -1,
		LastEventNumber:    int64(event.Event.EventNumber),
	}, nil

}
