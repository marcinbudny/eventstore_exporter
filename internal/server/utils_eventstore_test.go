package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/EventStore/EventStore-Client-Go/v3/esdb"
	"github.com/gofrs/uuid"
	"github.com/marcinbudny/eventstore_exporter/internal/client"
	"github.com/marcinbudny/eventstore_exporter/internal/config"
)

func getEventstoreHttpClient() *http.Client {
	return getEventstoreHttpClientWithConfig(nil)
}

func getEventstoreHttpClientWithConfig(configureClient func(*http.Client)) *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   time.Second * 10,
	}

	if configureClient != nil {
		configureClient(client)
	}

	return client
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

func getEsInfo(t *testing.T) *client.EsInfo {
	c := client.New(&config.Config{
		EventStoreURL:      getEventStoreURL(),
		EventStoreUser:     "admin",
		EventStorePassword: "changeit",
		InsecureSkipVerify: true,
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	esInfo, err := c.GetEsInfo(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return esInfo
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
	esClient, err := esdb.NewClient(config)
	if err != nil {
		t.Fatal(err)
	}

	return esClient
}

func newStreamAndGroup() (streamID string, groupName string) {
	return newUUID(), newUUID()
}

func newUUID() string {
	id, _ := uuid.NewV4()
	return id.String()
}

func writeTestEvents(t *testing.T, eventCount int, streamID string, client *esdb.Client) {
	events := make([]esdb.EventData, 0)
	for i := 0; i < eventCount; i++ {
		events = append(events, esdb.EventData{
			EventType:   "TestEvent",
			ContentType: esdb.ContentTypeBinary,
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
		if err := subscription.Ack(event.Event); err != nil {
			t.Fatal(err)
		}
	}
}

func parkMessages(t *testing.T, parkCount int, subscription *esdb.PersistentSubscription) {

	for i := 0; i < parkCount; i++ {
		event := subscription.Recv().EventAppeared
		if err := subscription.Nack("reason", esdb.NackActionPark, event.Event); err != nil {
			t.Fatal(err)
		}
	}
}
