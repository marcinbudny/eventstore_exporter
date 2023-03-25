package server

import (
	"net/http/httptest"
	"testing"
)

func Test_ClusterMetrics(t *testing.T) {
	es := prepareExporterServer()
	ts := httptest.NewServer(es.mux)
	defer ts.Close()

	metrics := getMetrics(ts.URL, t)
	assertHasMetric(t, metrics, "eventstore_cluster_member_alive", "gauge")
	assertHasMetric(t, metrics, "eventstore_cluster_member_is_clone", "gauge")
	assertHasMetric(t, metrics, "eventstore_cluster_member_is_follower", "gauge")
	assertHasMetric(t, metrics, "eventstore_cluster_member_is_leader", "gauge")
	assertHasMetric(t, metrics, "eventstore_cluster_member_is_readonly_replica", "gauge")
}
