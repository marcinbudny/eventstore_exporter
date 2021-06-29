package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"

	"github.com/marcinbudny/eventstore_exporter/client"
	"github.com/marcinbudny/eventstore_exporter/collector"
	"github.com/marcinbudny/eventstore_exporter/config"
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

func assertHasMetric(name string, metricType string, metrics map[string]*dto.MetricFamily, t *testing.T) {
	if family, ok := metrics[name]; ok {
		if len(family.Metric) > 0 {
			metric := family.Metric[0]
			switch strings.ToLower(metricType) {
			case "counter":
				if metric.Counter == nil {
					t.Errorf("Metric %s is not a counter", name)
				}

			case "gauge":
				if metric.Gauge == nil {
					t.Errorf("Metric %s is not a gauge", name)
				}

			default:
				t.Fatalf("Unsupported metric type %s", metricType)
			}
		} else {
			t.Errorf("Metric family %s has no metrics", name)
		}
	} else {
		t.Errorf("Metric family %s is missing", name)
	}
}

func assertMetricValue(name string, metricType string, expectedValue float64, metrics map[string]*dto.MetricFamily, t *testing.T) {
	if family, ok := metrics[name]; ok {
		if len(family.Metric) == 1 {
			metric := family.Metric[0]
			switch strings.ToLower(metricType) {
			case "counter":
				if metric.Counter != nil {
					if *metric.Counter.Value != expectedValue {
						t.Errorf("Expected metric %s to have value %f, but it actually is %f", name, expectedValue, *metric.Counter.Value)
					}
				} else {
					t.Errorf("Metric %s is not a counter", name)
				}

			case "gauge":
				if metric.Gauge != nil {
					if *metric.Gauge.Value != expectedValue {
						t.Errorf("Expected metric %s to have value %f, but it actually is %f", name, expectedValue, *metric.Gauge.Value)
					}
				} else {
					t.Errorf("Metric %s is not a gauge", name)
				}

			default:
				t.Fatalf("Unsupported metric type %s", metricType)
			}
		} else {
			t.Errorf("Metric family %s as unexpected metric count %d", name, len(family.Metric))
		}
	} else {
		t.Errorf("Metric family %s is missing", name)
	}
}

func prepareExporterServer(overrideEventStoreURL string) *ExporterServer {
	eventStoreURL := "http://localhost:2113"
	if overrideEventStoreURL != "" {
		eventStoreURL = overrideEventStoreURL
	} else if os.Getenv("EVENTSTORE_URL") != "" {
		eventStoreURL = os.Getenv("EVENTSTORE_URL")
	}

	config := &config.Config{
		EventStoreURL:      eventStoreURL,
		EventStoreUser:     "admin",
		EventStorePassword: "changeit",
		InsecureSkipVerify: true,
		Timeout:            time.Second * 10,
	}

	client := client.New(config)
	collector := collector.NewCollector(config, client)
	return NewExporterServer(config, collector)
}

func getMetrics(url string, t *testing.T) map[string]*dto.MetricFamily {
	res, err := http.Get(url + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var parser expfmt.TextParser
	mf, err := parser.TextToMetricFamilies(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	return mf
}

func getString(url string, t *testing.T) string {
	res, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	content, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		t.Fatal(err)
	}

	return string(content)
}
