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

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	defer provider.Shutdown(context.Background())

	metrics, err := newHTTPMetrics(provider.Meter("test"))
	if err != nil {
		t.Fatalf("create metrics: %v", err)
	}

	router := gin.New()
	router.Use(metrics.Middleware())
	router.GET("/tasks", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/tasks", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	metrics.TaskCount.Add(context.Background(), 1)

	var collected metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &collected); err != nil {
		t.Fatalf("collect metrics: %v", err)
	}

	found := map[string]bool{}
	for _, scope := range collected.ScopeMetrics {
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
