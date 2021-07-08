package server

import (
	"net/http/httptest"
	"os"
	"testing"
)

func Test206PlusClusterMetrics(t *testing.T) {
	if !shouldRunClusterTest() || getEsVersion(t).IsVersionLowerThan("20.6.0.0") {
		t.Log("Skipping cluster metrics tests for ES >= 20.6")
		return
	}

	es := prepareExporterServer("")
	ts := httptest.NewServer(es.mux)
	defer ts.Close()

	metrics := getMetrics(ts.URL, t)
	assertHasMetric("eventstore_cluster_member_alive", "gauge", metrics, t)
	assertHasMetric("eventstore_cluster_member_is_clone", "gauge", metrics, t)
	assertHasMetric("eventstore_cluster_member_is_follower", "gauge", metrics, t)
	assertHasMetric("eventstore_cluster_member_is_leader", "gauge", metrics, t)
	assertHasMetric("eventstore_cluster_member_is_readonly_replica", "gauge", metrics, t)
}

func Test50ClusterMetrics(t *testing.T) {
	if !shouldRunClusterTest() || getEsVersion(t).IsAtLeastVersion("20.6.0.0") {
		t.Log("Skipping cluster metrics tests for ES 5.0")
		return
	}

	es := prepareExporterServer("")
	ts := httptest.NewServer(es.mux)
	defer ts.Close()

	metrics := getMetrics(ts.URL, t)
	assertHasMetric("eventstore_cluster_member_alive", "gauge", metrics, t)
	assertHasMetric("eventstore_cluster_member_is_clone", "gauge", metrics, t)
	assertHasMetric("eventstore_cluster_member_is_master", "gauge", metrics, t)
	assertHasMetric("eventstore_cluster_member_is_slave", "gauge", metrics, t)
}

func shouldRunClusterTest() bool {
	return os.Getenv("TEST_CLUSTER_MODE") == "cluster"
}
