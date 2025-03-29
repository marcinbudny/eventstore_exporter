package client

import (
	"context"
	"sync"

	"github.com/EventStore/EventStore-Client-Go/v3/esdb"
	log "github.com/sirupsen/logrus"
)

type StreamStats struct {
	EventStreamID      string
	LastCommitPosition int64
	LastEventNumber    int64
}

func (client *EventStoreStatsClient) getStreamStats(ctx context.Context) ([]StreamStats, error) {
	streamStats, err := getStreamStatsFromEachStream(ctx, client)
	if err != nil {
		return nil, err
	}

	return streamStats, nil
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

	streamStats := make([]StreamStats, len(client.config.Streams))
	var wg sync.WaitGroup

	for i, stream := range client.config.Streams {
		wg.Add(1)

		go func(stream string, idx int) {
			defer wg.Done()

			log.WithField("stream", stream).Debug("Getting stream stats")
			if stats, getErr := getSingleStreamStats(ctx, grpcClient, stream); getErr == nil {
				streamStats[idx] = stats
			} else {
				streamStats[idx] = StreamStats{EventStreamID: stream, LastCommitPosition: -1, LastEventNumber: -1}
			}

		}(stream, i)
	}

	wg.Wait()

	return streamStats, nil
}

func getSingleStreamStats(ctx context.Context, grpcClient *esdb.Client, stream string) (StreamStats, error) {
	if stream == "$all" {
		return getAllStreamStats(ctx, grpcClient)
	}

	return getRegularStreamStats(ctx, grpcClient, stream)
}

func getAllStreamStats(ctx context.Context, grpcClient *esdb.Client) (StreamStats, error) {
	event, err := readSingleEventFromAll(ctx, grpcClient, esdb.ReadAllOptions{Direction: esdb.Backwards, From: esdb.End{}})

	if err != nil {
		log.WithError(err).Error("Error when reading last event from $all stream")

		return StreamStats{}, err
	}

	return StreamStats{
		EventStreamID:      "$all",
		LastCommitPosition: int64(event.Event.Position.Commit), //nolint:gosec // TODO: fix this
		LastEventNumber:    -1,
	}, nil

}

func getRegularStreamStats(ctx context.Context, grpcClient *esdb.Client, stream string) (StreamStats, error) {

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
		LastEventNumber:    int64(event.Event.EventNumber), //nolint:gosec // TODO: fix this
	}, nil

}
