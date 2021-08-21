package server

import (
	"context"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/EventStore/EventStore-Client-Go/messages"
	"github.com/EventStore/EventStore-Client-Go/persistent"
	"github.com/EventStore/EventStore-Client-Go/streamrevision"
	"github.com/gofrs/uuid"

	esclient "github.com/EventStore/EventStore-Client-Go/client"
)

func Test_Basic_SubscriptionMetrics(t *testing.T) {
	if !shouldRunSubscriptionTests(t) {
		t.Log("Skipping subscriptions tests")
		return
	}

	totalCount := 60
	ackCount := 10
	parkCount := 20
	_, groupName := prepareSubscriptionEnvironment(t, totalCount, ackCount, parkCount)

	es := prepareExporterServer()
	ts := httptest.NewServer(es.mux)
	defer ts.Close()

	metrics := getMetrics(ts.URL, t)
	assertMetric(t, metrics, "eventstore_subscription_connections", "gauge",
		metricByLabelValue("group_name", groupName), anyValue)
	assertMetric(t, metrics, "eventstore_subscription_messages_in_flight", "gauge",
		metricByLabelValue("group_name", groupName), anyValue)
	assertMetric(t, metrics, "eventstore_subscription_last_known_event_number", "gauge",
		metricByLabelValue("group_name", groupName), anyValue)
	assertMetric(t, metrics, "eventstore_subscription_items_processed_total", "counter",
		metricByLabelValue("group_name", groupName), hasValue(float64(ackCount+parkCount+1))) // account for one buffered event
	assertMetric(t, metrics, "eventstore_subscription_last_processed_event_number", "gauge",
		metricByLabelValue("group_name", groupName), hasValue(float64(ackCount+parkCount-1)))

}

func Test_ParkedMessages_SubscriptionMetric(t *testing.T) {
	if !shouldRunSubscriptionTests(t) {
		t.Log("Skipping subscriptions tests")
		return
	}
	if !supportsParkedMessagesMetric(t) {
		t.Log("Skipping parked message tests, the metric is not supported by current ES setup")
		return
	}

	totalCount := 60
	ackCount := 10
	parkCount := 20
	_, groupName := prepareSubscriptionEnvironment(t, totalCount, ackCount, parkCount)

	es := prepareExporterServer()
	ts := httptest.NewServer(es.mux)
	defer ts.Close()

	metrics := getMetrics(ts.URL, t)
	assertMetric(t, metrics, "eventstore_subscription_parked_messages", "gauge",
		metricByLabelValue("group_name", groupName), hasValue(float64(parkCount)))
}

func Test_OldestParkedMessage_SubscriptionMetric(t *testing.T) {
	if !shouldRunSubscriptionTests(t) {
		t.Log("Skipping subscriptions tests")
		return
	}
	if !supportsOldestParkedMessagesMetric(t) {
		t.Log("Skipping oldest parked message tests, the metric is not supported by current ES setup")
		return
	}

	totalCount := 60
	ackCount := 10
	parkCount := 20
	_, groupName := prepareSubscriptionEnvironment(t, totalCount, ackCount, parkCount)

	es := prepareExporterServer()
	ts := httptest.NewServer(es.mux)
	defer ts.Close()

	metrics := getMetrics(ts.URL, t)
	assertMetric(t, metrics, "eventstore_subscription_oldest_parked_message_age_seconds", "gauge",
		metricByLabelValue("group_name", groupName), anyValue)
}

// TODO: possible es client bug preventing from properly reading replayed messages

// func Test_ParkedMessages_SubscriptionMetric_With_Replayed_Messages(t *testing.T) {
// 	if !shouldRunSubscriptionTests(t) {
// 		t.Log("Skipping subscriptions tests")
// 		return
// 	}
// 	if !supportsParkedMessagesMetric(t) {
// 		t.Log("Skipping parked message tests, the metric is not supported by current ES setup")
// 		return
// 	}

// 	parkCount := 20
// 	_, groupName := prepareSubscriptionEnvironmentWithReplayedMessages(t, parkCount)

// 	es := prepareExporterServer()
// 	ts := httptest.NewServer(es.mux)
// 	defer ts.Close()

// 	metrics := getMetrics(ts.URL, t)
// 	assertMetric(t, metrics, "eventstore_subscription_parked_messages", "gauge",
// 		metricByLabelValue("group_name", groupName), hasValue(float64(0)))
// }

func shouldRunSubscriptionTests(t *testing.T) bool {
	if getEsVersion(t).IsVersionLowerThan("20.6.0.0") {
		return false
	}

	// do not run in cluster mode, as this causes issues when not connected to leader node
	return os.Getenv("TEST_CLUSTER_MODE") != "cluster"
}

func supportsParkedMessagesMetric(t *testing.T) bool {
	if getEsVersion(t).ReportsParkedMessageNumber() || atomPubIsEnabled(t) {
		return true
	}

	return false
}

func supportsOldestParkedMessagesMetric(t *testing.T) bool {
	return atomPubIsEnabled(t)
}

func prepareSubscriptionEnvironment(t *testing.T, totalCount int, ackCount int, parkCount int) (streamID string, groupName string) {
	streamID, groupName = newStreamAndGroup()
	t.Logf("Stream: %s, group: %s", streamID, groupName)

	client := getEsClient(t)
	defer client.Close()

	writeTestEvents(t, totalCount, streamID, client)

	createSubscription(t, streamID, groupName, client)

	readClient := connectToSubscription(t, streamID, groupName, client)

	ackMessages(t, ackCount, readClient)
	parkMessages(t, parkCount, readClient)

	time.Sleep(time.Millisecond * 2000)
	client.Close()

	return
}

// TODO: possible es client bug preventing from properly reading replayed messages

// func prepareSubscriptionEnvironmentWithReplayedMessages(t *testing.T, parkCount int) (streamID string, groupName string) {
// 	streamID, groupName = newStreamAndGroup()
// 	t.Logf("Stream: %s, group: %s", streamID, groupName)

// 	client := getEsClient(t)
// 	defer client.Close()

// 	writeTestEvents(t, parkCount, streamID, client)

// 	createSubscription(t, streamID, groupName, client)

// 	readClient := connectToSubscription(t, streamID, groupName, client)
// 	parkMessages(t, parkCount, readClient)
// 	time.Sleep(time.Millisecond * 1000)

// 	replayParkedMessages(t, streamID, groupName)
// 	time.Sleep(time.Millisecond * 1000)

// 	ackMessages(t, parkCount, readClient)

// 	// give internal stats time to be updated
// 	time.Sleep(time.Millisecond * 1000)
// 	client.Close()

// 	return
// }

func newStreamAndGroup() (streamID string, groupName string) {
	streamUUID, _ := uuid.NewV4()
	streamID = streamUUID.String()

	groupUUID, _ := uuid.NewV4()
	groupName = groupUUID.String()

	return
}

func writeTestEvents(t *testing.T, eventCount int, streamID string, client *esclient.Client) {
	events := make([]messages.ProposedEvent, 0)
	for i := 0; i < eventCount; i++ {
		eventID, _ := uuid.NewV4()
		events = append(events, messages.ProposedEvent{
			EventID:     eventID,
			EventType:   "TestEvent",
			ContentType: "application/octet-stream",
			Data:        []byte{0xb, 0xe, 0xe, 0xf},
		})
	}

	if _, err := client.AppendToStream(context.Background(), streamID, streamrevision.StreamRevisionNoStream, events); err != nil {
		t.Fatal(err)
	}
}

func createSubscription(t *testing.T, streamID string, groupName string, client *esclient.Client) {
	settings := persistent.DefaultSubscriptionSettings
	settings.ReadBatchSize = 1

	if err := client.CreatePersistentSubscription(context.Background(), persistent.SubscriptionStreamConfig{
		StreamOption: persistent.StreamSettings{
			StreamName: []byte(streamID),
			Revision:   persistent.Revision_Start,
		},
		GroupName: groupName,
		Settings:  settings,
	}); err != nil {
		t.Fatal(err)
	}
}

func connectToSubscription(t *testing.T, streamID string, groupName string, client *esclient.Client) persistent.SyncReadConnection {
	readClient, err := client.ConnectToPersistentSubscription(
		context.Background(), 1, groupName, []byte(streamID))

	if err != nil {
		t.Fatal(err)
	}

	return readClient
}

func ackMessages(t *testing.T, ackCount int, readClient persistent.SyncReadConnection) {
	for i := 0; i < ackCount; i++ {
		if readEvent, err := readClient.Read(); err != nil {
			t.Fatal(err)
		} else {
			readClient.Ack(readEvent.EventID)
		}
	}
}

func parkMessages(t *testing.T, parkCount int, readClient persistent.SyncReadConnection) {

	for i := 0; i < parkCount; i++ {
		if readEvent, err := readClient.Read(); err != nil {
			t.Fatal(err)
		} else {
			readClient.Nack("test", persistent.Nack_Park, readEvent.EventID)
		}
	}
}
