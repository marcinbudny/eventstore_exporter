package server

import (
	"net/http/httptest"
	"os"
	"testing"
)

func Test_ProjectionMetrics(t *testing.T) {
	if !shouldRunProjectionsTest() {
		t.Log("Skipping projection metrics")
		return
	}

	es := prepareExporterServer()
	ts := httptest.NewServer(es.mux)
	defer ts.Close()

	metrics := getMetrics(ts.URL, t)
	assertHasMetric(t, metrics, "eventstore_projection_running", "gauge")
	assertHasMetric(t, metrics, "eventstore_projection_status", "gauge")
	assertHasMetric(t, metrics, "eventstore_projection_progress", "gauge")
	assertHasMetric(t, metrics, "eventstore_projection_events_processed_after_restart_total", "counter")
}

func shouldRunProjectionsTest() bool {
	_, run := os.LookupEnv("TEST_PROJECTION_METRICS")
	return run
}
