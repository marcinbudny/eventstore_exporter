package server

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func Test_LandingPage(t *testing.T) {
	es := prepareExporterServer()
	ts := httptest.NewServer(es.mux)
	defer ts.Close()

	content := getStringFromExporterEndpoint(ts.URL, t)

	if !strings.Contains(string(content), "EventStore exporter") {
		t.Errorf("Expected content to have 'EventStore exporter' but got %s", content)
	}
}

func Test_BasicMetrics(t *testing.T) {
	es := prepareExporterServer()
	ts := httptest.NewServer(es.mux)
	defer ts.Close()

	metrics := getMetrics(ts.URL, t)
	assertHasMetric(t, metrics, "eventstore_up", "gauge")
	assertHasMetric(t, metrics, "eventstore_disk_io_read_bytes", "gauge")
	assertHasMetric(t, metrics, "eventstore_disk_io_read_ops", "gauge")
	assertHasMetric(t, metrics, "eventstore_disk_io_write_ops", "gauge")
	assertHasMetric(t, metrics, "eventstore_disk_io_written_bytes", "gauge")
	assertHasMetric(t, metrics, "eventstore_drive_available_bytes", "gauge")
	assertHasMetric(t, metrics, "eventstore_process_cpu", "gauge")
	assertHasMetric(t, metrics, "eventstore_process_memory_bytes", "gauge")
	assertHasMetric(t, metrics, "eventstore_queue_items_processed_total", "counter")
	assertHasMetric(t, metrics, "eventstore_queue_length", "gauge")
	assertHasMetric(t, metrics, "eventstore_tcp_connections", "gauge")
	assertHasMetric(t, metrics, "eventstore_tcp_received_bytes", "gauge")
	assertHasMetric(t, metrics, "eventstore_tcp_sent_bytes", "gauge")
}

func Test_EventStoreUp_Up(t *testing.T) {

	es := prepareExporterServer()
	ts := httptest.NewServer(es.mux)
	defer ts.Close()

	metrics := getMetrics(ts.URL, t)
	assertMetric(t, metrics, "eventstore_up", "gauge", singleValuedMetric, hasValue(1))
}

func Test_EventStoreUp_Down(t *testing.T) {
	es := prepareExporterServerWithInvalidConnection()
	ts := httptest.NewServer(es.mux)
	defer ts.Close()

	metrics := getMetrics(ts.URL, t)
	assertMetric(t, metrics, "eventstore_up", "gauge", singleValuedMetric, hasValue(0))
}
