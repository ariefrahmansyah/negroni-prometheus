package negroniprometheus

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/urfave/negroni"
)

var (
	dflBuckets = []float64{300, 1000, 2500, 5000}
)

const (
	requestName = "requests_total"
	latencyName = "request_duration_milliseconds"
)

// PromMiddlewareOpts specifies options how to create new PromMiddleware.
type PromMiddlewareOpts struct {
	// Buckets specifies an custom buckets to be used in request histograpm.
	Buckets []float64
}

// PromMiddleware is a handler that exposes prometheus metrics for the number of requests,
// the latency and the response size, partitioned by status code, method and HTTP path.
type PromMiddleware struct {
	request *prometheus.CounterVec
	latency *prometheus.HistogramVec
}

// NewPromMiddleware returns a new PromMiddleware handler.
// You can use other buckets than the default (300, 1000, 2500, 5000).
func NewPromMiddleware(namespace string, opt PromMiddlewareOpts) *PromMiddleware {
	var pm PromMiddleware

	pm.request = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      requestName,
			Help:      "How many HTTP requests processed, partitioned by status code, method and HTTP path.",
		},
		[]string{"code", "method", "path"},
	)
	prometheus.MustRegister(pm.request)

	buckets := opt.Buckets
	if len(buckets) == 0 {
		buckets = dflBuckets
	}
	pm.latency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      latencyName,
		Help:      "How long it took to process the request, partitioned by status code, method and HTTP path.",
		Buckets:   buckets,
	},
		[]string{"code", "method", "path"},
	)
	prometheus.MustRegister(pm.latency)

	return &pm
}

func (pm *PromMiddleware) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	start := time.Now()

	res := negroni.NewResponseWriter(rw)

	next(rw, r)

	go pm.request.WithLabelValues(fmt.Sprint(res.Status()), r.Method, r.URL.Path).Inc()
	go pm.latency.WithLabelValues(fmt.Sprint(res.Status()), r.Method, r.URL.Path).Observe(float64(time.Since(start).Nanoseconds()) / 1000000)
}
