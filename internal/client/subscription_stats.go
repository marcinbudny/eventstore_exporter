package client

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

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
	TotalItemsProcessed             float64
	ConnectionCount                 float64
	LastKnownEventNumber            float64
	LastProcessedEventNumber        float64
	TotalInFlightMessages           float64
	TotalNumberOfParkedMessages     float64
	OldestParkedMessageAgeInSeconds float64
}

func (client *EventStoreStatsClient) getSubscriptionStats(esVersion EventStoreVersion) <-chan getSubscriptionStatsResult {
	stats := make(chan (getSubscriptionStatsResult), 1)

	go func() {
		subscriptionsJson, err := client.get("/subscriptions", false)
		if err != nil {
			stats <- getSubscriptionStatsResult{err: err}
		}

		atomPubEnabled := true
		if _, err := client.get("/streams/$all/head/backward/1", false); err != nil {
			atomPubEnabled = false
		}

		subscriptions := getSubscriptions(subscriptionsJson, esVersion)

		if client.config.EnableParkedMessagesStats {
			client.addParkedMessagesStats(subscriptions, esVersion, atomPubEnabled)
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

func getSubscriptions(subscriptionsJson []byte, esVersion EventStoreVersion) []SubscriptionStats {
	subscriptions := []SubscriptionStats{}

	jp.ArrayEach(subscriptionsJson, func(jsonValue []byte, dataType jp.ValueType, offset int, err error) {
		subscriptions = append(subscriptions, SubscriptionStats{
			EventStreamID:            getString(jsonValue, "eventStreamId"),
			GroupName:                getString(jsonValue, "groupName"),
			TotalItemsProcessed:      getFloat(jsonValue, "totalItemsProcessed"),
			ConnectionCount:          getFloat(jsonValue, "connectionCount"),
			LastKnownEventNumber:     getFloat(jsonValue, "lastKnownEventNumber"),
			LastProcessedEventNumber: getFloat(jsonValue, "lastProcessedEventNumber"),
			TotalInFlightMessages:    getFloat(jsonValue, "totalInFlightMessages"),
		})
	})

	return subscriptions
}

func (client *EventStoreStatsClient) addParkedMessagesStats(subscriptions []SubscriptionStats, esVersion EventStoreVersion, atomPubEnabled bool) {

	if atomPubEnabled {
		log.Debug("Detected Atom Pub to be available, getting parked message stats via Atom Pub")

		client.addParkedMessagesStatsViaAtomPub(subscriptions)

	} else if esVersion.ReportsParkedMessageNumber() {
		log.Debug("Detected Atom Pub to be unavailable, getting limited paked message stats from group info endpoint")

		client.addParkedMessagesStatsViaGroupInfo(subscriptions)

	} else {
		log.Error("Atom Pub is disabled and ES version is < 21.2, there is no way to retrieve parmed message stats")
	}
}

func (client *EventStoreStatsClient) addParkedMessagesStatsViaGroupInfo(subscriptions []SubscriptionStats) {
	var wg sync.WaitGroup

	for i := range subscriptions {
		wg.Add(1)

		go func(subscription *SubscriptionStats) {
			defer wg.Done()

			groupInfoURL := fmt.Sprintf("/subscriptions/%s/%s/info", subscription.EventStreamID, subscription.GroupName)
			if groupInfoJson, err := client.get(groupInfoURL, false); err == nil {
				subscription.TotalNumberOfParkedMessages = getFloat(groupInfoJson, "parkedMessageCount")
				subscription.OldestParkedMessageAgeInSeconds = -1
			} else {
				log.WithFields(log.Fields{
					"eventStreamId": subscription.EventStreamID,
					"groupName":     subscription.GroupName,
					"error":         err,
				}).Error("Error when getting subscription group info")

			}
		}(&subscriptions[i])
	}

	wg.Wait()
}

func (client *EventStoreStatsClient) addParkedMessagesStatsViaAtomPub(subscriptions []SubscriptionStats) {

	var wg sync.WaitGroup

	for i := range subscriptions {
		wg.Add(1)

		go func(subscription *SubscriptionStats) {
			defer wg.Done()

			lastEventNumber, err := client.getParkedMessagesLastEventNumber(subscription.EventStreamID, subscription.GroupName)

			if err != nil || lastEventNumber == 0 {
				return
			}

			truncateBeforeValue, err := client.getParkedMessagesTruncateBeforeValue(subscription.EventStreamID, subscription.GroupName)

			if err != nil {
				return
			}

			totalNumberOfParkedMessages := lastEventNumber - truncateBeforeValue

			var oldestParkedMessageAgeInSeconds float64 = 0
			if totalNumberOfParkedMessages > 0 {
				oldestMessageID := lastEventNumber - totalNumberOfParkedMessages
				oldestParkedMessageAgeInSeconds, _ = client.getOldestParkedMessageTimeInSeconds(subscription.EventStreamID, subscription.GroupName, oldestMessageID)
			}

			subscription.TotalNumberOfParkedMessages = float64(totalNumberOfParkedMessages)
			subscription.OldestParkedMessageAgeInSeconds = oldestParkedMessageAgeInSeconds

		}(&subscriptions[i])
	}

	wg.Wait()
}

func (client *EventStoreStatsClient) getOldestParkedMessageTimeInSeconds(eventStreamID string, groupName string, oldestMessageID int64) (float64, error) {
	getOldestMessageURL := fmt.Sprintf("/streams/$persistentsubscription-%s::%s-parked/%s/forward/1", eventStreamID, groupName, strconv.FormatInt(oldestMessageID, 10))
	oldestMessageJson, err := client.get(getOldestMessageURL, false)

	if err != nil {
		log.WithFields(log.Fields{
			"eventStreamId": eventStreamID,
			"groupName":     groupName,
			"error":         err,
		}).Error("Error while getting oldest parked message.")
		return 0, err
	}

	oldestMessageUpdatedDateResult := ""
	jp.ArrayEach(oldestMessageJson, func(value []byte, dataType jp.ValueType, offset int, err error) {
		oldestMessageUpdatedDateResult = getString(value, "updated")
	}, "entries")

	loc, _ := time.LoadLocation("UTC")
	timeNow := time.Now().In(loc)
	oldestMessageUpdatedDate, err := time.Parse(time.RFC3339Nano, oldestMessageUpdatedDateResult)

	if err != nil {
		log.WithFields(log.Fields{
			"eventStreamId": eventStreamID,
			"groupName":     groupName,
			"timeString":    oldestMessageUpdatedDateResult,
			"error":         err,
		}).Error("Cannot parse update time on the oldest parked message.")
		return 0, err
	}

	timeInSeconds := float64(timeNow.Sub(oldestMessageUpdatedDate) / time.Second)

	return timeInSeconds, nil
}

func (client *EventStoreStatsClient) getParkedMessagesLastEventNumber(eventStreamID string, groupName string) (int64, error) {
	parkedMessagesURL := fmt.Sprintf("/streams/$persistentsubscription-%s::%s-parked/head/backward/1", eventStreamID, groupName)
	parkedMessagesJson, err := client.get(parkedMessagesURL, true)

	if err != nil {
		log.WithFields(log.Fields{
			"eventStreamId": eventStreamID,
			"groupName":     groupName,
			"error":         err,
		}).Error("Error when getting parked messages stream.")
		return 0, err
	}

	if parkedMessagesJson == nil {
		return 0, nil
	}

	eTagString := getString(parkedMessagesJson, "eTag")

	lastEventNumber, err := strconv.ParseInt(strings.Split(eTagString, ";")[0], 10, 64)

	if err != nil {
		log.WithFields(log.Fields{
			"eventStreamId": eventStreamID,
			"groupName":     groupName,
			"eTagString":    eTagString,
			"error":         err,
		}).Error("Cannot parse eTag on parked messages stream.")
		return 0, err
	}

	lastEventNumber++ // +1 because Ids start from 0

	return lastEventNumber, nil
}

func (client *EventStoreStatsClient) getParkedMessagesTruncateBeforeValue(eventStreamID string, groupName string) (int64, error) {
	metadataURL := fmt.Sprintf("/streams/$persistentsubscription-%s::%s-parked/metadata", eventStreamID, groupName)
	metadataJson, err := client.get(metadataURL, false)

	if err != nil {
		log.WithFields(log.Fields{
			"eventStreamId": eventStreamID,
			"groupName":     groupName,
			"error":         err,
		}).Error("Error when getting parked message stream metadata")
		return 0, err
	}

	truncateBeforeValue, err := jp.GetInt(metadataJson, "$tb")

	if err != nil {
		log.WithFields(log.Fields{
			"eventStreamId": eventStreamID,
			"groupName":     groupName,
		}).Debug("Parked messages have not been replayed yet, as $tb value does not exist in the metadata. Defaulting to 0.")
		return 0, nil
	}

	return truncateBeforeValue, nil
}
