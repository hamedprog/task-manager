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
	Create(task *dao.Task) error
	List(filter dao.TaskFilter, cursor dao.Cursor) ([]dao.Task, error)
	GetByID(id int) (*dao.Task, error)
	Update(task *dao.Task) error
	Delete(id int) error
}

type TaskListCache interface {
	Version(ctx context.Context) (int64, error)
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Invalidate(ctx context.Context) error
}

type TaskService struct {
	repository TaskRepository
	cache      TaskListCache
}

func NewTaskService(repository TaskRepository, cache TaskListCache) *TaskService {
	return &TaskService{repository: repository, cache: cache}
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
	s.invalidateTaskListCache()

	return toBusinessTask(daoTask), nil
}

func (s *TaskService) List(filter dao.TaskFilter, cursor dao.Cursor) ([]bo.Task, error) {
	ctx := context.Background()
	cacheKey, cacheReady := s.taskListCacheKey(ctx, filter, cursor)
	if cacheReady {
		cachedTasks, found, err := s.cache.Get(ctx, cacheKey)
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

	daoTasks, err := s.repository.List(filter, cursor)
	if err != nil {
		return nil, err
	}

	tasks := make([]bo.Task, 0, len(daoTasks))
	for i := range daoTasks {
		tasks = append(tasks, *toBusinessTask(&daoTasks[i]))
	}
	if cacheReady {
		cachedTasks, err := json.Marshal(tasks)
		if err != nil {
			log.Printf("encode task list cache: %v", err)
		} else if err := s.cache.Set(ctx, cacheKey, cachedTasks, 30*time.Second); err != nil {
			log.Printf("write task list cache: %v", err)
		}
	}

	return tasks, nil
}

func (s *TaskService) taskListCacheKey(ctx context.Context, filter dao.TaskFilter, cursor dao.Cursor) (string, bool) {
	if s.cache == nil {
		return "", false
	}

	version, err := s.cache.Version(ctx)
	if err != nil {
		log.Printf("read task list cache version: %v", err)
		return "", false
	}

	request, err := json.Marshal(struct {
		Filter dao.TaskFilter
		Cursor dao.Cursor
	}{Filter: filter, Cursor: cursor})
	if err != nil {
		log.Printf("encode task list cache key: %v", err)
		return "", false
	}
	hash := sha256.Sum256(request)
	return fmt.Sprintf("tasks:list:v%d:%s", version, hex.EncodeToString(hash[:])), true
}

func (s *TaskService) invalidateTaskListCache() {
	if s.cache == nil {
		return
	}
	if err := s.cache.Invalidate(context.Background()); err != nil {
		log.Printf("invalidate task list cache: %v", err)
	}
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
	s.invalidateTaskListCache()

	return toBusinessTask(daoTask), nil
}

func (s *TaskService) Delete(id int) error {
	if err := s.repository.Delete(id); err != nil {
		return err
	}
	s.invalidateTaskListCache()
	return nil
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
