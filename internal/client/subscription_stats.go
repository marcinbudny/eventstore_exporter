package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/EventStore/EventStore-Client-Go/v3/esdb"
	log "github.com/sirupsen/logrus"
)

type SubscriptionStats struct {
	EventStreamID                   string `json:"eventStreamId"`
	GroupName                       string `json:"groupName"`
	TotalItemsProcessed             int64  `json:"totalItemsProcessed"`
	ConnectionCount                 int64  `json:"connectionCount"`
	LastKnownEventNumber            int64  `json:"lastKnownEventNumber"`
	LastProcessedEventNumber        int64  `json:"lastProcessedEventNumber"`
	LastCheckpointedEventPosition   string `json:"lastCheckpointedEventPosition"`
	LastKnownEventPosition          string `json:"lastKnownEventPosition"`
	TotalInFlightMessages           int64  `json:"totalInFlightMessages"`
	TotalNumberOfParkedMessages     int64
	OldestParkedMessageAgeInSeconds float64
}

func (client *EventStoreStatsClient) getSubscriptionStats(ctx context.Context) ([]SubscriptionStats, error) {
	subscriptions, err := esHTTPGetAndParse[[]SubscriptionStats](ctx, client, "/subscriptions", false)
	if err != nil {
		return nil, err
	}

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

			subscription.TotalNumberOfParkedMessages, subscription.OldestParkedMessageAgeInSeconds, _ =
				getParkedMessagesStats(ctx, grpcClient, subscription.EventStreamID, subscription.GroupName)
		}(&subscriptions[i])
	}

	wg.Wait()
}

func getParkedMessagesStats(ctx context.Context, grpc *esdb.Client, eventStreamID, groupName string) (numParked int64, oldestAgeInSec float64, err error) {
	oldestAgeInSec = -1

	parkedMessageFound, lastEventNumber, err := getParkedMessagesLastEventNumber(ctx, grpc, eventStreamID, groupName)

	if err != nil || !parkedMessageFound {
		return
	}

	truncateBeforeValue, err := getParkedMessagesTruncateBeforeValue(ctx, grpc, eventStreamID, groupName)

	if err != nil {
		return
	}

	totalNumberOfParkedMessages := lastEventNumber + 1 - truncateBeforeValue // +1 because ids start from 0

	if totalNumberOfParkedMessages > 0 {
		oldestMessagePosition := lastEventNumber + 1 - totalNumberOfParkedMessages
		oldestAgeInSec, _ = getOldestParkedMessageAgeInSeconds(ctx, grpc, eventStreamID, groupName, oldestMessagePosition)
	}

	numParked = int64(totalNumberOfParkedMessages)

	return
}

func getOldestParkedMessageAgeInSeconds(ctx context.Context, grpcClient *esdb.Client, eventStreamID string, groupName string, oldestMessagePosition uint64) (float64, error) {
	event, err := readSingleEvent(ctx, grpcClient, parkedStreamID(eventStreamID, groupName), esdb.ReadStreamOptions{Direction: esdb.Forwards, From: esdb.Revision(oldestMessagePosition)})

	if errors.Is(err, io.EOF) {
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

	if errors.Is(err, io.EOF) {
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
