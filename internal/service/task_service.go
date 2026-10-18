package service

import (
	"errors"
	"strings"
	"task-manager/internal/model"
)

var ErrTaskNotFound = errors.New("task not found")

type TaskRepository interface {
	Create(task *model.Task) error
	List() ([]model.Task, error)
	GetByID(id int) (*model.Task, error)
	Update(task *model.Task) error
	Delete(id int) error
}

type TaskService struct {
	repository TaskRepository
}

func NewTaskService(repository TaskRepository) *TaskService {
	return &TaskService{repository: repository}
}

func (s *TaskService) Create(title string) (*model.Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, errors.New("title cannot be empty")
	}

	modelTask := &model.Task{Title: title}
	if err := s.repository.Create(modelTask); err != nil {
		return nil, err
	}

	return modelTask, nil
}

func (s *TaskService) List() ([]model.Task, error) {
	return s.repository.List()
}

func (s *TaskService) GetByID(id int) (*model.Task, error) {
	return s.repository.GetByID(id)
}

func (s *TaskService) Update(id int, title *string, completed *bool) (*model.Task, error) {
	modelTask, err := s.repository.GetByID(id)
	if err != nil {
		return nil, ErrTaskNotFound
	}

	if title != nil {
		trimmedTitle := strings.TrimSpace(*title)
		if trimmedTitle == "" {
			return nil, errors.New("title cannot be empty")
		}
		modelTask.Title = trimmedTitle
	}

	if completed != nil {
		modelTask.Completed = *completed
	}

	if err := s.repository.Update(modelTask); err != nil {
		return nil, err
	}

	return modelTask, nil
}

func (s *TaskService) Delete(id int) error {
	return s.repository.Delete(id)
}
