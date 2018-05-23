package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "eventstore"
	subsystem = ""
)

type exporter struct {
	up                         prometheus.Gauge
}

func newExporter() *exporter {
	return &exporter {
		up:                         createGauge("up", "Whether the EventStore scrape was successful"),
	}
}

func (e *exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.up.Desc()
}

func (e *exporter) Collect(ch chan<- prometheus.Metric) {
	log.Info("Running scrape")

	if stats, err := getStats(); err != nil {
		log.WithError(err).Error("Error while getting data from EventStore")

		e.up.Set(0)
		ch <- e.up

		stats.serverStats = nil
	} else {
		e.up.Set(1)
		ch <- e.up
	}
}

func createGauge(name string, help string) prometheus.Gauge {
	return prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      name,
		Help:      help,
	})
}

func createDatabaseGaugeVec(name string, help string) *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      name,
		Help:      help,
	}, []string{"database"})
}

func createCounter(name string, help string) prometheus.Counter {
	return prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      name,
		Help:      help,
	})
}

func createDatabaseCounterVec(name string, help string) *prometheus.CounterVec {
	return prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      name,
		Help:      help,
	}, []string{"database"})
}