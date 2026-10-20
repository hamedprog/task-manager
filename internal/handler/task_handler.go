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
	Create(boTask bo.Task) (*bo.Task, error)
	List(daoTaskFilter dao.TaskFilter, daoCursor dao.Cursor) ([]bo.Task, error)
	GetByID(id int) (*bo.Task, error)
	Update(boTask bo.Task) (*bo.Task, error)
	Delete(id int) error
}

type TaskHandler struct {
	taskService TaskService
	taskCount   metric.Int64Counter
}

func NewTaskHandler(TaskService TaskService, taskCount metric.Int64Counter) *TaskHandler {
	return &TaskHandler{taskService: TaskService, taskCount: taskCount}
}

func (h *TaskHandler) RegisterRoutes(router gin.IRouter) {
	api.RegisterHandlersWithOptions(router, h, api.GinServerOptions{
		ErrorHandler: func(ginContext *gin.Context, err error, statusCode int) {
			message := "invalid task filter"
			if strings.Contains(err.Error(), "parameter id") {
				message = "invalid task ID"
			}
			ginContext.JSON(statusCode, api.Error{Error: message})
		},
	})
}

func (h *TaskHandler) CreateTask(ginContext *gin.Context) {
	var apiCreateTaskRequest api.CreateTaskRequest
	if err := ginContext.ShouldBindJSON(&apiCreateTaskRequest); err != nil {
		ginContext.JSON(http.StatusBadRequest, api.Error{Error: err.Error()})
		return
	}

	assignee := ""
	if apiCreateTaskRequest.Assignee != nil {
		assignee = *apiCreateTaskRequest.Assignee
	}

	boTask, err := h.taskService.Create(bo.Task{Title: &apiCreateTaskRequest.Title, Assignee: &assignee})
	if err != nil {
		if err.Error() == "title cannot be empty" {
			ginContext.JSON(http.StatusBadRequest, api.Error{Error: err.Error()})
			return
		}
		ginContext.JSON(http.StatusInternalServerError, api.Error{Error: "failed to create task"})
		return
	}

	apiTask := api.Task{
		Id:        int32(boTask.ID),
		Title:     *boTask.Title,
		Completed: *boTask.Completed,
		Assignee:  *boTask.Assignee,
	}

	ginContext.JSON(http.StatusCreated, apiTask)
}

func (h *TaskHandler) ListTasks(ginContext *gin.Context, apiListTasksParams api.ListTasksParams) {
	if apiListTasksParams.Offset != nil && *apiListTasksParams.Offset < 0 {
		ginContext.JSON(http.StatusBadRequest, api.Error{Error: "offset must be non-negative"})
		return
	}
	if apiListTasksParams.Limit != nil && *apiListTasksParams.Limit < 1 {
		ginContext.JSON(http.StatusBadRequest, api.Error{Error: "limit must be positive"})
		return
	}

	daoTaskFilter := dao.TaskFilter{Assignee: apiListTasksParams.Assignee, Completed: apiListTasksParams.Completed}
	daoCursor := dao.Cursor{Offset: apiListTasksParams.Offset, Limit: apiListTasksParams.Limit}
	boTasks, err := h.taskService.List(daoTaskFilter, daoCursor)
	if err != nil {
		ginContext.JSON(http.StatusInternalServerError, api.Error{Error: "failed to list tasks"})
		return
	}

	h.taskCount.Add(ginContext.Request.Context(), int64(len(boTasks)))

	apiTasks := make([]api.Task, 0, len(boTasks))
	for _, task := range boTasks {
		apiTasks = append(apiTasks, api.Task{
			Id:        int32(task.ID),
			Title:     *task.Title,
			Completed: *task.Completed,
			Assignee:  *task.Assignee,
		})
	}

	ginContext.JSON(http.StatusOK, apiTasks)
}

func (h *TaskHandler) GetTask(ginContext *gin.Context, id int) {
	if id < 1 {
		ginContext.JSON(http.StatusBadRequest, api.Error{Error: "invalid task ID"})
		return
	}

	boTask, err := h.taskService.GetByID(id)
	if err != nil {
		ginContext.JSON(http.StatusNotFound, api.Error{Error: "task not found"})
		return
	}

	apiTask := api.Task{
		Id:        int32(boTask.ID),
		Title:     *boTask.Title,
		Completed: *boTask.Completed,
		Assignee:  *boTask.Assignee,
	}

	ginContext.JSON(http.StatusOK, apiTask)
}

func (h *TaskHandler) UpdateTask(ginContext *gin.Context, id int) {
	if id < 1 {
		ginContext.JSON(http.StatusBadRequest, api.Error{Error: "invalid task ID"})
		return
	}

	var apiUpdateTaskRequest api.UpdateTaskRequest
	if err := ginContext.ShouldBindJSON(&apiUpdateTaskRequest); err != nil {
		ginContext.JSON(http.StatusBadRequest, api.Error{Error: err.Error()})
		return
	}

	if apiUpdateTaskRequest.Title == nil && apiUpdateTaskRequest.Completed == nil && apiUpdateTaskRequest.Assignee == nil {
		ginContext.JSON(http.StatusBadRequest, api.Error{Error: "no fields to update"})
		return
	}

	boTask, err := h.taskService.Update(bo.Task{
		ID:        id,
		Title:     apiUpdateTaskRequest.Title,
		Completed: apiUpdateTaskRequest.Completed,
		Assignee:  apiUpdateTaskRequest.Assignee,
	})
	if err != nil {
		if err.Error() == "title cannot be empty" {
			ginContext.JSON(http.StatusBadRequest, api.Error{Error: err.Error()})
			return
		}
		if err.Error() == "task not found" {
			ginContext.JSON(http.StatusNotFound, api.Error{Error: "task not found"})
			return
		}
		ginContext.JSON(http.StatusInternalServerError, api.Error{Error: "failed to update task"})
		return
	}

	aptTask := api.Task{
		Id:        int32(boTask.ID),
		Title:     *boTask.Title,
		Completed: *boTask.Completed,
		Assignee:  *boTask.Assignee,
	}

	ginContext.JSON(http.StatusOK, aptTask)
}

func (h *TaskHandler) DeleteTask(ginContext *gin.Context, id int) {
	if id < 1 {
		ginContext.JSON(http.StatusBadRequest, api.Error{Error: "invalid task ID"})
		return
	}

	if err := h.taskService.Delete(id); err != nil {
		ginContext.JSON(http.StatusNotFound, api.Error{Error: "task not found"})
		return
	}

	ginContext.Status(http.StatusNoContent)
}
