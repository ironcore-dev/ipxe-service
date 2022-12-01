package pkg

import "github.com/prometheus/client_golang/prometheus"

var (
	requestIPXEDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ipxe_request_duration_seconds",
		Help:    "Histogram for the runtime of a simple ipxe(getChainByUUID) function.",
		Buckets: prometheus.LinearBuckets(0.01, 0.05, 10),
	},
		[]string{"mac"},
	)
	requestIGNITIONDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ignition_request_duration_seconds",
		Help:    "Histogram for the runtime of a simple ignition(getIgnitionByUUID) function.",
		Buckets: prometheus.LinearBuckets(0.01, 0.05, 10),
	},
		[]string{"mac"},
	)
)
