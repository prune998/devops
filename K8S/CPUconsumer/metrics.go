package main

import "github.com/prometheus/client_golang/prometheus"

var (
	PromThreadGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "cpuconsumer_threads",
		Help: "Number of running threads",
	})

	PromOpsCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cpuconsumer_ops",
		Help: "Number of operation performed",
	}, []string{"gomaxprocs"},
	)
)

func init() {
	prometheus.MustRegister(PromThreadGauge)
	prometheus.MustRegister(PromOpsCounter)
}
