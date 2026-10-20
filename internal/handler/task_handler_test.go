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

	service := NewMockTaskService(gomock.NewController(t))
	taskCount, err := noop.NewMeterProvider().Meter("test").Int64Counter("tasks_count")
	if err != nil {
		t.Fatalf("create task count counter: %v", err)
	}
	return service, NewTaskHandler(service, taskCount)
}

func stringPointer(value string) *string {
	return &value
}

func boolPointer(value bool) *bool {
	return &value
}

func newTestContext(method, body string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(method, "/", strings.NewReader(body))
	return context, recorder
}

func TestTaskHandlerCreateTask(t *testing.T) {
	t.Run("created", func(t *testing.T) {
		service, handler := newMockTaskHandler(t)
		service.EXPECT().
			Create(gomock.Eq(bo.Task{Title: stringPointer("Write tests"), Assignee: stringPointer("Alex")})).
			Return(&bo.Task{ID: 3, Title: stringPointer("Write tests"), Completed: boolPointer(false), Assignee: stringPointer("Alex")}, nil)
		context, recorder := newTestContext(http.MethodPost, `{"title":"Write tests","assignee":"Alex"}`)

		handler.CreateTask(context)

		if recorder.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusCreated)
		}
		var got api.Task
		if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got != (api.Task{Id: 3, Title: "Write tests", Assignee: "Alex"}) {
			t.Errorf("response = %+v, want created task", got)
		}
	})

	t.Run("missing assignee defaults to empty", func(t *testing.T) {
		service, handler := newMockTaskHandler(t)
		service.EXPECT().
			Create(gomock.Eq(bo.Task{Title: stringPointer("Write tests"), Assignee: stringPointer("")})).
			Return(&bo.Task{ID: 3, Title: stringPointer("Write tests"), Completed: boolPointer(false), Assignee: stringPointer("")}, nil)
		context, recorder := newTestContext(http.MethodPost, `{"title":"Write tests"}`)

		handler.CreateTask(context)

		if recorder.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusCreated)
		}
		var got api.Task
		if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.Assignee != "" {
			t.Errorf("response assignee = %q, want empty", got.Assignee)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		_, handler := newMockTaskHandler(t)
		context, recorder := newTestContext(http.MethodPost, `{`)

		handler.CreateTask(context)

		if recorder.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
		}
	})

	t.Run("empty title", func(t *testing.T) {
		service, handler := newMockTaskHandler(t)
		service.EXPECT().
			Create(gomock.Any()).
			Return(nil, errors.New("title cannot be empty"))
		context, recorder := newTestContext(http.MethodPost, `{"title":" "}`)

		handler.CreateTask(context)

		if recorder.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
		}
		var got api.Error
		if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.Error != "title cannot be empty" {
			t.Errorf("error = %q, want %q", got.Error, "title cannot be empty")
		}
	})

	t.Run("service failure", func(t *testing.T) {
		service, handler := newMockTaskHandler(t)
		service.EXPECT().
			Create(gomock.Any()).
			Return(nil, errors.New("database unavailable"))
		context, recorder := newTestContext(http.MethodPost, `{"title":"Task"}`)

		handler.CreateTask(context)

		if recorder.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
		}
	})
}

func TestTaskHandlerListTasks(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		service, handler := newMockTaskHandler(t)
		completed := false
		service.EXPECT().List(dao.TaskFilter{Assignee: stringPointer("Alex"), Completed: &completed}).Return([]bo.Task{
			{ID: 1, Title: stringPointer("First"), Completed: boolPointer(false), Assignee: stringPointer("Alex")},
			{ID: 2, Title: stringPointer("Second"), Completed: boolPointer(true), Assignee: stringPointer("Sam")},
		}, nil)
		context, recorder := newTestContext(http.MethodGet, "")
		context.Request.URL.RawQuery = "assignee=Alex&completed=false"

		handler.ListTasks(context, api.ListTasksParams{
			Assignee:  stringPointer("Alex"),
			Completed: &completed,
		})

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
		}
		var got []api.Task
		if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
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
		service, handler := newMockTaskHandler(t)
		service.EXPECT().List(dao.TaskFilter{}).Return(nil, errors.New("database unavailable"))
		context, recorder := newTestContext(http.MethodGet, "")

		handler.ListTasks(context, api.ListTasksParams{})

		if recorder.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
		}
	})

	t.Run("invalid completed filter", func(t *testing.T) {
		_, handler := newMockTaskHandler(t)
		router := gin.New()
		handler.RegisterRoutes(router)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/tasks?completed=sometimes", nil))

		if recorder.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
		}
		var got api.Error
		if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.Error != "invalid task filter" {
			t.Errorf("error = %q, want invalid task filter", got.Error)
		}
	})
}

func TestTaskHandlerGetTask(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		service, handler := newMockTaskHandler(t)
		service.EXPECT().
			GetByID(4).
			Return(&bo.Task{ID: 4, Title: stringPointer("Read docs"), Completed: boolPointer(true), Assignee: stringPointer("Alex")}, nil)
		context, recorder := newTestContext(http.MethodGet, "")

		handler.GetTask(context, 4)

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
		}
		var got api.Task
		if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got != (api.Task{Id: 4, Title: "Read docs", Completed: true, Assignee: "Alex"}) {
			t.Errorf("response = %+v, want task 4", got)
		}
	})

	t.Run("invalid ID", func(t *testing.T) {
		_, handler := newMockTaskHandler(t)
		context, recorder := newTestContext(http.MethodGet, "")

		handler.GetTask(context, 0)

		if recorder.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
		}
	})

	t.Run("not found", func(t *testing.T) {
		service, handler := newMockTaskHandler(t)
		service.EXPECT().GetByID(4).Return(nil, errors.New("missing"))
		context, recorder := newTestContext(http.MethodGet, "")

		handler.GetTask(context, 4)

		if recorder.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", recorder.Code, http.StatusNotFound)
		}
	})
}

func TestTaskHandlerUpdateTask(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		service, handler := newMockTaskHandler(t)
		service.EXPECT().
			Update(gomock.Eq(bo.Task{
				ID:        4,
				Title:     stringPointer("Updated"),
				Completed: boolPointer(true),
				Assignee:  stringPointer("Alex"),
			})).
			Return(&bo.Task{ID: 4, Title: stringPointer("Updated"), Completed: boolPointer(true), Assignee: stringPointer("Alex")}, nil)
		context, recorder := newTestContext(http.MethodPatch, `{"title":"Updated","completed":true,"assignee":"Alex"}`)

		handler.UpdateTask(context, 4)

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
		}
		var got api.Task
		if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got != (api.Task{Id: 4, Title: "Updated", Completed: true, Assignee: "Alex"}) {
			t.Errorf("response = %+v, want updated task", got)
		}
	})

	t.Run("assignee only", func(t *testing.T) {
		service, handler := newMockTaskHandler(t)
		service.EXPECT().
			Update(gomock.Eq(bo.Task{ID: 4, Assignee: stringPointer("Sam")})).
			Return(&bo.Task{
				ID:        4,
				Title:     stringPointer("Existing task"),
				Completed: boolPointer(false),
				Assignee:  stringPointer("Sam"),
			}, nil)
		context, recorder := newTestContext(http.MethodPatch, `{"assignee":"Sam"}`)

		handler.UpdateTask(context, 4)

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
		}
		var got api.Task
		if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.Assignee != "Sam" {
			t.Errorf("response assignee = %q, want %q", got.Assignee, "Sam")
		}
	})

	t.Run("invalid ID", func(t *testing.T) {
		_, handler := newMockTaskHandler(t)
		context, recorder := newTestContext(http.MethodPatch, `{"title":"Updated"}`)

		handler.UpdateTask(context, 0)

		if recorder.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		_, handler := newMockTaskHandler(t)
		context, recorder := newTestContext(http.MethodPatch, `{`)

		handler.UpdateTask(context, 4)

		if recorder.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
		}
	})

	t.Run("no fields to update", func(t *testing.T) {
		_, handler := newMockTaskHandler(t)
		context, recorder := newTestContext(http.MethodPatch, `{}`)

		handler.UpdateTask(context, 4)

		if recorder.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
		}
		var got api.Error
		if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.Error != "no fields to update" {
			t.Errorf("error = %q, want %q", got.Error, "no fields to update")
		}
	})

	t.Run("empty title", func(t *testing.T) {
		service, handler := newMockTaskHandler(t)
		service.EXPECT().
			Update(gomock.Any()).
			Return(nil, errors.New("title cannot be empty"))
		context, recorder := newTestContext(http.MethodPatch, `{"title":" "}`)

		handler.UpdateTask(context, 4)

		if recorder.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
		}
	})

	t.Run("not found", func(t *testing.T) {
		service, handler := newMockTaskHandler(t)
		service.EXPECT().
			Update(gomock.Any()).
			Return(nil, errors.New("task not found"))
		context, recorder := newTestContext(http.MethodPatch, `{"completed":true}`)

		handler.UpdateTask(context, 4)

		if recorder.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", recorder.Code, http.StatusNotFound)
		}
	})

	t.Run("service failure", func(t *testing.T) {
		service, handler := newMockTaskHandler(t)
		service.EXPECT().
			Update(gomock.Any()).
			Return(nil, errors.New("database unavailable"))
		context, recorder := newTestContext(http.MethodPatch, `{"completed":true}`)

		handler.UpdateTask(context, 4)

		if recorder.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
		}
	})
}

func TestTaskHandlerDeleteTask(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		service, handler := newMockTaskHandler(t)
		service.EXPECT().Delete(4).Return(nil)
		context, recorder := newTestContext(http.MethodDelete, "")

		handler.DeleteTask(context, 4)
		context.Writer.WriteHeaderNow()

		if recorder.Code != http.StatusNoContent {
			t.Errorf("status = %d, want %d", recorder.Code, http.StatusNoContent)
		}
		if recorder.Body.Len() != 0 {
			t.Errorf("response body = %q, want empty", recorder.Body.String())
		}
	})

	t.Run("invalid ID", func(t *testing.T) {
		_, handler := newMockTaskHandler(t)
		context, recorder := newTestContext(http.MethodDelete, "")

		handler.DeleteTask(context, 0)

		if recorder.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
		}
	})

	t.Run("not found", func(t *testing.T) {
		service, handler := newMockTaskHandler(t)
		service.EXPECT().Delete(4).Return(errors.New("missing"))
		context, recorder := newTestContext(http.MethodDelete, "")

		handler.DeleteTask(context, 4)

		if recorder.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", recorder.Code, http.StatusNotFound)
		}
	})
}
