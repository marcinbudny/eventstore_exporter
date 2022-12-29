package server

import (
	"context"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/EventStore/EventStore-Client-Go/v3/esdb"
)

func Test_Basic_SubscriptionAllMetrics(t *testing.T) {
	if !shouldRunSubscriptionToAllTests(t) {
		t.Log("Skipping $all subscriptions tests")
		return
	}

	totalCount := 60
	ackCount := 10
	parkCount := 20
	groupName := prepareSubscriptionToAllEnvironment(t, totalCount, ackCount, parkCount)

	es := prepareExporterServer()
	ts := httptest.NewServer(es.mux)
	defer ts.Close()

	metrics := getMetrics(ts.URL, t)
	assertMetric(t, metrics, "eventstore_subscription_connections", "gauge",
		metricByLabelValue("group_name", groupName), anyValue)
	assertMetric(t, metrics, "eventstore_subscription_messages_in_flight", "gauge",
		metricByLabelValue("group_name", groupName), anyValue)
	assertMetric(t, metrics, "eventstore_subscription_last_known_event_commit_position", "gauge",
		metricByLabelValue("group_name", groupName), nonZeroValue)
	assertMetric(t, metrics, "eventstore_subscription_items_processed_total", "counter",
		metricByLabelValue("group_name", groupName), nonZeroValue) // no idea how many, because $all has events from other streams as well
	assertMetric(t, metrics, "eventstore_subscription_last_checkpointed_event_commit_position", "gauge",
		metricByLabelValue("group_name", groupName), nonZeroValue) // no idea how many, because $all has events from other streams as well

}

func Test_ParkedMessages_SubscriptionToAllMetric(t *testing.T) {
	if !shouldRunSubscriptionToAllTests(t) {
		t.Log("Skipping $all subscriptions tests")
		return
	}

	totalCount := 60
	ackCount := 10
	parkCount := 20
	groupName := prepareSubscriptionToAllEnvironment(t, totalCount, ackCount, parkCount)

	es := prepareExporterServer()
	ts := httptest.NewServer(es.mux)
	defer ts.Close()

	metrics := getMetrics(ts.URL, t)
	assertMetric(t, metrics, "eventstore_subscription_parked_messages", "gauge",
		metricByLabelValue("group_name", groupName), hasValue(float64(parkCount)))
}

func Test_OldestParkedMessage_SubscriptionToAllMetric(t *testing.T) {
	if !shouldRunSubscriptionToAllTests(t) {
		t.Log("Skipping $all subscriptions tests")
		return
	}

	totalCount := 60
	ackCount := 10
	parkCount := 20
	groupName := prepareSubscriptionToAllEnvironment(t, totalCount, ackCount, parkCount)

	es := prepareExporterServer()
	ts := httptest.NewServer(es.mux)
	defer ts.Close()

	metrics := getMetrics(ts.URL, t)
	assertMetric(t, metrics, "eventstore_subscription_oldest_parked_message_age_seconds", "gauge",
		metricByLabelValue("group_name", groupName), anyValue)
}

func Test_ParkedMessages_SubscriptionToAllMetric_With_Replayed_Messages(t *testing.T) {
	if !shouldRunSubscriptionToAllTests(t) {
		t.Log("Skipping $all subscriptions tests")
		return
	}

	parkCount := 20
	groupName := prepareSubscriptionToAllEnvironmentWithReplayedMessages(t, parkCount)

	es := prepareExporterServer()
	ts := httptest.NewServer(es.mux)
	defer ts.Close()

	metrics := getMetrics(ts.URL, t)
	assertMetric(t, metrics, "eventstore_subscription_parked_messages", "gauge",
		metricByLabelValue("group_name", groupName), hasValue(float64(0)))
}

func shouldRunSubscriptionToAllTests(t *testing.T) bool {

	esInfo := getEsInfo(t)
	// do not run in cluster mode, as this causes issues when not connected to leader node
	return os.Getenv("TEST_CLUSTER_MODE") != "cluster" && esInfo.EsVersion.IsAtLeastVersion("21.10.0.0")
}

func prepareSubscriptionToAllEnvironment(t *testing.T, totalCount int, ackCount int, parkCount int) string {
	streamID, groupName := newStreamAndGroup()
	t.Logf("Stream: %s, group: %s", streamID, groupName)

	client := getEsClient(t)
	defer client.Close()

	writeTestEvents(t, totalCount, streamID, client)

	createSubscriptionToAll(t, groupName, client)

	subscription := connectToSubscriptionToAll(t, groupName, client)

	ackMessages(t, ackCount, subscription)
	parkMessages(t, parkCount, subscription)

	time.Sleep(time.Millisecond * 2000)

	return groupName
}

func prepareSubscriptionToAllEnvironmentWithReplayedMessages(t *testing.T, parkCount int) string {
	streamID, groupName := newStreamAndGroup()
	t.Logf("Stream: %s, group: %s", streamID, groupName)

	client := getEsClient(t)
	defer client.Close()

	writeTestEvents(t, parkCount, streamID, client)

	createSubscriptionToAll(t, groupName, client)

	readClient := connectToSubscriptionToAll(t, groupName, client)
	parkMessages(t, parkCount, readClient)
	time.Sleep(time.Millisecond * 1000)

	replayParkedMessages(t, "$all", groupName)
	time.Sleep(time.Millisecond * 1000)

	ackMessages(t, parkCount, readClient)

	// give internal stats time to be updated
	time.Sleep(time.Millisecond * 1000)

	return groupName
}

func createSubscriptionToAll(t *testing.T, groupName string, client *esdb.Client) {
	if err := client.CreatePersistentSubscriptionToAll(context.Background(), groupName, esdb.PersistentAllSubscriptionOptions{
		StartFrom: esdb.Start{},
	}); err != nil {
		t.Fatal(err)
	}
}

func connectToSubscriptionToAll(t *testing.T, groupName string, client *esdb.Client) *esdb.PersistentSubscription {
	subscription, err := client.SubscribeToPersistentSubscriptionToAll(

		context.Background(), groupName, esdb.SubscribeToPersistentSubscriptionOptions{BufferSize: 1})

	if err != nil {
		t.Fatal(err)
	}

	return subscription
}
