package client

import (
	"context"
	"sync"
	"time"

	"github.com/EventStore/EventStore-Client-Go/v2/esdb"
	log "github.com/sirupsen/logrus"
)

type StreamStats struct {
	EventStreamID      string
	LastCommitPosition int64
	LastEventNumber    int64
}

type getStreamStatsResult struct {
	streams []StreamStats
	err     error
}

func (client *EventStoreStatsClient) getStreamStats() <-chan getStreamStatsResult {
	resultChan := make(chan getStreamStatsResult, 1)

	go func() {
		if streamStats, err := getStreamStatsFromEachStream(client); err == nil {
			resultChan <- getStreamStatsResult{streams: streamStats}
		} else {
			resultChan <- getStreamStatsResult{err: err}
		}
	}()

	return resultChan
}

func getStreamStatsFromEachStream(client *EventStoreStatsClient) ([]StreamStats, error) {
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
			if stats, getErr := getSingleStreamStats(grpcClient, stream, client.config.Timeout); getErr == nil {
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

func getSingleStreamStats(grpcClient *esdb.Client, stream string, timeout time.Duration) (StreamStats, error) {
	if stream == "$all" {
		return getAllStreamStats(grpcClient, timeout)
	}

	return getRegularStreamStats(grpcClient, stream, timeout)
}

func getAllStreamStats(grpcClient *esdb.Client, timeout time.Duration) (StreamStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	read, err := grpcClient.ReadAll(ctx, esdb.ReadAllOptions{Direction: esdb.Backwards, From: esdb.EndPosition}, 1)
	if err == nil {
		defer read.Close()
		event, err := read.Recv()
		if err == nil {
			return StreamStats{
				EventStreamID:      "$all",
				LastCommitPosition: int64(event.Event.Position.Commit),
				LastEventNumber:    -1,
			}, nil
		}
	}

	log.WithFields(log.Fields{
		"error": err,
	}).Error("Error when reading last event from $all stream")

	return StreamStats{}, err
}

func getRegularStreamStats(grpcClient *esdb.Client, stream string, timeout time.Duration) (StreamStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	read, err := grpcClient.ReadStream(ctx, stream, esdb.ReadStreamOptions{Direction: esdb.Backwards, From: esdb.End{}}, 1)
	if err == nil {
		defer read.Close()
		event, err := read.Recv()
		if err == nil {
			return StreamStats{
				EventStreamID:      stream,
				LastCommitPosition: -1,
				LastEventNumber:    int64(event.Event.EventNumber),
			}, nil
		}
	}

	log.WithFields(log.Fields{
		"streamId": stream,
		"error":    err,
	}).Error("Error when reading last event from stream")

	return StreamStats{}, err
}
