package server

import (
	"crypto/tls"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	jp "github.com/buger/jsonparser"
	"github.com/marcinbudny/eventstore_exporter/client"
	"github.com/marcinbudny/eventstore_exporter/collector"
	"github.com/marcinbudny/eventstore_exporter/config"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

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
	} else if os.Getenv("TEST_EVENTSTORE_URL") != "" {
		eventStoreURL = os.Getenv("TEST_EVENTSTORE_URL")
	}

	clusterMode := "single"
	if os.Getenv("TEST_CLUSTER_MODE") != "" {
		clusterMode = os.Getenv("TEST_CLUSTER_MODE")
	}

	config := &config.Config{
		EventStoreURL:      eventStoreURL,
		EventStoreUser:     "admin",
		EventStorePassword: "changeit",
		ClusterMode:        clusterMode,
		InsecureSkipVerify: true,
		Timeout:            time.Second * 10,
	}

	client := client.New(config)
	collector := collector.NewCollector(config, client)
	return NewExporterServer(config, collector)
}

func getEsVersion(t *testing.T) client.EventStoreVersion {

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient := http.Client{
		Transport: tr,
	}

	eventStoreURL := "https://localhost:2113"
	if os.Getenv("TEST_EVENTSTORE_URL") != "" {
		eventStoreURL = os.Getenv("TEST_EVENTSTORE_URL")
	}

	req, _ := http.NewRequest("GET", eventStoreURL+"/info", nil)
	req.SetBasicAuth("admin", "changeit")
	req.Header.Add("Accept", "application/json")
	res, errGet := httpClient.Do(req)

	if errGet != nil {
		t.Fatal(errGet)
	}
	info, errRead := io.ReadAll(res.Body)
	res.Body.Close()
	if errRead != nil {
		t.Fatal(errRead)
	}

	value, _ := jp.GetString(info, "esVersion")
	if value == "" {
		value = "0.0.0.0"
	}
	return client.EventStoreVersion(value)
}
