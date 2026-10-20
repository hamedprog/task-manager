package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

type HTTPMetrics struct {
	requests  metric.Int64Counter
	durations metric.Float64Histogram
	TaskCount metric.Int64Counter
}

func InitMetrics(ctx context.Context, serviceName string) (*HTTPMetrics, func(context.Context) error, error) {
	httpExporter, err := otlpmetrichttp.New(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("create OTLP metrics exporter: %w", err)
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(httpExporter, sdkmetric.WithInterval(10*time.Second))),
		sdkmetric.WithResource(resource.NewSchemaless(
			attribute.String("service.name", serviceName),
		)),
	)
	httpMetrics, err := newHTTPMetrics(meterProvider.Meter(serviceName))
	if err != nil {
		if shutdownErr := meterProvider.Shutdown(ctx); shutdownErr != nil {
			return nil, nil, fmt.Errorf("create metrics instruments: %w (shutdown provider: %v)", err, shutdownErr)
		}
		return nil, nil, fmt.Errorf("create metrics instruments: %w", err)
	}

	return httpMetrics, meterProvider.Shutdown, nil
}

func newHTTPMetrics(meter metric.Meter) (*HTTPMetrics, error) {
	requests, err := meter.Int64Counter("http_requests")
	if err != nil {
		return nil, fmt.Errorf("create HTTP request counter: %w", err)
	}
	durations, err := meter.Float64Histogram(
		"http_request_duration",
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10),
	)
	if err != nil {
		return nil, fmt.Errorf("create HTTP request duration histogram: %w", err)
	}
	taskCount, err := meter.Int64Counter("tasks_count")
	if err != nil {
		return nil, fmt.Errorf("create task count counter: %w", err)
	}

	return &HTTPMetrics{requests: requests, durations: durations, TaskCount: taskCount}, nil
}

func (m *HTTPMetrics) Middleware() gin.HandlerFunc {
	return func(ginContext *gin.Context) {
		start := time.Now()
		ginContext.Next()

		route := ginContext.FullPath()
		if route == "" {
			route = "unmatched"
		}
		method := ginContext.Request.Method
		switch method {
		case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
			http.MethodPatch, http.MethodDelete, http.MethodConnect, http.MethodOptions, http.MethodTrace:
		default:
			method = "_OTHER"
		}
		attrs := metric.WithAttributes(
			attribute.String("http.request.method", method),
			attribute.String("http.route", route),
			attribute.Int("http.response.status_code", ginContext.Writer.Status()),
		)
		m.requests.Add(ginContext.Request.Context(), 1, attrs)
		m.durations.Record(ginContext.Request.Context(), time.Since(start).Seconds(), attrs)
	}
}
