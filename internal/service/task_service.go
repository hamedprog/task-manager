package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"task-manager/internal/object/bo"
	"task-manager/internal/object/dao"
)

var ErrTaskNotFound = errors.New("task not found")

type TaskRepository interface {
	Create(daoTask *dao.Task) error
	List(daoTaskFilter dao.TaskFilter, daoCursor dao.Cursor) ([]dao.Task, error)
	GetByID(id int) (*dao.Task, error)
	Update(daoTask *dao.Task) error
	Delete(id int) error
}

type TaskListCache interface {
	Version(ctx context.Context) (int64, error)
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Invalidate(ctx context.Context) error
}

type TaskService struct {
	taskRepository TaskRepository
	taskListCache  TaskListCache
}

func NewTaskService(taskRepository TaskRepository, taskListCache TaskListCache) *TaskService {
	return &TaskService{taskRepository: taskRepository, taskListCache: taskListCache}
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
	if err := s.taskRepository.Create(daoTask); err != nil {
		return nil, err
	}
	s.invalidateTaskListCache()

	return toBusinessTask(daoTask), nil
}

func (s *TaskService) List(daoTaskFilter dao.TaskFilter, daoCursor dao.Cursor) ([]bo.Task, error) {
	ctx := context.Background()
	cacheKey, cacheReady := s.taskListCacheKey(ctx, daoTaskFilter, daoCursor)
	if cacheReady {
		cachedTasks, found, err := s.taskListCache.Get(ctx, cacheKey)
		if err != nil {
			log.Printf("read task list cache: %v", err)
		} else if found {
			var tasks []bo.Task
			if err := json.Unmarshal(cachedTasks, &tasks); err == nil {
				return tasks, nil
			}
			log.Printf("decode cached task list: invalid cached JSON")
		}
	}

	daoTasks, err := s.taskRepository.List(daoTaskFilter, daoCursor)
	if err != nil {
		return nil, err
	}

	boTasks := make([]bo.Task, 0, len(daoTasks))
	for i := range daoTasks {
		boTasks = append(boTasks, *toBusinessTask(&daoTasks[i]))
	}
	if cacheReady {
		cachedTasks, err := json.Marshal(boTasks)
		if err != nil {
			log.Printf("encode task list cache: %v", err)
		} else if err := s.taskListCache.Set(ctx, cacheKey, cachedTasks, 30*time.Second); err != nil {
			log.Printf("write task list cache: %v", err)
		}
	}

	return boTasks, nil
}

func (s *TaskService) taskListCacheKey(ctx context.Context, daoTaskFilter dao.TaskFilter, daoCursor dao.Cursor) (string, bool) {
	if s.taskListCache == nil {
		return "", false
	}

	version, err := s.taskListCache.Version(ctx)
	if err != nil {
		log.Printf("read task list cache version: %v", err)
		return "", false
	}

	request, err := json.Marshal(struct {
		daoTaskFilter dao.TaskFilter
		daoCursor     dao.Cursor
	}{daoTaskFilter: daoTaskFilter, daoCursor: daoCursor})
	if err != nil {
		log.Printf("encode task list cache key: %v", err)
		return "", false
	}
	hash := sha256.Sum256(request)
	return fmt.Sprintf("tasks:list:v%d:%s", version, hex.EncodeToString(hash[:])), true
}

func (s *TaskService) invalidateTaskListCache() {
	if s.taskListCache == nil {
		return
	}
	if err := s.taskListCache.Invalidate(context.Background()); err != nil {
		log.Printf("invalidate task list cache: %v", err)
	}
}

func (s *TaskService) GetByID(id int) (*bo.Task, error) {
	daoTask, err := s.taskRepository.GetByID(id)
	if err != nil {
		return nil, err
	}

	return toBusinessTask(daoTask), nil
}

func (s *TaskService) Update(boTask bo.Task) (*bo.Task, error) {
	daoTask, err := s.taskRepository.GetByID(boTask.ID)
	if err != nil {
		return nil, ErrTaskNotFound
	}

	if boTask.Title != nil {
		trimmedTitle := strings.TrimSpace(*boTask.Title)
		if trimmedTitle == "" {
			return nil, errors.New("title cannot be empty")
		}
		daoTask.Title = trimmedTitle
	}

	if boTask.Completed != nil {
		daoTask.Completed = *boTask.Completed
	}

	if boTask.Assignee != nil {
		daoTask.Assignee = *boTask.Assignee
	}

	if err := s.taskRepository.Update(daoTask); err != nil {
		return nil, err
	}
	s.invalidateTaskListCache()

	return toBusinessTask(daoTask), nil
}

func (s *TaskService) Delete(id int) error {
	if err := s.taskRepository.Delete(id); err != nil {
		return err
	}
	s.invalidateTaskListCache()
	return nil
}

func toBusinessTask(daotask *dao.Task) *bo.Task {
	title := daotask.Title
	completed := daotask.Completed
	assignee := daotask.Assignee

	return &bo.Task{
		ID:        daotask.ID,
		Title:     &title,
		Completed: &completed,
		Assignee:  &assignee,
	}
}
