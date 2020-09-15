package main

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/buger/jsonparser"
	jp "github.com/buger/jsonparser"
)

var (
	client http.Client
)

func initializeClient() {
	if insecureSkipVerify {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client = http.Client{
			Timeout:   timeout,
			Transport: tr,
		}
	} else {
		client = http.Client{
			Timeout: timeout,
		}
	}
}

type getResult struct {
	result []byte
	err    error
}

type stats struct {
	serverStats         []byte
	gossipStats         []byte
	projectionStats     []byte
	info                []byte
	subscriptionsStats  []byte
	parkedMessagesStats []parkedMessagesStats
}

type parkedMessagesStats struct {
	eventStreamID                   string
	groupName                       string
	totalNumberOfParkedMessages     float64
	oldestParkedMessageAgeInSeconds float64
}

func getStats() (*stats, error) {
	serverStatsChan := get("/stats", false)
	projectionStatsChan := get("/projections/all-non-transient", true)
	infoChan := get("/info", false)
	subscriptionsStatsChan := get("/subscriptions", false)

	serverStatsResult := <-serverStatsChan
	if serverStatsResult.err != nil {
		return nil, serverStatsResult.err
	}

	projectionStatsResult := <-projectionStatsChan
	if projectionStatsResult.err != nil {
		return nil, projectionStatsResult.err
	}

	infoResult := <-infoChan
	if infoResult.err != nil {
		return nil, infoResult.err
	}

	subscriptionsStatsResult := <-subscriptionsStatsChan
	if subscriptionsStatsResult.err != nil {
		return nil, subscriptionsStatsResult.err
	}

	gossipStatsResult := getResult{}
	if isInClusterMode() {
		gossipStatsChan := get("/gossip", false)

		gossipStatsResult = <-gossipStatsChan
		if gossipStatsResult.err != nil {
			return nil, gossipStatsResult.err
		}
	}

	var parkedMessagesStatsResult []parkedMessagesStats
	if enableParkedMessagesStats {
		parkedMessagesStats, err := getSubscriptionParkedMessagesStats(subscriptionsStatsResult.result)
		if err != nil {
			log.WithError(err).Error("Error while getting parked messages for subscriptions.")
		} else {
			parkedMessagesStatsResult = *parkedMessagesStats
		}
	}

	return &stats{
		serverStatsResult.result,
		gossipStatsResult.result,
		projectionStatsResult.result,
		infoResult.result,
		subscriptionsStatsResult.result,
		parkedMessagesStatsResult,
	}, nil
}

func getSubscriptionParkedMessagesStats(subscriptions []byte) (*[]parkedMessagesStats, error) {
	var result []parkedMessagesStats
	jp.ArrayEach(subscriptions, func(jsonValue []byte, dataType jp.ValueType, offset int, err error) {
		eventStreamID, _ := jp.GetString(jsonValue, "eventStreamId")
		groupName, _ := jp.GetString(jsonValue, "groupName")

		lastEventNumber, err := getParkedMessagesLastEventNumber(eventStreamID, groupName)

		if err != nil {
			log.WithFields(logrus.Fields{
				"eventStreamId": eventStreamID,
				"groupName":     groupName,
				"err":           err,
			}).Warn("Error while getting parked messages last event number.")
		}

		truncateBeforeValue, err := getParkedMessagesTruncateBeforeValue(eventStreamID, groupName, lastEventNumber)

		if err != nil {
			log.WithFields(logrus.Fields{
				"eventStreamId": eventStreamID,
				"groupName":     groupName,
				"error":         err,
			}).Warn("Error while getting parked messages truncate before value.")
		}

		totalNumberOfParkedMessages := lastEventNumber - truncateBeforeValue

		var oldestParkedMessage float64
		if true {
			oldestMessageID := lastEventNumber - totalNumberOfParkedMessages
			oldestParkedMessage, err = getOldestParkedMessageTimeInSeconds(eventStreamID, groupName, oldestMessageID)
			if err != nil {
				log.WithFields(logrus.Fields{
					"eventStreamId": eventStreamID,
					"groupName":     groupName,
					"error":         err,
				}).Warn("Error while getting oldest parked message.")
			}
		}

		result = append(result, parkedMessagesStats{
			eventStreamID:                   eventStreamID,
			groupName:                       groupName,
			totalNumberOfParkedMessages:     float64(totalNumberOfParkedMessages),
			oldestParkedMessageAgeInSeconds: oldestParkedMessage})
	})
	return &result, nil
}

func getOldestParkedMessageTimeInSeconds(eventStreamID string, groupName string, oldestMessageID int64) (float64, error) {
	getOldestMessageURL := fmt.Sprintf("/streams/$persistentsubscription-%s::%s-parked/%s/forward/1", eventStreamID, groupName, strconv.FormatInt(oldestMessageID, 10))
	getOldestMessageResultChan := get(getOldestMessageURL, false)
	getOldestMessageResult := <-getOldestMessageResultChan
	if getOldestMessageResult.err != nil {
		return 0, getOldestMessageResult.err
	}
	oldestMessageUpdatedDateResult := ""
	jsonparser.ArrayEach(getOldestMessageResult.result, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		oldestMessageUpdatedDateResult, _ = jp.GetString(value, "updated")
	}, "entries")

	loc, _ := time.LoadLocation("UTC")
	timeNow := time.Now().In(loc)
	oldestMessageUpdatedDate, err := time.Parse(time.RFC3339Nano, oldestMessageUpdatedDateResult)

	if err != nil {
		return 0, err
	}

	timeInSeconds := float64(timeNow.Sub(oldestMessageUpdatedDate) / time.Second)

	return timeInSeconds, nil
}

func getParkedMessagesLastEventNumber(eventStreamID string, groupName string) (int64, error) {
	parkedMessagesURL := fmt.Sprintf("/streams/$persistentsubscription-%s::%s-parked/head/backward/1", eventStreamID, groupName)
	parkedMessagesResultChan := get(parkedMessagesURL, false)
	parkedMessagesResult := <-parkedMessagesResultChan

	if parkedMessagesResult.err != nil {
		return 0, parkedMessagesResult.err
	}

	eTagString, _ := jp.GetString(parkedMessagesResult.result, "eTag")

	lastEventNumber, err := strconv.ParseInt(strings.Split(eTagString, ";")[0], 10, 64)

	if err != nil {
		return 0, err
	}

	lastEventNumber++ // +1 because Ids start from 0

	return lastEventNumber, nil
}

func getParkedMessagesTruncateBeforeValue(eventStreamID string, groupName string, lastEventNumber int64) (int64, error) {
	metadataURL := fmt.Sprintf("/streams/$persistentsubscription-%s::%s-parked/metadata", eventStreamID, groupName)
	metadataResultChan := get(metadataURL, false)
	metadataResult := <-metadataResultChan

	if metadataResult.err != nil {
		return 0, metadataResult.err
	}

	truncateBeforeValue, err := jp.GetInt(metadataResult.result, "$tb")

	if err != nil {
		return 0, err
	}

	return truncateBeforeValue, nil
}

func get(path string, acceptNotFound bool) <-chan getResult {
	url := eventStoreURL + path

	result := make(chan getResult)

	go func() {
		log.WithField("url", url).Debug("GET request to EventStore")

		req, err := http.NewRequest("GET", url, nil)
		if eventStoreUser != "" && eventStorePassword != "" {
			req.SetBasicAuth(eventStoreUser, eventStorePassword)
		}
		req.Header.Add("Accept", "application/json")
		response, err := client.Do(req)
		if err != nil {
			result <- getResult{nil, err}
			return
		}
		defer response.Body.Close()

		if response.StatusCode == 404 && acceptNotFound {
			result <- getResult{nil, nil}
		}

		if response.StatusCode >= 400 {
			result <- getResult{nil, fmt.Errorf("HTTP call to %s resulted in status code %d", url, response.StatusCode)}
		}

		buf, err := ioutil.ReadAll(response.Body)
		if err != nil {
			result <- getResult{nil, err}
			return
		}

		result <- getResult{buf, nil}
	}()

	return result
}
