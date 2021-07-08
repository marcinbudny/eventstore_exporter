package server

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLandingPage(t *testing.T) {
	es := prepareExporterServer("")
	ts := httptest.NewServer(es.mux)
	defer ts.Close()

	content := getString(ts.URL, t)

	if !strings.Contains(string(content), "EventStore exporter") {
		t.Errorf("Expected content to have 'EventSTore exporeter' but got %s", content)
	}
}

func TestBasicMetrics(t *testing.T) {
	es := prepareExporterServer("")
	ts := httptest.NewServer(es.mux)
	defer ts.Close()

	metrics := getMetrics(ts.URL, t)
	assertHasMetric("eventstore_up", "gauge", metrics, t)
	assertHasMetric("eventstore_disk_io_read_bytes", "gauge", metrics, t)
	assertHasMetric("eventstore_disk_io_read_ops", "gauge", metrics, t)
	assertHasMetric("eventstore_disk_io_write_ops", "gauge", metrics, t)
	assertHasMetric("eventstore_disk_io_written_bytes", "gauge", metrics, t)
	assertHasMetric("eventstore_drive_available_bytes", "gauge", metrics, t)
	assertHasMetric("eventstore_process_cpu", "gauge", metrics, t)
	assertHasMetric("eventstore_process_memory_bytes", "gauge", metrics, t)
	assertHasMetric("eventstore_queue_items_processed_total", "counter", metrics, t)
	assertHasMetric("eventstore_queue_length", "gauge", metrics, t)
	assertHasMetric("eventstore_tcp_connections", "gauge", metrics, t)
	assertHasMetric("eventstore_tcp_received_bytes", "gauge", metrics, t)
	assertHasMetric("eventstore_tcp_sent_bytes", "gauge", metrics, t)
}

func TestEventStoreUp(t *testing.T) {
	tests := []struct {
		name            string
		eventStoreURL   string
		expectedUpValue float64
	}{
		{
			name:            "eventstore_up should be 1 if it can connect to ES",
			eventStoreURL:   "",
			expectedUpValue: 1,
		},
		{
			name:            "eventstore_up should be 0 if it cannot connect to ES",
			eventStoreURL:   "http://does_not_exist",
			expectedUpValue: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			es := prepareExporterServer(test.eventStoreURL)
			ts := httptest.NewServer(es.mux)
			defer ts.Close()

			metrics := getMetrics(ts.URL, t)
			assertMetricValue("eventstore_up", "gauge", test.expectedUpValue, metrics, t)
		})
	}
}
