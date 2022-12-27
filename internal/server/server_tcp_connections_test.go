package server

import (
	"net/http/httptest"
	"os"
	"testing"

	"github.com/marcinbudny/eventstore_exporter/internal/config"
)

func Test_Returns_Tcp_Connection_Stats(t *testing.T) {
	if !shouldRunTcpConnectionTests() {
		t.Log("Skipping TCP connection tests")
		return
	}

	es := prepareExporterServer()
	ts := httptest.NewServer(es.mux)
	defer ts.Close()

	metrics := getMetrics(ts.URL, t)
	assertMetric(t, metrics, "eventstore_tcp_connection_sent_bytes", "counter", anyMetric, nonZeroValue)
	assertMetric(t, metrics, "eventstore_tcp_connection_received_bytes", "counter", anyMetric, nonZeroValue)

	assertHasMetric(t, metrics, "eventstore_tcp_connection_pending_send_bytes", "gauge")
	assertHasMetric(t, metrics, "eventstore_tcp_connection_pending_received_bytes", "gauge")
}

func Test_Returns_No_Tcp_Connection_Stats_When_Disabled(t *testing.T) {
	if !shouldRunTcpConnectionTests() {
		t.Log("Skipping TCP connection tests")
		return
	}

	es := prepareExporterServerWithConfig(func(c *config.Config) {
		c.EnableTcpConnectionStats = false
	})
	ts := httptest.NewServer(es.mux)
	defer ts.Close()

	metrics := getMetrics(ts.URL, t)

	assertHasNoMetric(t, metrics, "eventstore_tcp_connection_sent_bytes")
	assertHasNoMetric(t, metrics, "eventstore_tcp_connection_sent_bytes")
	assertHasNoMetric(t, metrics, "eventstore_tcp_connection_pending_send_bytes")
	assertHasNoMetric(t, metrics, "eventstore_tcp_connection_pending_received_bytes")
}

func shouldRunTcpConnectionTests() bool {
	// we'll have TCP connections to test only in the cluster mode (inter node communication)
	return os.Getenv("TEST_CLUSTER_MODE") == "cluster"
}
