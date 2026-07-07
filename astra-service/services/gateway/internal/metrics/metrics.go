// Package metrics exposes a Prometheus instrumentation middleware for the
// gateway. It records request counts and latency distributions labelled by
// method, route pattern and status code.
package metrics

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "astra_gateway_http_request_duration_seconds",
		Help:    "HTTP request latency distribution",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path", "status"})

	requestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "astra_gateway_http_requests_total",
		Help: "Total HTTP requests by method, path and status",
	}, []string{"method", "path", "status"})
)

// Middleware records Prometheus metrics for every request.
func Middleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		duration := time.Since(start)

		status := c.Response().StatusCode()
		if err != nil {
			if fiberErr, ok := err.(*fiber.Error); ok {
				status = fiberErr.Code
			} else if status == fiber.StatusOK {
				status = fiber.StatusInternalServerError
			}
		}

		path := c.Path()
		if route := c.Route(); route != nil && route.Path != "" {
			path = route.Path
		}

		labels := prometheus.Labels{
			"method": c.Method(),
			"path":   path,
			"status": strconv.Itoa(status),
		}

		requestDuration.WithLabelValues(c.Method(), path, labels["status"]).Observe(duration.Seconds())
		requestsTotal.WithLabelValues(c.Method(), path, labels["status"]).Inc()
		return err
	}
}

// Registry returns the default Prometheus registry used by this package.
func Registry() *prometheus.Registry {
	return prometheus.DefaultRegisterer.(*prometheus.Registry)
}
