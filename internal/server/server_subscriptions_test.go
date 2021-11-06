package server

import (
	"context"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"github.com/EventStore/EventStore-Client-Go/esdb"
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

	subscription := connectToSubscription(t, streamID, groupName, client)

	ackMessages(t, ackCount, subscription)
	parkMessages(t, parkCount, subscription)

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

func writeTestEvents(t *testing.T, eventCount int, streamID string, client *esdb.Client) {
	events := make([]esdb.EventData, 0)
	for i := 0; i < eventCount; i++ {
		//eventID, _ := uuid.NewV4()
		events = append(events, esdb.EventData{
			//EventID:     eventID,
			EventType:   "TestEvent",
			ContentType: esdb.BinaryContentType,
			Data:        []byte{0xb, 0xe, 0xe, 0xf},
		})
	}

	options := esdb.AppendToStreamOptions{
		ExpectedRevision: esdb.NoStream{},
	}

	if _, err := client.AppendToStream(context.Background(), streamID, options, events...); err != nil {
		t.Fatal(err)
	}
}

func createSubscription(t *testing.T, streamID string, groupName string, client *esdb.Client) {
	if err := client.CreatePersistentSubscription(context.Background(), streamID, groupName, esdb.PersistentStreamSubscriptionOptions{
		From: esdb.Start{},
	}); err != nil {
		t.Fatal(err)
	}
}

func connectToSubscription(t *testing.T, streamID string, groupName string, client *esdb.Client) *esdb.PersistentSubscription {
	subscription, err := client.ConnectToPersistentSubscription(

		context.Background(), streamID, groupName, esdb.ConnectToPersistentSubscriptionOptions{BatchSize: 1})

	if err != nil {
		t.Fatal(err)
	}

	return subscription
}

func ackMessages(t *testing.T, ackCount int, subscription *esdb.PersistentSubscription) {
	for i := 0; i < ackCount; i++ {
		event := subscription.Recv().EventAppeared
		subscription.Ack(event)
	}
}

func parkMessages(t *testing.T, parkCount int, subscription *esdb.PersistentSubscription) {

	for i := 0; i < parkCount; i++ {
		event := subscription.Recv().EventAppeared
		subscription.Nack("reason", esdb.Nack_Park, event)
	}
}
