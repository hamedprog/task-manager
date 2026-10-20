package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestHTTPMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)

	manualReader := sdkmetric.NewManualReader()
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(manualReader))
	defer meterProvider.Shutdown(context.Background())

	httpMetrics, err := newHTTPMetrics(meterProvider.Meter("test"))
	if err != nil {
		t.Fatalf("create metrics: %v", err)
	}

	router := gin.New()
	router.Use(httpMetrics.Middleware())
	router.GET("/tasks", func(ginContext *gin.Context) {
		ginContext.Status(http.StatusOK)
	})
	responseRecorder := httptest.NewRecorder()
	router.ServeHTTP(responseRecorder, httptest.NewRequest(http.MethodGet, "/tasks", nil))

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", responseRecorder.Code, http.StatusOK)
	}
	httpMetrics.TaskCount.Add(context.Background(), 1)

	var resourceMetrics metricdata.ResourceMetrics
	if err := manualReader.Collect(context.Background(), &resourceMetrics); err != nil {
		t.Fatalf("collect metrics: %v", err)
	}

	found := map[string]bool{}
	for _, scope := range resourceMetrics.ScopeMetrics {
		for _, metric := range scope.Metrics {
			switch data := metric.Data.(type) {
			case metricdata.Sum[int64]:
				if len(data.DataPoints) == 1 && data.DataPoints[0].Value == 1 {
					found[metric.Name] = true
				}
			case metricdata.Histogram[float64]:
				if len(data.DataPoints) == 1 && data.DataPoints[0].Count == 1 {
					found[metric.Name] = true
				}
			}
		}
	}

	for _, name := range []string{"http_requests", "http_request_duration", "tasks_count"} {
		if !found[name] {
			t.Errorf("metric %q was not recorded", name)
		}
	}
}
