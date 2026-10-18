package handler

import (
	"net/http"
	"strconv"

	"task-manager/internal/dto"
	"task-manager/internal/model"

	"github.com/gin-gonic/gin"
)

type TaskRepository interface {
	Create(task *model.Task) error
	List() ([]model.Task, error)
	GetByID(id int) (*model.Task, error)
	Update(task *model.Task) error
	Delete(id int) error
}

type TaskHandler struct {
	repository TaskRepository
}

func NewTaskHandler(repository TaskRepository) *TaskHandler {
	return &TaskHandler{
		repository: repository,
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

	modelTask := model.Task{
		Title: req.Title,
	}

	if err := h.repository.Create(&modelTask); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create task"})
		return
	}

	dtoTaskResponse := dto.TaskResponse{
		ID:        modelTask.ID,
		Title:     modelTask.Title,
		Completed: modelTask.Completed,
	}

	c.JSON(http.StatusCreated, dtoTaskResponse)
}

func (h *TaskHandler) List(c *gin.Context) {
	modelTasks, err := h.repository.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tasks"})
		return
	}

	dtoTaskResponses := make([]dto.TaskResponse, 0, len(modelTasks))
	for _, task := range modelTasks {
		dtoTaskResponses = append(dtoTaskResponses, dto.TaskResponse{
			ID:        task.ID,
			Title:     task.Title,
			Completed: task.Completed,
		})
	}

	c.JSON(http.StatusOK, dtoTaskResponses)
}

func (h *TaskHandler) GetByID(c *gin.Context) {
	id, ok := parseTaskID(c)
	if !ok {
		return
	}

	modelTask, err := h.repository.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	dtoTaskResponse := dto.TaskResponse{
		ID:        modelTask.ID,
		Title:     modelTask.Title,
		Completed: modelTask.Completed,
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

	if dtoUpdateTaskRequest.Title != nil && *dtoUpdateTaskRequest.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title cannot be empty"})
		return
	}

	modelTask, err := h.repository.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	if dtoUpdateTaskRequest.Title != nil {
		modelTask.Title = *dtoUpdateTaskRequest.Title
	}
	if dtoUpdateTaskRequest.Completed != nil {
		modelTask.Completed = *dtoUpdateTaskRequest.Completed
	}

	if err := h.repository.Update(modelTask); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update task"})
		return
	}

	dtoTaskResponse := dto.TaskResponse{
		ID:        modelTask.ID,
		Title:     modelTask.Title,
		Completed: modelTask.Completed,
	}

	c.JSON(http.StatusOK, dtoTaskResponse)
}

func (h *TaskHandler) Delete(c *gin.Context) {
	id, ok := parseTaskID(c)
	if !ok {
		return
	}

	if err := h.repository.Delete(id); err != nil {
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
