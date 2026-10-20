package service

import (
	"errors"
	"strings"

	"task-manager/internal/object/bo"
	"task-manager/internal/object/dao"
)

var ErrTaskNotFound = errors.New("task not found")

type TaskRepository interface {
	Create(task *dao.Task) error
	List(filter dao.TaskFilter) ([]dao.Task, error)
	GetByID(id int) (*dao.Task, error)
	Update(task *dao.Task) error
	Delete(id int) error
}

type TaskService struct {
	repository TaskRepository
}

func NewTaskService(repository TaskRepository) *TaskService {
	return &TaskService{repository: repository}
}

func (s *TaskService) Create(boTask bo.Task) (*bo.Task, error) {
	title := ""
	if boTask.Title != nil {
		title = strings.TrimSpace(*boTask.Title)
	}
	if title == "" {
		return nil, errors.New("title cannot be empty")
	}

	assignee := ""
	if boTask.Assignee != nil {
		assignee = *boTask.Assignee
	}
	daoTask := &dao.Task{Title: title, Assignee: assignee}
	if err := s.repository.Create(daoTask); err != nil {
		return nil, err
	}

	return toBusinessTask(daoTask), nil
}

func (s *TaskService) List(filter dao.TaskFilter) ([]bo.Task, error) {
	daoTasks, err := s.repository.List(filter)
	if err != nil {
		return nil, err
	}

	tasks := make([]bo.Task, 0, len(daoTasks))
	for i := range daoTasks {
		tasks = append(tasks, *toBusinessTask(&daoTasks[i]))
	}

	return tasks, nil
}

func (s *TaskService) GetByID(id int) (*bo.Task, error) {
	daoTask, err := s.repository.GetByID(id)
	if err != nil {
		return nil, err
	}

	return toBusinessTask(daoTask), nil
}

func (s *TaskService) Update(task bo.Task) (*bo.Task, error) {
	daoTask, err := s.repository.GetByID(task.ID)
	if err != nil {
		return nil, ErrTaskNotFound
	}

	if task.Title != nil {
		trimmedTitle := strings.TrimSpace(*task.Title)
		if trimmedTitle == "" {
			return nil, errors.New("title cannot be empty")
		}
		daoTask.Title = trimmedTitle
	}

	if task.Completed != nil {
		daoTask.Completed = *task.Completed
	}

	if task.Assignee != nil {
		daoTask.Assignee = *task.Assignee
	}

	if err := s.repository.Update(daoTask); err != nil {
		return nil, err
	}

	return toBusinessTask(daoTask), nil
}

func (s *TaskService) Delete(id int) error {
	return s.repository.Delete(id)
}

func toBusinessTask(task *dao.Task) *bo.Task {
	title := task.Title
	completed := task.Completed
	assignee := task.Assignee

	return &bo.Task{
		ID:        task.ID,
		Title:     &title,
		Completed: &completed,
		Assignee:  &assignee,
	}
}
