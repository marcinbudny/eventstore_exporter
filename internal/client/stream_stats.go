package client

import (
	"context"
	"sync"
	"time"

	"github.com/EventStore/EventStore-Client-Go/esdb"
	log "github.com/sirupsen/logrus"
)

type StreamStats struct {
	EventStreamID   string
	LastEventNumber uint64
}

type getStreamStatsResult struct {
	streams []StreamStats
	err     error
}

func (client *EventStoreStatsClient) getStreamStats() <-chan getStreamStatsResult {
	resultChan := make(chan getStreamStatsResult, 1)

	go func() {
		streamStats := make(chan StreamStats, len(client.config.Streams))
		grpcClient, err := client.getGrpcClient()
		if err != nil {
			resultChan <- getStreamStatsResult{err: err}
			return
		}

		var wg sync.WaitGroup

		for i, stream := range client.config.Streams {
			wg.Add(1)

			go func(i int, stream string) {
				defer wg.Done()

				if stats, getErr := getStreamStats(grpcClient, stream, client.config.Timeout); getErr == nil {
					streamStats <- stats
				}

			}(i, stream)
		}

		wg.Wait()
		close(streamStats)

		s := make([]StreamStats, 0)
		for i := range streamStats {
			s = append(s, i)
		}

		resultChan <- getStreamStatsResult{streams: s}
	}()

	return resultChan
}

func getStreamStats(grpcClient *esdb.Client, stream string, timeout time.Duration) (StreamStats, error) {
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
		event, err := read.Recv()
		if err == nil {
			return StreamStats{EventStreamID: "$all", LastEventNumber: event.Event.Position.Commit}, nil
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
		event, err := read.Recv()
		if err == nil {
			return StreamStats{EventStreamID: stream, LastEventNumber: event.Event.EventNumber}, nil
		}
	}

	log.WithFields(log.Fields{
		"streamId": stream,
		"error":    err,
	}).Error("Error when reading last event from stream")

	return StreamStats{}, err
}
