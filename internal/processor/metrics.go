package processor

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	tasksTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "processor_tasks_total",
			Help: "Total number of image processing attempts by filter and status.",
		},
		[]string{"filter", "status"},
	)

	filterUsageTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "processor_filter_usage_total",
			Help: "Total number of times each image filter was used.",
		},
		[]string{"filter"},
	)

	processingDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "processor_processing_duration_seconds",
			Help: "Time spent processing images in seconds.",
			Buckets: []float64{
				0.01,
				0.05,
				0.1,
				0.25,
				0.5,
				1,
				2,
				5,
				10,
				30,
				60,
			},
		},
		[]string{"filter"},
	)
)

func init() {
	prometheus.MustRegister(
		tasksTotal,
		filterUsageTotal,
		processingDuration,
	)
}

func metricFilterName(name string) string {
	switch name {
	case "Negative", "Blur", "Sharpen", "FlipX":
		return name
	default:
		return "unknown"
	}
}
