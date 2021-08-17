package client

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"

	jp "github.com/buger/jsonparser"
	"github.com/marcinbudny/eventstore_exporter/config"
)

type EventStoreStatsClient struct {
	httpClient http.Client
	config     *config.Config
}

func New(config *config.Config) *EventStoreStatsClient {
	esClient := &EventStoreStatsClient{}
	esClient.config = config

	if config.InsecureSkipVerify {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		esClient.httpClient = http.Client{
			Timeout:   config.Timeout,
			Transport: tr,
		}
	} else {
		esClient.httpClient = http.Client{
			Timeout: config.Timeout,
		}
	}

	return esClient
}

type getResult struct {
	result []byte
	err    error
}

type Stats struct {
	EsVersion           EventStoreVersion
	AtomPubEnabled      bool
	ServerStats         []byte
	GossipStats         []byte
	ProjectionStats     []byte
	Info                []byte
	SubscriptionsStats  []byte
	ParkedMessagesStats []ParkedMessagesStats
}

type ParkedMessagesStats struct {
	EventStreamID                   string
	GroupName                       string
	TotalNumberOfParkedMessages     float64
	OldestParkedMessageAgeInSeconds float64
}

type EventStoreVersion string

func (esClient *EventStoreStatsClient) GetStats() (*Stats, error) {
	serverStatsChan := esClient.get("/stats", false)
	projectionStatsChan := esClient.get("/projections/all-non-transient", true)
	infoChan := esClient.get("/info", false)
	subscriptionsStatsChan := esClient.get("/subscriptions", false)
	allStreamChan := esClient.get("/streams/$all/head/backward/1", false)

	atomPubEnabled := false
	allStreamResult := <-allStreamChan
	if allStreamResult.err == nil {
		atomPubEnabled = true
	}

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
	esVersion := getEsVersion(infoResult.result)

	subscriptionsStatsResult := <-subscriptionsStatsChan
	if subscriptionsStatsResult.err != nil {
		return nil, subscriptionsStatsResult.err
	}

	gossipStatsResult := getResult{}
	if esClient.config.IsInClusterMode() {
		gossipStatsChan := esClient.get("/gossip", false)

		gossipStatsResult = <-gossipStatsChan
		if gossipStatsResult.err != nil {
			return nil, gossipStatsResult.err
		}
	}

	var parkedMessagesStatsResult []ParkedMessagesStats
	if esClient.config.EnableParkedMessagesStats {

		if atomPubEnabled {
			log.Debug("Detected Atom Pub to be available, getting subscription stats via Atom Pub")

			parkedMessagesStats, err := esClient.getParkedMessagesStatsViaAtomPub(subscriptionsStatsResult.result)
			if err != nil {
				log.WithError(err).Error("Error while getting parked messages for subscriptions via Atom Pub")
			} else {
				parkedMessagesStatsResult = *parkedMessagesStats
			}
		} else if esVersion.ReportsParkedMessageNumber() {
			log.Debug("Detected Atom Pub to be unavailable, getting limited subscription stats from group info endpoint")
			parkedMessageStats, err := esClient.getParkedMessagesStatsViaGroupInfo(subscriptionsStatsResult.result)
			if err != nil {
				log.WithError(err).Error("Error while getting parked messages fro subscriptions via group info endpoint")
			} else {
				parkedMessagesStatsResult = *parkedMessageStats
			}

		} else {
			log.Error("Atom Pub is disabled and ES version is < 21.2, there is no way to retrieve subscription stats")
		}
	}

	return &Stats{
		esVersion,
		atomPubEnabled,
		serverStatsResult.result,
		gossipStatsResult.result,
		projectionStatsResult.result,
		infoResult.result,
		subscriptionsStatsResult.result,
		parkedMessagesStatsResult,
	}, nil
}

func getEsVersion(info []byte) EventStoreVersion {
	value, _ := jp.GetString(info, "esVersion")
	if value == "" {
		value = "0.0.0.0"
	}
	return EventStoreVersion(value)
}

func (esVersion EventStoreVersion) ReportsParkedMessageNumber() bool {
	return esVersion.IsAtLeastVersion("21.2.0.0")
}

func (esVersion EventStoreVersion) UsesLeaderFollowerNomenclature() bool {
	return esVersion.IsAtLeastVersion("20.6.0.0")
}

func (esVersion EventStoreVersion) UsesHttpEndPointNomenclature() bool {
	return esVersion.IsAtLeastVersion("20.6.0.0")
}

func (esVersion EventStoreVersion) ReportsCpuScaled() bool {
	return esVersion.IsVersionLowerThan("20.6.0.0")
}

func (esVersion EventStoreVersion) IsAtLeastVersion(minVersion string) bool {
	ver, err := version.NewVersion(string(esVersion))
	if err != nil {
		return false
	}
	minSupportedVersion, err := version.NewVersion(minVersion)
	if err != nil {
		return false
	}

	return ver.GreaterThanOrEqual(minSupportedVersion)
}

func (esVersion EventStoreVersion) IsVersionLowerThan(maxVersion string) bool {
	ver, err := version.NewVersion(string(esVersion))
	if err != nil {
		return false
	}
	minSupportedVersion, err := version.NewVersion(maxVersion)
	if err != nil {
		return false
	}

	return ver.LessThan(minSupportedVersion)
}

func (esClient *EventStoreStatsClient) getParkedMessagesStatsViaAtomPub(subscriptions []byte) (*[]ParkedMessagesStats, error) {
	var result []ParkedMessagesStats
	jp.ArrayEach(subscriptions, func(jsonValue []byte, dataType jp.ValueType, offset int, e error) {
		eventStreamID, _ := jp.GetString(jsonValue, "eventStreamId")
		groupName, _ := jp.GetString(jsonValue, "groupName")

		lastEventNumber, err := esClient.getParkedMessagesLastEventNumber(eventStreamID, groupName)

		if err != nil || lastEventNumber == 0 {
			return
		}

		truncateBeforeValue, err := esClient.getParkedMessagesTruncateBeforeValue(eventStreamID, groupName, lastEventNumber)

		if err != nil {
			return
		}

		totalNumberOfParkedMessages := lastEventNumber - truncateBeforeValue

		var oldestParkedMessage float64 = 0
		if totalNumberOfParkedMessages > 0 {
			oldestMessageID := lastEventNumber - totalNumberOfParkedMessages
			oldestParkedMessage, _ = esClient.getOldestParkedMessageTimeInSeconds(eventStreamID, groupName, oldestMessageID)
		}

		result = append(result, ParkedMessagesStats{
			EventStreamID:                   eventStreamID,
			GroupName:                       groupName,
			TotalNumberOfParkedMessages:     float64(totalNumberOfParkedMessages),
			OldestParkedMessageAgeInSeconds: oldestParkedMessage})
	})
	return &result, nil
}

func (esClient *EventStoreStatsClient) getOldestParkedMessageTimeInSeconds(eventStreamID string, groupName string, oldestMessageID int64) (float64, error) {
	getOldestMessageURL := fmt.Sprintf("/streams/$persistentsubscription-%s::%s-parked/%s/forward/1", eventStreamID, groupName, strconv.FormatInt(oldestMessageID, 10))
	getOldestMessageResultChan := esClient.get(getOldestMessageURL, false)
	getOldestMessageResult := <-getOldestMessageResultChan

	if getOldestMessageResult.err != nil {
		log.WithFields(log.Fields{
			"eventStreamId": eventStreamID,
			"groupName":     groupName,
			"error":         getOldestMessageResult.err,
		}).Error("Error while getting oldest parked message.")
		return 0, getOldestMessageResult.err
	}

	oldestMessageUpdatedDateResult := ""
	jp.ArrayEach(getOldestMessageResult.result, func(value []byte, dataType jp.ValueType, offset int, err error) {
		oldestMessageUpdatedDateResult, _ = jp.GetString(value, "updated")
	}, "entries")

	loc, _ := time.LoadLocation("UTC")
	timeNow := time.Now().In(loc)
	oldestMessageUpdatedDate, err := time.Parse(time.RFC3339Nano, oldestMessageUpdatedDateResult)

	if err != nil {
		log.WithFields(log.Fields{
			"eventStreamId": eventStreamID,
			"groupName":     groupName,
			"error":         err,
		}).Error("Cannot parse update time on the oldest parked message.")
		return 0, err
	}

	timeInSeconds := float64(timeNow.Sub(oldestMessageUpdatedDate) / time.Second)

	return timeInSeconds, nil
}

func (esClient *EventStoreStatsClient) getParkedMessagesLastEventNumber(eventStreamID string, groupName string) (int64, error) {
	parkedMessagesURL := fmt.Sprintf("/streams/$persistentsubscription-%s::%s-parked/head/backward/1", eventStreamID, groupName)
	parkedMessagesResultChan := esClient.get(parkedMessagesURL, true)
	parkedMessagesResult := <-parkedMessagesResultChan

	if parkedMessagesResult.err != nil {
		log.WithFields(log.Fields{
			"eventStreamId": eventStreamID,
			"groupName":     groupName,
			"error":         parkedMessagesResult.err,
		}).Error("Error when getting parked messages stream.")
		return 0, parkedMessagesResult.err
	}

	if parkedMessagesResult.result == nil {
		return 0, nil
	}

	eTagString, _ := jp.GetString(parkedMessagesResult.result, "eTag")

	lastEventNumber, err := strconv.ParseInt(strings.Split(eTagString, ";")[0], 10, 64)

	if err != nil {
		log.WithFields(log.Fields{
			"eventStreamId": eventStreamID,
			"groupName":     groupName,
			"error":         err,
		}).Error("Cannot parse eTag on parked messages stream.")
		return 0, err
	}

	lastEventNumber++ // +1 because Ids start from 0

	return lastEventNumber, nil
}

func (esClient *EventStoreStatsClient) getParkedMessagesTruncateBeforeValue(eventStreamID string, groupName string, lastEventNumber int64) (int64, error) {
	metadataURL := fmt.Sprintf("/streams/$persistentsubscription-%s::%s-parked/metadata", eventStreamID, groupName)
	metadataResultChan := esClient.get(metadataURL, false)
	metadataResult := <-metadataResultChan

	if metadataResult.err != nil {
		log.WithFields(log.Fields{
			"eventStreamId": eventStreamID,
			"groupName":     groupName,
			"error":         metadataResult.err,
		}).Error("Error when getting parked message stream metadata")
		return 0, metadataResult.err
	}

	truncateBeforeValue, err := jp.GetInt(metadataResult.result, "$tb")

	if err != nil {
		log.WithFields(log.Fields{
			"eventStreamId": eventStreamID,
			"groupName":     groupName,
		}).Debug("Parked messages have not been replayed yet, as $tb value does not exist in the metadata. Defaulting to 0.")
		return 0, nil
	}

	return truncateBeforeValue, nil
}

func (esClient *EventStoreStatsClient) getParkedMessagesStatsViaGroupInfo(subscriptions []byte) (*[]ParkedMessagesStats, error) {
	var result []ParkedMessagesStats
	jp.ArrayEach(subscriptions, func(jsonValue []byte, dataType jp.ValueType, offset int, e error) {
		eventStreamID, _ := jp.GetString(jsonValue, "eventStreamId")
		groupName, _ := jp.GetString(jsonValue, "groupName")

		groupInfoURL := fmt.Sprintf("/subscriptions/%s/%s/info", eventStreamID, groupName)
		groupInfoChan := esClient.get(groupInfoURL, false)
		groupInfoResult := <-groupInfoChan

		if groupInfoResult.err != nil {
			log.WithFields(log.Fields{
				"eventStreamId": eventStreamID,
				"groupName":     groupName,
				"error":         groupInfoResult.err,
			}).Error("Error when getting subscription group info")
		}

		totalNumberOfParkedMessages, _ := jp.GetFloat(groupInfoResult.result, "parkedMessageCount")

		result = append(result, ParkedMessagesStats{
			EventStreamID:                   eventStreamID,
			GroupName:                       groupName,
			TotalNumberOfParkedMessages:     totalNumberOfParkedMessages,
			OldestParkedMessageAgeInSeconds: -1})
	})
	return &result, nil
}

func (client *EventStoreStatsClient) get(path string, acceptNotFound bool) <-chan getResult {
	url := client.config.EventStoreURL + path

	result := make(chan getResult, 1)

	go func() {
		log.WithField("url", url).Debug("GET request to EventStore")

		req, _ := http.NewRequest("GET", url, nil)
		if client.config.EventStoreUser != "" && client.config.EventStorePassword != "" {
			req.SetBasicAuth(client.config.EventStoreUser, client.config.EventStorePassword)
		}
		req.Header.Add("Accept", "application/json")
		response, err := client.httpClient.Do(req)
		if err != nil {
			result <- getResult{nil, err}
			return
		}
		defer response.Body.Close()

		if response.StatusCode == 404 && acceptNotFound {
			result <- getResult{nil, nil}
			return
		}

		if response.StatusCode >= 400 {
			result <- getResult{nil, fmt.Errorf("HTTP call to %s resulted in status code %d", url, response.StatusCode)}
			return
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
