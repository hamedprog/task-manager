package repository

import (
	"errors"
	"sync"
	"task-manager/internal/model"
)

var (
	ErrTaskNotFound = errors.New("task not found")
)

type TaskRepository interface {
	Create(task *model.Task) error
	GetByID(id int) (*model.Task, error)
	List() ([]model.Task, error)
	Update(task *model.Task) error
	Delete(id int) error
}

type taskRepository struct {
	mu     sync.RWMutex
	tasks  map[int]model.Task
	nextID int
}

func NewTaskRepository() TaskRepository {
	return &taskRepository{
		tasks:  make(map[int]model.Task),
		nextID: 1,
	}
}

func (r *taskRepository) Create(task *model.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	task.ID = r.nextID
	r.tasks[task.ID] = *task
	r.nextID++

	return nil
}

func (r *taskRepository) GetByID(id int) (*model.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	task, exists := r.tasks[id]
	if !exists {
		return nil, ErrTaskNotFound
	}

	return &task, nil
}

func (r *taskRepository) List() ([]model.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tasks := make([]model.Task, 0, len(r.tasks))
	for _, task := range r.tasks {
		tasks = append(tasks, task)
	}

	return tasks, nil
}

func (r *taskRepository) Update(task *model.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tasks[task.ID]; !exists {
		return ErrTaskNotFound
	}

	r.tasks[task.ID] = *task
	return nil
}

func (r *taskRepository) Delete(id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tasks[id]; !exists {
		return ErrTaskNotFound
	}

	delete(r.tasks, id)
	return nil
}
