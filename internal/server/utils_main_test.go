package server

import (
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
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
	} // else: ignore, we'll use default value

}

func tryDetectClusterMode() {
	if os.Getenv("TEST_CLUSTER_MODE") != "" {
		return // already set
	}

	// assume single node runs in insecure DEV config, while cluster runs over TLS
	if strings.HasPrefix(os.Getenv("TEST_EVENTSTORE_URL"), "https://") {
		os.Setenv("TEST_CLUSTER_MODE", "cluster")
	} // else: ignore, we'll use default value
}
