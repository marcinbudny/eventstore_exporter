package client

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/EventStore/EventStore-Client-Go/esdb"
	jp "github.com/buger/jsonparser"
	log "github.com/sirupsen/logrus"
)

type getSubscriptionStatsResult struct {
	subscriptions []SubscriptionStats
	err           error
}

type SubscriptionStats struct {
	EventStreamID                   string
	GroupName                       string
	TotalItemsProcessed             int64
	ConnectionCount                 int64
	LastKnownEventNumber            int64
	LastProcessedEventNumber        int64
	TotalInFlightMessages           int64
	TotalNumberOfParkedMessages     int64
	OldestParkedMessageAgeInSeconds float64
}

func (client *EventStoreStatsClient) getSubscriptionStats() <-chan getSubscriptionStatsResult {
	stats := make(chan (getSubscriptionStatsResult), 1)

	go func() {
		subscriptionsJson, err := client.esHttpGet("/subscriptions", false)
		if err != nil {
			stats <- getSubscriptionStatsResult{err: err}
		}

		subscriptions := getSubscriptions(subscriptionsJson)

		if client.config.EnableParkedMessagesStats {
			client.addParkedMessagesStats(subscriptions)
		} else {
			markParkedMessageStatsAsUnavailable(subscriptions)
		}

		stats <- getSubscriptionStatsResult{
			subscriptions: subscriptions,
		}

	}()

	return stats
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
		subscriptions = append(subscriptions, SubscriptionStats{
			EventStreamID:            getString(jsonValue, "eventStreamId"),
			GroupName:                getString(jsonValue, "groupName"),
			TotalItemsProcessed:      getInt(jsonValue, "totalItemsProcessed"),
			ConnectionCount:          getInt(jsonValue, "connectionCount"),
			LastKnownEventNumber:     getInt(jsonValue, "lastKnownEventNumber"),
			LastProcessedEventNumber: getInt(jsonValue, "lastProcessedEventNumber"),
			TotalInFlightMessages:    getInt(jsonValue, "totalInFlightMessages"),
		})
	})

	return subscriptions
}

func (client *EventStoreStatsClient) addParkedMessagesStats(subscriptions []SubscriptionStats) {
	grpcClient, err := client.getGrpcClient()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Error when creating grpc client")
	}
	defer grpcClient.Close()

	var wg sync.WaitGroup

	for i := range subscriptions {
		wg.Add(1)

		go func(subscription *SubscriptionStats) {
			defer wg.Done()

			subscription.OldestParkedMessageAgeInSeconds = -1

			parkedMessageFound, lastEventNumber, err := getParkedMessagesLastEventNumber(grpcClient, subscription.EventStreamID, subscription.GroupName, client.config.Timeout)

			if err != nil || !parkedMessageFound {
				return
			}

			truncateBeforeValue, err := getParkedMessagesTruncateBeforeValue(grpcClient, subscription.EventStreamID, subscription.GroupName, client.config.Timeout)

			if err != nil {
				return
			}

			totalNumberOfParkedMessages := lastEventNumber + 1 - truncateBeforeValue // +1 becaues ids start from 0

			var oldestParkedMessageAgeInSeconds float64 = 0
			if totalNumberOfParkedMessages > 0 {
				oldestMessagePosition := lastEventNumber + 1 - totalNumberOfParkedMessages
				oldestParkedMessageAgeInSeconds, _ = getOldestParkedMessageAgeInSeconds(grpcClient, subscription.EventStreamID, subscription.GroupName, oldestMessagePosition, client.config.Timeout)
			}

			subscription.TotalNumberOfParkedMessages = int64(totalNumberOfParkedMessages)
			subscription.OldestParkedMessageAgeInSeconds = oldestParkedMessageAgeInSeconds

		}(&subscriptions[i])
	}

	wg.Wait()
}

func getOldestParkedMessageAgeInSeconds(grpcClient *esdb.Client, eventStreamID string, groupName string, oldestMessagePosition uint64, timeout time.Duration) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	read, err := grpcClient.ReadStream(ctx, parkedStreamID(eventStreamID, groupName), esdb.ReadStreamOptions{Direction: esdb.Forwards, From: esdb.Revision(oldestMessagePosition)}, 1)
	if err == nil {
		event, err := read.Recv()
		if err == nil {
			created := event.Event.CreatedDate
			loc, _ := time.LoadLocation("UTC")
			timeNow := time.Now().In(loc)
			age := float64(timeNow.Sub(created) / time.Second)

			return age, nil
		}
	}

	log.WithFields(log.Fields{
		"eventStreamId": eventStreamID,
		"groupName":     groupName,
		"error":         err,
	}).Error("Error when getting parked messages stream.")

	return 0, err
}

func getParkedMessagesLastEventNumber(grpcClient *esdb.Client, eventStreamID string, groupName string, timeout time.Duration) (bool, uint64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	read, err := grpcClient.ReadStream(ctx, parkedStreamID(eventStreamID, groupName), esdb.ReadStreamOptions{Direction: esdb.Backwards, From: esdb.End{}}, 1)
	if err == nil {
		event, err := read.Recv()
		if err == nil {
			return true, event.Event.EventNumber, nil
		}
	}

	log.WithFields(log.Fields{
		"eventStreamId": eventStreamID,
		"groupName":     groupName,
		"error":         err,
	}).Error("Error when getting parked messages stream.")

	return false, 0, err
}

func getParkedMessagesTruncateBeforeValue(grpcClient *esdb.Client, eventStreamID string, groupName string, timeout time.Duration) (uint64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if meta, err := grpcClient.GetStreamMetadata(ctx, parkedStreamID(eventStreamID, groupName), esdb.ReadStreamOptions{}); err == nil {
		return *meta.TruncateBefore(), nil
	} else if strings.Contains(err.Error(), "not found") {
		return 0, nil
	} else {
		log.WithFields(log.Fields{
			"eventStreamId": eventStreamID,
			"groupName":     groupName,
			"error":         err,
		}).Error("Error when getting parked message stream metadata")

		return 0, err

	}
}

func parkedStreamID(eventStreamID string, groupName string) string {
	return fmt.Sprintf("$persistentsubscription-%s::%s-parked", eventStreamID, groupName)
}
