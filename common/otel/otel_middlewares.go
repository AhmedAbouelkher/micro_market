package common_otel

import (
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/semconv/v1.20.0/httpconv"
	"go.opentelemetry.io/otel/trace"
)

type statusResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func statusClass(status int) string { return fmt.Sprintf("%dxx", status/100) }

func (t *Telemetry) LogRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := trace.SpanFromContext(r.Context())
		span.SetAttributes(httpconv.ServerRequest(t.GetServiceName(), r)...)
		t.LogInfof("request: %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func (t *Telemetry) MeterRequestDuration(next http.Handler) http.Handler {
	histogram, err := t.MeterInt64Histogram(Metric{
		Name:        "micro_market.http.server.request.duration",
		Unit:        "ms",
		Description: "The duration of the HTTP request",
	})
	if err != nil {
		t.LogFatallnf("failed to create histogram: %v", err)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		st := time.Now()

		next.ServeHTTP(w, r)

		duration := time.Since(st)
		histogram.Record(r.Context(),
			duration.Milliseconds(),
			metric.WithAttributes(
				httpconv.ServerRequest(t.GetServiceName(), r)...,
			),
		)
	})
}

func (t *Telemetry) MeterRequestInFlight(next http.Handler) http.Handler {
	counter, err := t.MeterInt64UpDownCounter(Metric{
		Name:        "micro_market.http.server.active_requests",
		Unit:        "{count}",
		Description: "The number of HTTP requests that are in flight",
	})
	if err != nil {
		t.LogFatallnf("failed to create counter: %v", err)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attrs := metric.WithAttributes(httpconv.ServerRequest(t.GetServiceName(), r)...)

		counter.Add(r.Context(), 1, attrs)
		defer counter.Add(r.Context(), -1, attrs)

		next.ServeHTTP(w, r)

	})
}

func (t *Telemetry) MeterRequestStatus(next http.Handler) http.Handler {
	counter, err := t.MeterInt64Counter(Metric{
		Name:        "micro_market.http.server.response.count",
		Unit:        "{count}",
		Description: "The number of HTTP responses by status code",
	})
	if err != nil {
		t.LogFatallnf("failed to create counter: %v", err)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		srw := &statusResponseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(srw, r)

		counter.Add(r.Context(), 1, metric.WithAttributes(
			attribute.Int("http.response.status_code", srw.status),
			attribute.String("http.response.status_class", statusClass(srw.status)),
		))
	})
}

func (t *Telemetry) MuxMiddleware(next http.Handler) http.Handler {
	h := otelmux.Middleware(
		t.GetServiceName(),
		otelmux.WithTracerProvider(t.tp),
		otelmux.WithMeterProvider(t.mp),
		otelmux.WithPropagators(t.propagators),
	)
	return h(next)
}
