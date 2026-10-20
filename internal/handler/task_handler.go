package handler

import (
	"net/http"
	"strings"

	"task-manager/internal/api"
	"task-manager/internal/object/bo"
	"task-manager/internal/object/dao"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/metric"
)

type TaskService interface {
	Create(task bo.Task) (*bo.Task, error)
	List(filter dao.TaskFilter) ([]bo.Task, error)
	GetByID(id int) (*bo.Task, error)
	Update(task bo.Task) (*bo.Task, error)
	Delete(id int) error
}

type TaskHandler struct {
	service   TaskService
	taskCount metric.Int64Counter
}

func NewTaskHandler(service TaskService, taskCount metric.Int64Counter) *TaskHandler {
	return &TaskHandler{service: service, taskCount: taskCount}
}

func (h *TaskHandler) RegisterRoutes(router gin.IRouter) {
	api.RegisterHandlersWithOptions(router, h, api.GinServerOptions{
		ErrorHandler: func(c *gin.Context, err error, statusCode int) {
			message := "invalid task filter"
			if strings.Contains(err.Error(), "parameter id") {
				message = "invalid task ID"
			}
			c.JSON(statusCode, api.Error{Error: message})
		},
	})
}

func (h *TaskHandler) CreateTask(c *gin.Context) {
	var req api.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Error: err.Error()})
		return
	}

	assignee := ""
	if req.Assignee != nil {
		assignee = *req.Assignee
	}

	boTask, err := h.service.Create(bo.Task{Title: &req.Title, Assignee: &assignee})
	if err != nil {
		if err.Error() == "title cannot be empty" {
			c.JSON(http.StatusBadRequest, api.Error{Error: err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, api.Error{Error: "failed to create task"})
		return
	}

	apiTask := api.Task{
		Id:        int32(boTask.ID),
		Title:     *boTask.Title,
		Completed: *boTask.Completed,
		Assignee:  *boTask.Assignee,
	}

	c.JSON(http.StatusCreated, apiTask)
}

func (h *TaskHandler) ListTasks(c *gin.Context, params api.ListTasksParams) {
	filter := dao.TaskFilter{Assignee: params.Assignee, Completed: params.Completed}
	boTasks, err := h.service.List(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Error: "failed to list tasks"})
		return
	}

	h.taskCount.Add(c.Request.Context(), int64(len(boTasks)))

	apiTasks := make([]api.Task, 0, len(boTasks))
	for _, task := range boTasks {
		apiTasks = append(apiTasks, api.Task{
			Id:        int32(task.ID),
			Title:     *task.Title,
			Completed: *task.Completed,
			Assignee:  *task.Assignee,
		})
	}

	c.JSON(http.StatusOK, apiTasks)
}

func (h *TaskHandler) GetTask(c *gin.Context, id int) {
	if id < 1 {
		c.JSON(http.StatusBadRequest, api.Error{Error: "invalid task ID"})
		return
	}

	boTask, err := h.service.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, api.Error{Error: "task not found"})
		return
	}

	apiTask := api.Task{
		Id:        int32(boTask.ID),
		Title:     *boTask.Title,
		Completed: *boTask.Completed,
		Assignee:  *boTask.Assignee,
	}

	c.JSON(http.StatusOK, apiTask)
}

func (h *TaskHandler) UpdateTask(c *gin.Context, id int) {
	if id < 1 {
		c.JSON(http.StatusBadRequest, api.Error{Error: "invalid task ID"})
		return
	}

	var apiUpdateTaskRequest api.UpdateTaskRequest
	if err := c.ShouldBindJSON(&apiUpdateTaskRequest); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Error: err.Error()})
		return
	}

	if apiUpdateTaskRequest.Title == nil && apiUpdateTaskRequest.Completed == nil && apiUpdateTaskRequest.Assignee == nil {
		c.JSON(http.StatusBadRequest, api.Error{Error: "no fields to update"})
		return
	}

	boTask, err := h.service.Update(bo.Task{
		ID:        id,
		Title:     apiUpdateTaskRequest.Title,
		Completed: apiUpdateTaskRequest.Completed,
		Assignee:  apiUpdateTaskRequest.Assignee,
	})
	if err != nil {
		if err.Error() == "title cannot be empty" {
			c.JSON(http.StatusBadRequest, api.Error{Error: err.Error()})
			return
		}
		if err.Error() == "task not found" {
			c.JSON(http.StatusNotFound, api.Error{Error: "task not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, api.Error{Error: "failed to update task"})
		return
	}

	aptTask := api.Task{
		Id:        int32(boTask.ID),
		Title:     *boTask.Title,
		Completed: *boTask.Completed,
		Assignee:  *boTask.Assignee,
	}

	c.JSON(http.StatusOK, aptTask)
}

func (h *TaskHandler) DeleteTask(c *gin.Context, id int) {
	if id < 1 {
		c.JSON(http.StatusBadRequest, api.Error{Error: "invalid task ID"})
		return
	}

	if err := h.service.Delete(id); err != nil {
		c.JSON(http.StatusNotFound, api.Error{Error: "task not found"})
		return
	}

	c.Status(http.StatusNoContent)
}
