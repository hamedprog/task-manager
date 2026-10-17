package main

import (
	"net/http"
	"strconv"
	"task-manager/internal/dto"
	"task-manager/internal/model"
	"task-manager/internal/repository"

	"github.com/gin-gonic/gin"
)

func main() {
	repository := repository.NewTaskRepository()
	router := gin.Default()

	router.POST("/tasks", func(c *gin.Context) {
		var req dto.CreateTaskRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		modelTask := model.Task{Title: req.Title}
		repository.Create(&modelTask)
		c.JSON(http.StatusCreated, modelTask)
	})

	router.GET("/tasks", func(c *gin.Context) {
		modelTasks, err := repository.List()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "internal error"})
			return
		}
		c.JSON(http.StatusOK, modelTasks)
	})

	router.GET("/tasks/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil || id < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
			return
		}

		modelTask, err := repository.GetByID(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}

		dtoTaskResponse := dto.TaskResponse{ID: modelTask.ID, Title: modelTask.Title, Completed: modelTask.Completed}

		c.JSON(http.StatusOK, dtoTaskResponse)
	})

	router.PATCH("/tasks/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil || id < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
			return
		}

		var req dto.UpdateTaskRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if req.Title == nil && req.Completed == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
			return
		}

		if req.Title != nil && *req.Title == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "title cannot be empty"})
			return
		}

		modelTask := model.Task{
			Title:     *req.Title,
			ID:        id,
			Completed: *req.Completed,
		}

		err = repository.Update(&modelTask)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}

		c.JSON(http.StatusOK, modelTask)
	})

	router.DELETE("/tasks/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil || id < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
			return
		}

		err = repository.Delete(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}

		c.Status(http.StatusNoContent)
	})

	router.Run(":9090")
}
