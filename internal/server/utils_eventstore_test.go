package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/EventStore/EventStore-Client-Go/esdb"
	"github.com/gofrs/uuid"
)

func getEventstoreHttpClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &http.Client{
		Transport: tr,
	}
}

func replayParkedMessages(t *testing.T, streamID string, groupName string) {
	httpClient := getEventstoreHttpClient()

	eventStoreURL := getEventStoreURL()

	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/subscriptions/%s/%s/replayParked", eventStoreURL, streamID, groupName), nil)
	req.SetBasicAuth("admin", "changeit")
	req.Header.Add("Accept", "application/json")
	res, errPost := httpClient.Do(req)

	if errPost != nil {
		t.Fatal(errPost)
	}

	if res.StatusCode != 200 {
		t.Fatal("Unable to replay messages")
	}
}

func getEventStoreURL() string {
	eventStoreURL := "http://localhost:2113"
	if os.Getenv("TEST_EVENTSTORE_URL") != "" {
		eventStoreURL = os.Getenv("TEST_EVENTSTORE_URL")
	}

	return eventStoreURL
}

func getEventStoreConnectionString() string {
	u, _ := url.Parse(getEventStoreURL())
	originalScheme := u.Scheme
	u.Scheme = "esdb"
	u.User = url.UserPassword("admin", "changeit")
	q := u.Query()
	if originalScheme == "https" {
		q.Set("tls", "true")
		q.Set("tlsverifycert", "false")
	} else {
		q.Set("tls", "false")
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func getEsClient(t *testing.T) *esdb.Client {
	connectionString := getEventStoreConnectionString()
	t.Logf("ES connection string: %s", connectionString)

	config, err := esdb.ParseConnectionString(connectionString)
	if err != nil {
		t.Fatal(err)
	}
	client, err := esdb.NewClient(config)
	if err != nil {
		t.Fatal(err)
	}

	return client
}

func newStreamAndGroup() (streamID string, groupName string) {
	return newUUID(), newUUID()
}

func newUUID() string {
	uuid, _ := uuid.NewV4()
	return uuid.String()
}

func writeTestEvents(t *testing.T, eventCount int, streamID string, client *esdb.Client) {
	events := make([]esdb.EventData, 0)
	for i := 0; i < eventCount; i++ {
		events = append(events, esdb.EventData{
			EventType:   "TestEvent",
			ContentType: esdb.BinaryContentType,
			Data:        []byte{0xb, 0xe, 0xe, 0xf},
		})
	}

	options := esdb.AppendToStreamOptions{
		ExpectedRevision: esdb.Any{},
	}

	if _, err := client.AppendToStream(context.Background(), streamID, options, events...); err != nil {
		t.Fatal(err)
	}
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
