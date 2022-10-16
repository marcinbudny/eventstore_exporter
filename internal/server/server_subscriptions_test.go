package server

import (
	"context"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/EventStore/EventStore-Client-Go/v3/esdb"
)

func Test_Basic_SubscriptionMetrics(t *testing.T) {
	if !shouldRunSubscriptionTests() {
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
		metricByLabelValue("group_name", groupName), nonZeroValue)
	assertMetric(t, metrics, "eventstore_subscription_items_processed_total", "counter",
		metricByLabelValue("group_name", groupName), hasValue(float64(ackCount+parkCount+1))) // account for one buffered event
	assertMetric(t, metrics, "eventstore_subscription_last_processed_event_number", "gauge",
		metricByLabelValue("group_name", groupName), hasValue(float64(ackCount+parkCount-1)))

}

func Test_ParkedMessages_SubscriptionMetric(t *testing.T) {
	if !shouldRunSubscriptionTests() {
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
	assertMetric(t, metrics, "eventstore_subscription_parked_messages", "gauge",
		metricByLabelValue("group_name", groupName), hasValue(float64(parkCount)))
}

func Test_OldestParkedMessage_SubscriptionMetric(t *testing.T) {
	if !shouldRunSubscriptionTests() {
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
	assertMetric(t, metrics, "eventstore_subscription_oldest_parked_message_age_seconds", "gauge",
		metricByLabelValue("group_name", groupName), anyValue)
}

func Test_ParkedMessages_SubscriptionMetric_With_Replayed_Messages(t *testing.T) {
	if !shouldRunSubscriptionTests() {
		t.Log("Skipping subscriptions tests")
		return
	}

	parkCount := 20
	_, groupName := prepareSubscriptionEnvironmentWithReplayedMessages(t, parkCount)

	es := prepareExporterServer()
	ts := httptest.NewServer(es.mux)
	defer ts.Close()

	metrics := getMetrics(ts.URL, t)
	assertMetric(t, metrics, "eventstore_subscription_parked_messages", "gauge",
		metricByLabelValue("group_name", groupName), hasValue(float64(0)))
}

func shouldRunSubscriptionTests() bool {
	// do not run in cluster mode, as this causes issues when not connected to leader node
	return os.Getenv("TEST_CLUSTER_MODE") != "cluster"
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

	return
}

func prepareSubscriptionEnvironmentWithReplayedMessages(t *testing.T, parkCount int) (streamID string, groupName string) {
	streamID, groupName = newStreamAndGroup()
	t.Logf("Stream: %s, group: %s", streamID, groupName)

	client := getEsClient(t)
	defer client.Close()

	writeTestEvents(t, parkCount, streamID, client)

	createSubscription(t, streamID, groupName, client)

	readClient := connectToSubscription(t, streamID, groupName, client)
	parkMessages(t, parkCount, readClient)
	time.Sleep(time.Millisecond * 1000)

	replayParkedMessages(t, streamID, groupName)
	time.Sleep(time.Millisecond * 1000)

	ackMessages(t, parkCount, readClient)

	// give internal stats time to be updated
	time.Sleep(time.Millisecond * 1000)

	return
}

func createSubscription(t *testing.T, streamID string, groupName string, client *esdb.Client) {
	if err := client.CreatePersistentSubscription(context.Background(), streamID, groupName, esdb.PersistentStreamSubscriptionOptions{
		StartFrom: esdb.Start{},
	}); err != nil {
		t.Fatal(err)
	}
}

func connectToSubscription(t *testing.T, streamID string, groupName string, client *esdb.Client) *esdb.PersistentSubscription {
	subscription, err := client.SubscribeToPersistentSubscription(

		context.Background(), streamID, groupName, esdb.SubscribeToPersistentSubscriptionOptions{BufferSize: 1})

	if err != nil {
		t.Fatal(err)
	}

	return subscription
}
