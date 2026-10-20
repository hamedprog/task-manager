package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"task-manager/internal/api"
	"task-manager/internal/object/bo"
	"task-manager/internal/object/dao"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/metric/noop"
	"go.uber.org/mock/gomock"
)

func newMockTaskHandler(t *testing.T) (*MockTaskService, *TaskHandler) {
	t.Helper()

	mockTaskService := NewMockTaskService(gomock.NewController(t))
	taskCount, err := noop.NewMeterProvider().Meter("test").Int64Counter("tasks_count")
	if err != nil {
		t.Fatalf("create task count counter: %v", err)
	}
	return mockTaskService, NewTaskHandler(mockTaskService, taskCount)
}

func stringPointer(value string) *string {
	return &value
}

func boolPointer(value bool) *bool {
	return &value
}

func newTestContext(method, body string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	ResponseRecorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(ResponseRecorder)
	ginContext.Request = httptest.NewRequest(method, "/", strings.NewReader(body))
	return ginContext, ResponseRecorder
}

func TestTaskHandlerCreateTask(t *testing.T) {
	t.Run("created", func(t *testing.T) {
		mockTaskService, taskHandler := newMockTaskHandler(t)
		mockTaskService.EXPECT().
			Create(gomock.Eq(bo.Task{Title: stringPointer("Write tests"), Assignee: stringPointer("Alex")})).
			Return(&bo.Task{ID: 3, Title: stringPointer("Write tests"), Completed: boolPointer(false), Assignee: stringPointer("Alex")}, nil)
		ginContext, responseRecorder := newTestContext(http.MethodPost, `{"title":"Write tests","assignee":"Alex"}`)

		taskHandler.CreateTask(ginContext)

		if responseRecorder.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d", responseRecorder.Code, http.StatusCreated)
		}
		var got api.Task
		if err := json.Unmarshal(responseRecorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got != (api.Task{Id: 3, Title: "Write tests", Assignee: "Alex"}) {
			t.Errorf("response = %+v, want created task", got)
		}
	})

	t.Run("missing assignee defaults to empty", func(t *testing.T) {
		mockTaskService, taskHandler := newMockTaskHandler(t)
		mockTaskService.EXPECT().
			Create(gomock.Eq(bo.Task{Title: stringPointer("Write tests"), Assignee: stringPointer("")})).
			Return(&bo.Task{ID: 3, Title: stringPointer("Write tests"), Completed: boolPointer(false), Assignee: stringPointer("")}, nil)
		ginContext, responseRecorder := newTestContext(http.MethodPost, `{"title":"Write tests"}`)

		taskHandler.CreateTask(ginContext)

		if responseRecorder.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d", responseRecorder.Code, http.StatusCreated)
		}
		var got api.Task
		if err := json.Unmarshal(responseRecorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.Assignee != "" {
			t.Errorf("response assignee = %q, want empty", got.Assignee)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		_, taskHandler := newMockTaskHandler(t)
		ginContext, responseRecorder := newTestContext(http.MethodPost, `{`)

		taskHandler.CreateTask(ginContext)

		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", responseRecorder.Code, http.StatusBadRequest)
		}
	})

	t.Run("empty title", func(t *testing.T) {
		mockTaskService, taskHandler := newMockTaskHandler(t)
		mockTaskService.EXPECT().
			Create(gomock.Any()).
			Return(nil, errors.New("title cannot be empty"))
		ginContext, responseRecorder := newTestContext(http.MethodPost, `{"title":" "}`)

		taskHandler.CreateTask(ginContext)

		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", responseRecorder.Code, http.StatusBadRequest)
		}
		var got api.Error
		if err := json.Unmarshal(responseRecorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.Error != "title cannot be empty" {
			t.Errorf("error = %q, want %q", got.Error, "title cannot be empty")
		}
	})

	t.Run("service failure", func(t *testing.T) {
		mockTaskService, taskHandler := newMockTaskHandler(t)
		mockTaskService.EXPECT().
			Create(gomock.Any()).
			Return(nil, errors.New("database unavailable"))
		ginContext, responseRecorder := newTestContext(http.MethodPost, `{"title":"Task"}`)

		taskHandler.CreateTask(ginContext)

		if responseRecorder.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", responseRecorder.Code, http.StatusInternalServerError)
		}
	})
}

func TestTaskHandlerListTasks(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockTaskService, taskHandler := newMockTaskHandler(t)
		completed := false
		limit := 5
		offset := 10
		mockTaskService.EXPECT().List(
			dao.TaskFilter{Assignee: stringPointer("Alex"), Completed: &completed},
			dao.Cursor{Limit: &limit, Offset: &offset},
		).Return([]bo.Task{
			{ID: 1, Title: stringPointer("First"), Completed: boolPointer(false), Assignee: stringPointer("Alex")},
			{ID: 2, Title: stringPointer("Second"), Completed: boolPointer(true), Assignee: stringPointer("Sam")},
		}, nil)
		ginContext, responseRecorder := newTestContext(http.MethodGet, "")
		ginContext.Request.URL.RawQuery = "assignee=Alex&completed=false&limit=5&offset=10"

		taskHandler.ListTasks(ginContext, api.ListTasksParams{
			Assignee:  stringPointer("Alex"),
			Completed: &completed,
			Limit:     &limit,
			Offset:    &offset,
		})

		if responseRecorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", responseRecorder.Code, http.StatusOK)
		}
		var got []api.Task
		if err := json.Unmarshal(responseRecorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		want := []api.Task{
			{Id: 1, Title: "First", Assignee: "Alex"},
			{Id: 2, Title: "Second", Completed: true, Assignee: "Sam"},
		}
		if len(got) != len(want) {
			t.Fatalf("response has %d tasks, want %d", len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("response[%d] = %+v, want %+v", i, got[i], want[i])
			}
		}
	})

	t.Run("service failure", func(t *testing.T) {
		mockTaskService, taskHandler := newMockTaskHandler(t)
		mockTaskService.EXPECT().List(dao.TaskFilter{}, dao.Cursor{}).Return(nil, errors.New("database unavailable"))
		ginContext, responseRecorder := newTestContext(http.MethodGet, "")

		taskHandler.ListTasks(ginContext, api.ListTasksParams{})

		if responseRecorder.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", responseRecorder.Code, http.StatusInternalServerError)
		}
	})

	t.Run("invalid cursor values", func(t *testing.T) {
		_, taskHandler := newMockTaskHandler(t)
		limit := 0
		ginContext, responseRecorder := newTestContext(http.MethodGet, "")

		taskHandler.ListTasks(ginContext, api.ListTasksParams{Limit: &limit})

		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", responseRecorder.Code, http.StatusBadRequest)
		}
	})

	t.Run("invalid completed filter", func(t *testing.T) {
		_, taskHandler := newMockTaskHandler(t)
		router := gin.New()
		taskHandler.RegisterRoutes(router)
		responseRecorder := httptest.NewRecorder()
		router.ServeHTTP(responseRecorder, httptest.NewRequest(http.MethodGet, "/tasks?completed=sometimes", nil))

		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", responseRecorder.Code, http.StatusBadRequest)
		}
		var got api.Error
		if err := json.Unmarshal(responseRecorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.Error != "invalid task filter" {
			t.Errorf("error = %q, want invalid task filter", got.Error)
		}
	})
}

func TestTaskHandlerGetTask(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockTaskService, taskHandler := newMockTaskHandler(t)
		mockTaskService.EXPECT().
			GetByID(4).
			Return(&bo.Task{ID: 4, Title: stringPointer("Read docs"), Completed: boolPointer(true), Assignee: stringPointer("Alex")}, nil)
		ginContext, responseRecorder := newTestContext(http.MethodGet, "")

		taskHandler.GetTask(ginContext, 4)

		if responseRecorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", responseRecorder.Code, http.StatusOK)
		}
		var got api.Task
		if err := json.Unmarshal(responseRecorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got != (api.Task{Id: 4, Title: "Read docs", Completed: true, Assignee: "Alex"}) {
			t.Errorf("response = %+v, want task 4", got)
		}
	})

	t.Run("invalid ID", func(t *testing.T) {
		_, taskHandler := newMockTaskHandler(t)
		ginContext, responseRecorder := newTestContext(http.MethodGet, "")

		taskHandler.GetTask(ginContext, 0)

		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", responseRecorder.Code, http.StatusBadRequest)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mockTaskService, taskHandler := newMockTaskHandler(t)
		mockTaskService.EXPECT().GetByID(4).Return(nil, errors.New("missing"))
		ginContext, responseRecorder := newTestContext(http.MethodGet, "")

		taskHandler.GetTask(ginContext, 4)

		if responseRecorder.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", responseRecorder.Code, http.StatusNotFound)
		}
	})
}

func TestTaskHandlerUpdateTask(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockTaskService, taskHandler := newMockTaskHandler(t)
		mockTaskService.EXPECT().
			Update(gomock.Eq(bo.Task{
				ID:        4,
				Title:     stringPointer("Updated"),
				Completed: boolPointer(true),
				Assignee:  stringPointer("Alex"),
			})).
			Return(&bo.Task{ID: 4, Title: stringPointer("Updated"), Completed: boolPointer(true), Assignee: stringPointer("Alex")}, nil)
		ginContext, responseRecorder := newTestContext(http.MethodPatch, `{"title":"Updated","completed":true,"assignee":"Alex"}`)

		taskHandler.UpdateTask(ginContext, 4)

		if responseRecorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", responseRecorder.Code, http.StatusOK)
		}
		var got api.Task
		if err := json.Unmarshal(responseRecorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got != (api.Task{Id: 4, Title: "Updated", Completed: true, Assignee: "Alex"}) {
			t.Errorf("response = %+v, want updated task", got)
		}
	})

	t.Run("assignee only", func(t *testing.T) {
		mockTaskService, taskHandler := newMockTaskHandler(t)
		mockTaskService.EXPECT().
			Update(gomock.Eq(bo.Task{ID: 4, Assignee: stringPointer("Sam")})).
			Return(&bo.Task{
				ID:        4,
				Title:     stringPointer("Existing task"),
				Completed: boolPointer(false),
				Assignee:  stringPointer("Sam"),
			}, nil)
		ginContext, responseRecorder := newTestContext(http.MethodPatch, `{"assignee":"Sam"}`)

		taskHandler.UpdateTask(ginContext, 4)

		if responseRecorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", responseRecorder.Code, http.StatusOK)
		}
		var got api.Task
		if err := json.Unmarshal(responseRecorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.Assignee != "Sam" {
			t.Errorf("response assignee = %q, want %q", got.Assignee, "Sam")
		}
	})

	t.Run("invalid ID", func(t *testing.T) {
		_, taskHandler := newMockTaskHandler(t)
		ginContext, responseRecorder := newTestContext(http.MethodPatch, `{"title":"Updated"}`)

		taskHandler.UpdateTask(ginContext, 0)

		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", responseRecorder.Code, http.StatusBadRequest)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		_, taskHandler := newMockTaskHandler(t)
		ginContext, responseRecorder := newTestContext(http.MethodPatch, `{`)

		taskHandler.UpdateTask(ginContext, 4)

		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", responseRecorder.Code, http.StatusBadRequest)
		}
	})

	t.Run("no fields to update", func(t *testing.T) {
		_, taskHandler := newMockTaskHandler(t)
		ginContext, responseRecorder := newTestContext(http.MethodPatch, `{}`)

		taskHandler.UpdateTask(ginContext, 4)

		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", responseRecorder.Code, http.StatusBadRequest)
		}
		var got api.Error
		if err := json.Unmarshal(responseRecorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.Error != "no fields to update" {
			t.Errorf("error = %q, want %q", got.Error, "no fields to update")
		}
	})

	t.Run("empty title", func(t *testing.T) {
		mockTaskService, taskHandler := newMockTaskHandler(t)
		mockTaskService.EXPECT().
			Update(gomock.Any()).
			Return(nil, errors.New("title cannot be empty"))
		ginContext, responseRecorder := newTestContext(http.MethodPatch, `{"title":" "}`)

		taskHandler.UpdateTask(ginContext, 4)

		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", responseRecorder.Code, http.StatusBadRequest)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mockTaskService, taskHandler := newMockTaskHandler(t)
		mockTaskService.EXPECT().
			Update(gomock.Any()).
			Return(nil, errors.New("task not found"))
		ginContext, responseRecorder := newTestContext(http.MethodPatch, `{"completed":true}`)

		taskHandler.UpdateTask(ginContext, 4)

		if responseRecorder.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", responseRecorder.Code, http.StatusNotFound)
		}
	})

	t.Run("service failure", func(t *testing.T) {
		mockTaskService, taskHandler := newMockTaskHandler(t)
		mockTaskService.EXPECT().
			Update(gomock.Any()).
			Return(nil, errors.New("database unavailable"))
		ginContext, responseRecorder := newTestContext(http.MethodPatch, `{"completed":true}`)

		taskHandler.UpdateTask(ginContext, 4)

		if responseRecorder.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", responseRecorder.Code, http.StatusInternalServerError)
		}
	})
}

func TestTaskHandlerDeleteTask(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockTaskService, taskHandler := newMockTaskHandler(t)
		mockTaskService.EXPECT().Delete(4).Return(nil)
		ginContext, responseRecorder := newTestContext(http.MethodDelete, "")

		taskHandler.DeleteTask(ginContext, 4)
		ginContext.Writer.WriteHeaderNow()

		if responseRecorder.Code != http.StatusNoContent {
			t.Errorf("status = %d, want %d", responseRecorder.Code, http.StatusNoContent)
		}
		if responseRecorder.Body.Len() != 0 {
			t.Errorf("response body = %q, want empty", responseRecorder.Body.String())
		}
	})

	t.Run("invalid ID", func(t *testing.T) {
		_, taskHandler := newMockTaskHandler(t)
		ginContext, responseRecorder := newTestContext(http.MethodDelete, "")

		taskHandler.DeleteTask(ginContext, 0)

		if responseRecorder.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", responseRecorder.Code, http.StatusBadRequest)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mockTaskService, taskHandler := newMockTaskHandler(t)
		mockTaskService.EXPECT().Delete(4).Return(errors.New("missing"))
		ginContext, responseRecorder := newTestContext(http.MethodDelete, "")

		taskHandler.DeleteTask(ginContext, 4)

		if responseRecorder.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", responseRecorder.Code, http.StatusNotFound)
		}
	})
}
