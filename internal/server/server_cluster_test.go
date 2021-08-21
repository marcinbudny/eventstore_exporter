package server

import (
	"net/http/httptest"
	"os"
	"testing"
)

func Test_206Plus_ClusterMetrics(t *testing.T) {
	if !shouldRunClusterTest() || getEsVersion(t).IsVersionLowerThan("20.6.0.0") {
		t.Log("Skipping cluster metrics tests for ES >= 20.6")
		return
	}

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

func Test_50_ClusterMetrics(t *testing.T) {
	if !shouldRunClusterTest() || getEsVersion(t).IsAtLeastVersion("20.6.0.0") {
		t.Log("Skipping cluster metrics tests for ES 5.0")
		return
	}

	es := prepareExporterServer()
	ts := httptest.NewServer(es.mux)
	defer ts.Close()

	metrics := getMetrics(ts.URL, t)
	assertHasMetric(t, metrics, "eventstore_cluster_member_alive", "gauge")
	assertHasMetric(t, metrics, "eventstore_cluster_member_is_clone", "gauge")
	assertHasMetric(t, metrics, "eventstore_cluster_member_is_master", "gauge")
	assertHasMetric(t, metrics, "eventstore_cluster_member_is_slave", "gauge")
}

func shouldRunClusterTest() bool {
	return os.Getenv("TEST_CLUSTER_MODE") == "cluster"
}
