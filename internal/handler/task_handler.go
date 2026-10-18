package handler

import (
	"net/http"
	"strconv"

	"task-manager/internal/object/bo"
	"task-manager/internal/object/dto"

	"github.com/gin-gonic/gin"
)

type TaskService interface {
	Create(task bo.Task) (*bo.Task, error)
	List() ([]bo.Task, error)
	GetByID(id int) (*bo.Task, error)
	Update(task bo.Task) (*bo.Task, error)
	Delete(id int) error
}

type TaskHandler struct {
	service TaskService
}

func NewTaskHandler(service TaskService) *TaskHandler {
	return &TaskHandler{
		service: service,
	}
}

func (h *TaskHandler) RegisterRoutes(router *gin.Engine) {
	routerGroup := router.Group("/tasks")

	routerGroup.POST("", h.Create)
	routerGroup.GET("", h.List)
	routerGroup.GET("/:id", h.GetByID)
	routerGroup.PATCH("/:id", h.Update)
	routerGroup.DELETE("/:id", h.Delete)
}

func (h *TaskHandler) Create(c *gin.Context) {
	var req dto.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	boTask, err := h.service.Create(bo.Task{Title: &req.Title})
	if err != nil {
		if err.Error() == "title cannot be empty" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create task"})
		return
	}

	dtoTaskResponse := dto.TaskResponse{
		ID:        boTask.ID,
		Title:     *boTask.Title,
		Completed: *boTask.Completed,
	}

	c.JSON(http.StatusCreated, dtoTaskResponse)
}

func (h *TaskHandler) List(c *gin.Context) {
	boTasks, err := h.service.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tasks"})
		return
	}

	dtoTaskResponses := make([]dto.TaskResponse, 0, len(boTasks))
	for _, task := range boTasks {
		dtoTaskResponses = append(dtoTaskResponses, dto.TaskResponse{
			ID:        task.ID,
			Title:     *task.Title,
			Completed: *task.Completed,
		})
	}

	c.JSON(http.StatusOK, dtoTaskResponses)
}

func (h *TaskHandler) GetByID(c *gin.Context) {
	id, ok := parseTaskID(c)
	if !ok {
		return
	}

	boTask, err := h.service.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	dtoTaskResponse := dto.TaskResponse{
		ID:        boTask.ID,
		Title:     *boTask.Title,
		Completed: *boTask.Completed,
	}

	c.JSON(http.StatusOK, dtoTaskResponse)
}

func (h *TaskHandler) Update(c *gin.Context) {
	id, ok := parseTaskID(c)
	if !ok {
		return
	}

	var dtoUpdateTaskRequest dto.UpdateTaskRequest
	if err := c.ShouldBindJSON(&dtoUpdateTaskRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if dtoUpdateTaskRequest.Title == nil && dtoUpdateTaskRequest.Completed == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}

	boTask, err := h.service.Update(bo.Task{
		ID:        id,
		Title:     dtoUpdateTaskRequest.Title,
		Completed: dtoUpdateTaskRequest.Completed,
	})
	if err != nil {
		if err.Error() == "title cannot be empty" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "task not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update task"})
		return
	}

	dtoTaskResponse := dto.TaskResponse{
		ID:        boTask.ID,
		Title:     *boTask.Title,
		Completed: *boTask.Completed,
	}

	c.JSON(http.StatusOK, dtoTaskResponse)
}

func (h *TaskHandler) Delete(c *gin.Context) {
	id, ok := parseTaskID(c)
	if !ok {
		return
	}

	if err := h.service.Delete(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

func parseTaskID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return 0, false
	}

	return id, true
}
