package server

import (
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {

	tryDetectEventStoreURL()
	tryDetectClusterMode()

	code := m.Run()
	os.Exit(code)
}

func tryDetectEventStoreURL() {
	if os.Getenv("TEST_EVENTSTORE_URL") != "" {
		return // already set
	}

	client := getEventstoreHTTPClientWithConfig(func(client *http.Client) {
		client.Timeout = time.Second * 1
	})

	r, err := client.Get("https://localhost:2113")
	if err == nil {
		r.Body.Close()
		os.Setenv("TEST_EVENTSTORE_URL", "https://localhost:2113")
		log.Infof("Using https://localhost:2113 as EventStore URL")
		return
	}
	log.Infof("Failed to connect to EventStore at https://localhost:2113, using default EventStore URL: %v", err)
}

func tryDetectClusterMode() {
	if os.Getenv("TEST_CLUSTER_MODE") != "" {
		log.Info("Tests running in cluster mode")
		return // already set
	}

	// assume single node runs in insecure DEV config, while cluster runs over TLS
	if strings.HasPrefix(os.Getenv("TEST_EVENTSTORE_URL"), "https://") {
		os.Setenv("TEST_CLUSTER_MODE", "cluster")
		log.Info("Detected tests in cluster mode")
	} // else: ignore, we'll use default value
}
