package server

import (
	"net/http/httptest"
	"os"
	"testing"
)

func TestProjectionMetrics(t *testing.T) {
	if !shouldRunProjectionsTest() {
		t.Log("Skipping projection metrics")
		return
	}

	es := prepareExporterServer("")
	ts := httptest.NewServer(es.mux)
	defer ts.Close()

	metrics := getMetrics(ts.URL, t)
	assertHasMetric("eventstore_projection_running", "gauge", metrics, t)
	assertHasMetric("eventstore_projection_progress", "gauge", metrics, t)
	assertHasMetric("eventstore_projection_events_processed_after_restart_total", "counter", metrics, t)
}

func shouldRunProjectionsTest() bool {
	_, run := os.LookupEnv("TEST_PROJECTION_METRICS")
	return run
}
