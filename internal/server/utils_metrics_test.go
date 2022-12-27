package server

import (
	"io"
	"net/http"
	"strings"
	"testing"

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

func getStringFromExporterEndpoint(url string, t *testing.T) string {
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

func assertHasMetric(
	t *testing.T,
	metrics map[string]*dto.MetricFamily,
	name string,
	metricType string,
) {
	assertMetric(t, metrics, name, metricType, anyMetric, anyValue)
}

func assertMetric(
	t *testing.T,
	metrics map[string]*dto.MetricFamily,
	name string,
	metricType string,
	selectMetric func(*testing.T, []*dto.Metric) *dto.Metric,
	assertValue func(*testing.T, float64),
) {
	t.Logf("Asserting metric %s", name)
	if family, ok := metrics[name]; ok {
		metric := selectMetric(t, family.Metric)
		if metric == nil {
			t.Errorf("Metric %s matching the specification was not found", name)
			return
		}
		switch strings.ToLower(metricType) {
		case "counter":
			if metric.Counter != nil {
				assertValue(t, *metric.Counter.Value)
			} else {
				t.Errorf("Metric %s is not a counter", name)
			}

		case "gauge":
			if metric.Gauge != nil {
				assertValue(t, *metric.Gauge.Value)
			} else {
				t.Errorf("Metric %s is not a gauge", name)
			}

		default:
			t.Fatalf("Unsupported metric type %s", metricType)
		}
	} else {
		t.Errorf("Metric family %s is missing", name)
	}
}

func assertHasNoMetric(t *testing.T, metrics map[string]*dto.MetricFamily, name string) {
	if _, ok := metrics[name]; ok {
		t.Errorf("Metric family %s is present but should not be", name)
	}
}

func anyValue(*testing.T, float64) {
}

func nonZeroValue(t *testing.T, actualValue float64) {
	if actualValue == 0.0 {
		t.Errorf("Expected non-zero value")
	}
}

func hasValue(expectedValue float64) func(*testing.T, float64) {
	return func(t *testing.T, actualValue float64) {
		if expectedValue != actualValue {
			t.Errorf("Expected metric value to be %v but is actually %v", expectedValue, actualValue)
		}
	}
}

func singleValuedMetric(t *testing.T, metrics []*dto.Metric) *dto.Metric {
	if len(metrics) != 1 {
		t.Errorf("Expected single valued metric, but %d metrics are in the family", len(metrics))
	}
	return metrics[0]
}

func anyMetric(t *testing.T, metrics []*dto.Metric) *dto.Metric {
	if len(metrics) == 0 {
		t.Error("No metrics in the family")
	}
	return metrics[0]
}

func metricByLabelValue(name string, value string) func(*testing.T, []*dto.Metric) *dto.Metric {
	return func(t *testing.T, metrics []*dto.Metric) *dto.Metric {
		for _, metric := range metrics {
			for _, label := range metric.Label {
				if *label.Name == name && *label.Value == value {
					return metric
				}
			}
		}
		t.Errorf("Did not find metric with name = %s and value = %s", name, value)
		return nil
	}
}
