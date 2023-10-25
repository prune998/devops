package main

import "github.com/prometheus/client_golang/prometheus"

var (
	PromThreadGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "cpuconsumer_threads",
		Help: "Number of running threads",
	})
)

func init() {
	prometheus.MustRegister(PromThreadGauge)
}
