package telemetry

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
)

func TestInitTracingExportsSpansOverOTLPHTTP(t *testing.T) {
	type exportRequest struct {
		method string
		path   string
		size   int
	}
	requests := make(chan exportRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read OTLP request body: %v", err)
		}
		requests <- exportRequest{method: r.Method, path: r.URL.Path, size: len(body)}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", server.URL)

	shutdown, err := InitTracing(context.Background(), "test-service")
	if err != nil {
		t.Fatalf("InitTracing() error = %v", err)
	}

	_, span := otel.Tracer("test").Start(context.Background(), "test-span")
	span.End()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := shutdown(ctx); err != nil {
		t.Fatalf("shutdown tracer provider: %v", err)
	}

	select {
	case request := <-requests:
		if request.method != http.MethodPost {
			t.Errorf("request method = %s, want POST", request.method)
		}
		if request.path != "/v1/traces" {
			t.Errorf("request path = %s, want /v1/traces", request.path)
		}
		if request.size == 0 {
			t.Error("OTLP trace request has an empty body")
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for OTLP trace export")
	}
}
