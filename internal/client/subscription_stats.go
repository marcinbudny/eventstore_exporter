package client

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/EventStore/EventStore-Client-Go/v3/esdb"
	jp "github.com/buger/jsonparser"
	log "github.com/sirupsen/logrus"
)

type SubscriptionStats struct {
	EventStreamID                   string
	GroupName                       string
	TotalItemsProcessed             int64
	ConnectionCount                 int64
	LastKnownEventNumber            int64
	LastProcessedEventNumber        int64
	LastCheckpointedEventPosition   int64
	LastKnownEventPosition          int64
	TotalInFlightMessages           int64
	TotalNumberOfParkedMessages     int64
	OldestParkedMessageAgeInSeconds float64
}

func (client *EventStoreStatsClient) getSubscriptionStats(ctx context.Context) ([]SubscriptionStats, error) {
	subscriptionsJson, err := client.esHttpGet(ctx, "/subscriptions", false)
	if err != nil {
		return nil, err
	}

	subscriptions := getSubscriptions(subscriptionsJson)

	if client.config.EnableParkedMessagesStats {
		client.addParkedMessagesStats(ctx, subscriptions)
	} else {
		markParkedMessageStatsAsUnavailable(subscriptions)
	}

	return subscriptions, nil
}

func markParkedMessageStatsAsUnavailable(subscriptions []SubscriptionStats) {
	for i := range subscriptions {
		subscriptions[i].TotalNumberOfParkedMessages = -1
		subscriptions[i].OldestParkedMessageAgeInSeconds = -1
	}
}

func getSubscriptions(subscriptionsJson []byte) []SubscriptionStats {
	subscriptions := []SubscriptionStats{}

	jp.ArrayEach(subscriptionsJson, func(jsonValue []byte, dataType jp.ValueType, offset int, err error) {
		stats := SubscriptionStats{
			EventStreamID:            getString(jsonValue, "eventStreamId"),
			GroupName:                getString(jsonValue, "groupName"),
			TotalItemsProcessed:      getInt(jsonValue, "totalItemsProcessed"),
			ConnectionCount:          getInt(jsonValue, "connectionCount"),
			LastKnownEventNumber:     getInt(jsonValue, "lastKnownEventNumber"),
			LastProcessedEventNumber: getInt(jsonValue, "lastProcessedEventNumber"),
			TotalInFlightMessages:    getInt(jsonValue, "totalInFlightMessages"),
		}

		if stats.EventStreamID == "$all" {
			stats.LastKnownEventNumber = -1
			stats.LastProcessedEventNumber = -1

			if lastCheckpointedEventPosition, err := parseEventPosition(getString(jsonValue, "lastCheckpointedEventPosition")); err != nil {
				log.WithError(err).Errorf("Error while parsing last checkpointed event position of $all stream subscription group %s", stats.GroupName)
			} else {
				stats.LastCheckpointedEventPosition = lastCheckpointedEventPosition
			}

			if lastKnownEventPosition, err := parseEventPosition(getString(jsonValue, "lastKnownEventPosition")); err != nil {
				log.WithError(err).Errorf("Error while parsing last known even position of $all stream subscription group %s", stats.GroupName)
			} else {
				stats.LastKnownEventPosition = lastKnownEventPosition
			}
		} else {
			stats.LastKnownEventNumber = getInt(jsonValue, "lastKnownEventNumber")
			stats.LastProcessedEventNumber = getInt(jsonValue, "lastProcessedEventNumber")
			stats.LastCheckpointedEventPosition = -1
			stats.LastKnownEventPosition = -1
		}

		subscriptions = append(subscriptions, stats)
	})

	return subscriptions
}

func (client *EventStoreStatsClient) addParkedMessagesStats(ctx context.Context, subscriptions []SubscriptionStats) {
	if len(subscriptions) == 0 {
		return
	}

	grpcClient, err := client.getGrpcClient()

	if err != nil {
		log.WithError(err).Error("Error when creating grpc client")
	}
	defer grpcClient.Close()

	var wg sync.WaitGroup

	for i := range subscriptions {
		wg.Add(1)

		go func(subscription *SubscriptionStats) {
			defer wg.Done()

			log.WithField("eventStreamId", subscription.EventStreamID).WithField("groupName", subscription.GroupName).Debug("Getting subscription parked message stats")

			subscription.OldestParkedMessageAgeInSeconds = -1

			parkedMessageFound, lastEventNumber, err := getParkedMessagesLastEventNumber(ctx, grpcClient, subscription.EventStreamID, subscription.GroupName)

			if err != nil || !parkedMessageFound {
				return
			}

			truncateBeforeValue, err := getParkedMessagesTruncateBeforeValue(ctx, grpcClient, subscription.EventStreamID, subscription.GroupName)

			if err != nil {
				return
			}

			totalNumberOfParkedMessages := lastEventNumber + 1 - truncateBeforeValue // +1 because ids start from 0

			var oldestParkedMessageAgeInSeconds float64 = 0
			if totalNumberOfParkedMessages > 0 {
				oldestMessagePosition := lastEventNumber + 1 - totalNumberOfParkedMessages
				oldestParkedMessageAgeInSeconds, _ = getOldestParkedMessageAgeInSeconds(ctx, grpcClient, subscription.EventStreamID, subscription.GroupName, oldestMessagePosition)
			}

			subscription.TotalNumberOfParkedMessages = int64(totalNumberOfParkedMessages)
			subscription.OldestParkedMessageAgeInSeconds = oldestParkedMessageAgeInSeconds

		}(&subscriptions[i])
	}

	wg.Wait()
}

func getOldestParkedMessageAgeInSeconds(ctx context.Context, grpcClient *esdb.Client, eventStreamID string, groupName string, oldestMessagePosition uint64) (float64, error) {
	event, err := readSingleEvent(ctx, grpcClient, parkedStreamID(eventStreamID, groupName), esdb.ReadStreamOptions{Direction: esdb.Forwards, From: esdb.Revision(oldestMessagePosition)})

	if err == io.EOF {
		return -1, nil
	} else if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"eventStreamId": eventStreamID,
			"groupName":     groupName,
		}).Error("Error when getting parked messages stream.")

		return 0, err
	}

	created := event.Event.CreatedDate
	loc, _ := time.LoadLocation("UTC")
	timeNow := time.Now().In(loc)
	age := float64(timeNow.Sub(created) / time.Second)

	return age, nil
}

func getParkedMessagesLastEventNumber(ctx context.Context, grpcClient *esdb.Client, eventStreamID string, groupName string) (bool, uint64, error) {
	event, err := readSingleEvent(ctx, grpcClient, parkedStreamID(eventStreamID, groupName), esdb.ReadStreamOptions{Direction: esdb.Backwards, From: esdb.End{}})

	if err == io.EOF {
		return false, 0, nil
	} else if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"eventStreamId": eventStreamID,
			"groupName":     groupName,
		}).Error("Error when getting parked messages stream.")

		return false, 0, err
	}

	return true, event.Event.EventNumber, nil
}

func getParkedMessagesTruncateBeforeValue(ctx context.Context, grpcClient *esdb.Client, eventStreamID string, groupName string) (uint64, error) {
	if meta, err := grpcClient.GetStreamMetadata(ctx, parkedStreamID(eventStreamID, groupName), esdb.ReadStreamOptions{}); err == nil {
		return *meta.TruncateBefore(), nil
	} else if strings.Contains(err.Error(), "not found") {
		return 0, nil
	} else {
		log.WithError(err).WithFields(log.Fields{
			"eventStreamId": eventStreamID,
			"groupName":     groupName,
		}).Error("Error when getting parked message stream metadata")

		return 0, err

	}
}

func parkedStreamID(eventStreamID string, groupName string) string {
	return fmt.Sprintf("$persistentsubscription-%s::%s-parked", eventStreamID, groupName)
}

// extracts commit position from strings like "C:1234/P:5678"
func parseEventPosition(position string) (int64, error) {
	if position == "" {
		return -1, fmt.Errorf("empty position")
	}

	parts := strings.Split(position, "/")
	if len(parts) != 2 {
		return -1, fmt.Errorf("invalid event position: %s", position)
	}

	commitPosition, err := strconv.ParseInt(parts[0][2:], 10, 64)
	if err != nil {
		return -1, fmt.Errorf("invalid commit position in event position: %s", position)
	}

	return commitPosition, nil
}
