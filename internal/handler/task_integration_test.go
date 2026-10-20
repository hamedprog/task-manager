package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"task-manager/internal/api"
	"task-manager/internal/repository"
	"task-manager/internal/service"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.opentelemetry.io/otel/metric/noop"
)

func TestTaskAPIIntegration(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL is not set; skipping database integration test")
	}

	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer sqlDB.Close()

	ctx := context.Background()
	if err := sqlDB.PingContext(ctx); err != nil {
		t.Fatalf("database ping failed: %v", err)
	}

	taskRepository := repository.NewTaskRepository(sqlDB)

	if _, err := sqlDB.ExecContext(ctx, "TRUNCATE TABLE tasks"); err != nil {
		t.Fatalf("truncate tasks table: %v", err)
	}
	t.Cleanup(func() {
		_, _ = sqlDB.ExecContext(ctx, "TRUNCATE TABLE tasks")
	})

	router := gin.New()
	taskCount, err := noop.NewMeterProvider().Meter("test").Int64Counter("tasks_count")
	if err != nil {
		t.Fatalf("create task count counter: %v", err)
	}
	NewTaskHandler(service.NewTaskService(taskRepository, nil), taskCount).RegisterRoutes(router)

	createResp := doRequest(t, router, http.MethodPost, "/tasks", `{"title":"Write integration test","assignee":"Alex"}`)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d; body: %s", createResp.Code, http.StatusCreated, createResp.Body.String())
	}

	var created api.Task
	if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.Title != "Write integration test" {
		t.Fatalf("created title = %q, want %q", created.Title, "Write integration test")
	}
	if created.Assignee != "Alex" {
		t.Fatalf("created assignee = %q, want %q", created.Assignee, "Alex")
	}
	if created.Id == 0 {
		t.Fatal("created task ID is zero")
	}

	getResp := doRequest(t, router, http.MethodGet, fmt.Sprintf("/tasks/%d", created.Id), "")
	if getResp.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d; body: %s", getResp.Code, http.StatusOK, getResp.Body.String())
	}

	var got api.Task
	if err := json.Unmarshal(getResp.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode get response: %v", err)
	}
	if got.Id != created.Id || got.Title != created.Title || got.Assignee != created.Assignee {
		t.Fatalf("get task = %+v, want %+v", got, created)
	}

	updateResp := doRequest(t, router, http.MethodPatch, fmt.Sprintf("/tasks/%d", created.Id), `{"completed":true,"assignee":"Sam"}`)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("update status = %d, want %d; body: %s", updateResp.Code, http.StatusOK, updateResp.Body.String())
	}

	var updated api.Task
	if err := json.Unmarshal(updateResp.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if !updated.Completed {
		t.Fatal("updated completed = false, want true")
	}
	if updated.Assignee != "Sam" {
		t.Fatalf("updated assignee = %q, want %q", updated.Assignee, "Sam")
	}

	deleteResp := doRequest(t, router, http.MethodDelete, fmt.Sprintf("/tasks/%d", created.Id), "")
	if deleteResp.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d; body: %s", deleteResp.Code, http.StatusNoContent, deleteResp.Body.String())
	}

	missingResp := doRequest(t, router, http.MethodGet, fmt.Sprintf("/tasks/%d", created.Id), "")
	if missingResp.Code != http.StatusNotFound {
		t.Fatalf("missing task status = %d, want %d; body: %s", missingResp.Code, http.StatusNotFound, missingResp.Body.String())
	}
}

func doRequest(t *testing.T, router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	httpRequest := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		httpRequest.Header.Set("Content-Type", "application/json")
	}

	responseRecorder := httptest.NewRecorder()
	router.ServeHTTP(responseRecorder, httpRequest)
	return responseRecorder
}
