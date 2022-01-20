package server

import (
	"net/http/httptest"
	"testing"

	"github.com/marcinbudny/eventstore_exporter/internal/config"
)

func Test_StreamStats(t *testing.T) {

	client := getEsClient(t)
	streamID := newUUID()

	writeTestEvents(t, 12, streamID, client)

	es := prepareExporterServerWithConfig(func(config *config.Config) {
		config.Streams = []string{streamID, "$all"}
	})
	ts := httptest.NewServer(es.mux)
	defer ts.Close()

	metrics := getMetrics(ts.URL, t)

	assertMetric(t, metrics, "eventstore_stream_last_position", "gauge", metricByLabelValue("event_stream_id", "$all"), anyValue)
	assertMetric(t, metrics, "eventstore_stream_last_position", "gauge", metricByLabelValue("event_stream_id", streamID), hasValue(float64(12-1))) // event ids start at 0
}
