package main

import (
	"net/http"
	"strconv"
	"sync"
	"task-manager/dto"
	"task-manager/model"

	"github.com/gin-gonic/gin"
)

type TaskStore struct {
	mu     sync.RWMutex
	tasks  map[int]model.Task
	nextID int
}

func NewTaskStore() *TaskStore {
	return &TaskStore{
		tasks:  make(map[int]model.Task),
		nextID: 1,
	}
}

func (s *TaskStore) Create(title string) model.Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	task := model.Task{
		ID:        s.nextID,
		Title:     title,
		Completed: false,
	}

	s.tasks[task.ID] = task
	s.nextID++

	return task
}

func (s *TaskStore) List() []model.Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]model.Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}

	return tasks
}

func (s *TaskStore) Get(id int) (model.Task, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, ok := s.tasks[id]
	return task, ok
}

func (s *TaskStore) Update(id int, req dto.UpdateTaskRequest) (model.Task, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[id]
	if !ok {
		return model.Task{}, false
	}

	if req.Title != nil {
		task.Title = *req.Title
	}
	if req.Completed != nil {
		task.Completed = *req.Completed
	}

	s.tasks[id] = task
	return task, true
}

func (s *TaskStore) Delete(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[id]; !ok {
		return false
	}

	delete(s.tasks, id)
	return true
}

func main() {
	store := NewTaskStore()
	router := gin.Default()

	router.POST("/tasks", func(c *gin.Context) {
		var req dto.CreateTaskRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		task := store.Create(req.Title)
		c.JSON(http.StatusCreated, task)
	})

	router.GET("/tasks", func(c *gin.Context) {
		c.JSON(http.StatusOK, store.List())
	})

	router.GET("/tasks/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil || id < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
			return
		}

		task, ok := store.Get(id)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}

		dtoTask := dto.TaskResponse{ID: task.ID, Title: task.Title, Completed: task.Completed}

		c.JSON(http.StatusOK, dtoTask)
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

		task, ok := store.Update(id, req)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}

		c.JSON(http.StatusOK, task)
	})

	router.DELETE("/tasks/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil || id < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
			return
		}

		if !store.Delete(id) {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}

		c.Status(http.StatusNoContent)
	})

	router.Run(":9090")
}
