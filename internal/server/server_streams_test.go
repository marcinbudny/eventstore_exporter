package server

import (
	"net/http/httptest"
	"testing"

	"github.com/marcinbudny/eventstore_exporter/internal/config"
)

func Test_StreamStats(t *testing.T) {

	client := getEsClient(t)
	stream1ID := newUUID()
	stream2ID := newUUID()

	writeTestEvents(t, 12, stream1ID, client)
	writeTestEvents(t, 9, stream2ID, client)

	es := prepareExporterServerWithConfig(func(config *config.Config) {
		config.Streams = []string{stream1ID, stream2ID, "$all"}
	})
	ts := httptest.NewServer(es.mux)
	defer ts.Close()

	metrics := getMetrics(ts.URL, t)

	assertMetric(t, metrics, "eventstore_stream_last_commit_position", "gauge", metricByLabelValue("event_stream_id", "$all"), anyValue)
	assertMetric(t, metrics, "eventstore_stream_last_event_number", "gauge", metricByLabelValue("event_stream_id", stream1ID), hasValue(float64(12-1))) // event ids start at 0
	assertMetric(t, metrics, "eventstore_stream_last_event_number", "gauge", metricByLabelValue("event_stream_id", stream2ID), hasValue(float64(9-1)))  // event ids start at 0
}
