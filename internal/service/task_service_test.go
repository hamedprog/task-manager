package service

import (
	"errors"
	"testing"

	"task-manager/internal/object/bo"
	"task-manager/internal/object/dao"

	"go.uber.org/mock/gomock"
)

func newMockTaskService(t *testing.T) (*MockTaskRepository, *TaskService) {
	t.Helper()

	repository := NewMockTaskRepository(gomock.NewController(t))
	return repository, NewTaskService(repository)
}

func stringPointer(value string) *string {
	return &value
}

func boolPointer(value bool) *bool {
	return &value
}

func TestTaskServiceCreate(t *testing.T) {
	t.Run("success trims title and maps task", func(t *testing.T) {
		repository, taskService := newMockTaskService(t)
		repository.EXPECT().
			Create(gomock.Any()).
			DoAndReturn(func(task *dao.Task) error {
				if task.Title != "Write tests" {
					t.Errorf("Create() received title %q, want %q", task.Title, "Write tests")
				}
				if task.Assignee != "Alex" {
					t.Errorf("Create() received assignee %q, want %q", task.Assignee, "Alex")
				}
				task.ID = 1
				return nil
			})

		got, err := taskService.Create(bo.Task{Title: stringPointer("  Write tests  "), Assignee: stringPointer("Alex")})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if got.ID != 1 || got.Title == nil || *got.Title != "Write tests" || got.Completed == nil || *got.Completed {
			t.Errorf("Create() = %+v, want task ID 1, title %q, incomplete", got, "Write tests")
		}
		if got.Assignee == nil || *got.Assignee != "Alex" {
			t.Errorf("Create() assignee = %v, want %q", got.Assignee, "Alex")
		}
	})

	t.Run("nil title is invalid", func(t *testing.T) {
		_, taskService := newMockTaskService(t)

		got, err := taskService.Create(bo.Task{})
		if err == nil || err.Error() != "title cannot be empty" {
			t.Errorf("Create() error = %v, want title cannot be empty", err)
		}
		if got != nil {
			t.Errorf("Create() = %+v, want nil", got)
		}
	})

	t.Run("blank title is invalid", func(t *testing.T) {
		_, taskService := newMockTaskService(t)

		got, err := taskService.Create(bo.Task{Title: stringPointer(" \t ")})
		if err == nil || err.Error() != "title cannot be empty" {
			t.Errorf("Create() error = %v, want title cannot be empty", err)
		}
		if got != nil {
			t.Errorf("Create() = %+v, want nil", got)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		repository, taskService := newMockTaskService(t)
		repositoryErr := errors.New("insert failed")
		repository.EXPECT().Create(gomock.Any()).Return(repositoryErr)

		got, err := taskService.Create(bo.Task{Title: stringPointer("Task")})
		if !errors.Is(err, repositoryErr) {
			t.Errorf("Create() error = %v, want %v", err, repositoryErr)
		}
		if got != nil {
			t.Errorf("Create() = %+v, want nil", got)
		}
	})
}

func TestTaskServiceList(t *testing.T) {
	t.Run("success maps all tasks", func(t *testing.T) {
		repository, taskService := newMockTaskService(t)
		repository.EXPECT().List(dao.TaskFilter{}).Return([]dao.Task{
			{ID: 1, Title: "First", Assignee: "Alex"},
			{ID: 2, Title: "Second", Completed: true, Assignee: "Sam"},
		}, nil)

		got, err := taskService.List(dao.TaskFilter{})
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("List() returned %d tasks, want 2", len(got))
		}
		if got[0].ID != 1 || got[0].Title == nil || *got[0].Title != "First" ||
			got[0].Completed == nil || *got[0].Completed {
			t.Errorf("List()[0] = %+v, want first incomplete task", got[0])
		}
		if got[0].Assignee == nil || *got[0].Assignee != "Alex" {
			t.Errorf("List()[0].Assignee = %v, want %q", got[0].Assignee, "Alex")
		}
		if got[1].ID != 2 || got[1].Title == nil || *got[1].Title != "Second" ||
			got[1].Completed == nil || !*got[1].Completed {
			t.Errorf("List()[1] = %+v, want second completed task", got[1])
		}
		if got[1].Assignee == nil || *got[1].Assignee != "Sam" {
			t.Errorf("List()[1].Assignee = %v, want %q", got[1].Assignee, "Sam")
		}
	})

	t.Run("passes filters to repository", func(t *testing.T) {
		repository, taskService := newMockTaskService(t)
		assignee := "Alex"
		completed := false
		filter := dao.TaskFilter{Assignee: &assignee, Completed: &completed}
		repository.EXPECT().List(filter).Return([]dao.Task{}, nil)

		got, err := taskService.List(filter)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(got) != 0 {
			t.Errorf("List() returned %d tasks, want 0", len(got))
		}
	})

	t.Run("repository error", func(t *testing.T) {
		repository, taskService := newMockTaskService(t)
		repositoryErr := errors.New("query failed")
		repository.EXPECT().List(dao.TaskFilter{}).Return(nil, repositoryErr)

		got, err := taskService.List(dao.TaskFilter{})
		if !errors.Is(err, repositoryErr) {
			t.Errorf("List() error = %v, want %v", err, repositoryErr)
		}
		if got != nil {
			t.Errorf("List() = %+v, want nil", got)
		}
	})
}

func TestTaskServiceGetByID(t *testing.T) {
	t.Run("success maps task", func(t *testing.T) {
		repository, taskService := newMockTaskService(t)
		repository.EXPECT().
			GetByID(5).
			Return(&dao.Task{ID: 5, Title: "Read docs", Completed: true, Assignee: "Alex"}, nil)

		got, err := taskService.GetByID(5)
		if err != nil {
			t.Fatalf("GetByID(5) error = %v", err)
		}
		if got.ID != 5 || got.Title == nil || *got.Title != "Read docs" ||
			got.Completed == nil || !*got.Completed {
			t.Errorf("GetByID(5) = %+v, want mapped task", got)
		}
		if got.Assignee == nil || *got.Assignee != "Alex" {
			t.Errorf("GetByID(5).Assignee = %v, want %q", got.Assignee, "Alex")
		}
	})

	t.Run("repository error", func(t *testing.T) {
		repository, taskService := newMockTaskService(t)
		repositoryErr := errors.New("query failed")
		repository.EXPECT().GetByID(5).Return(nil, repositoryErr)

		got, err := taskService.GetByID(5)
		if !errors.Is(err, repositoryErr) {
			t.Errorf("GetByID(5) error = %v, want %v", err, repositoryErr)
		}
		if got != nil {
			t.Errorf("GetByID(5) = %+v, want nil", got)
		}
	})
}

func TestTaskServiceUpdate(t *testing.T) {
	t.Run("success trims title and updates completed value", func(t *testing.T) {
		repository, taskService := newMockTaskService(t)
		daoTask := &dao.Task{ID: 5, Title: "Old title", Assignee: "Alex"}
		repository.EXPECT().GetByID(5).Return(daoTask, nil)
		repository.EXPECT().
			Update(&dao.Task{ID: 5, Title: "New title", Assignee: "Sam"}).
			Return(nil)

		got, err := taskService.Update(bo.Task{
			ID:        5,
			Title:     stringPointer("  New title "),
			Completed: boolPointer(false),
			Assignee:  stringPointer("Sam"),
		})
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}
		if got.ID != 5 || got.Title == nil || *got.Title != "New title" ||
			got.Completed == nil || *got.Completed {
			t.Errorf("Update() = %+v, want updated task", got)
		}
		if got.Assignee == nil || *got.Assignee != "Sam" {
			t.Errorf("Update().Assignee = %v, want %q", got.Assignee, "Sam")
		}
	})

	t.Run("omitted assignee is preserved", func(t *testing.T) {
		repository, taskService := newMockTaskService(t)
		daoTask := &dao.Task{ID: 5, Title: "Task", Assignee: "Alex"}
		repository.EXPECT().GetByID(5).Return(daoTask, nil)
		repository.EXPECT().Update(&dao.Task{ID: 5, Title: "Task", Completed: true, Assignee: "Alex"}).Return(nil)

		got, err := taskService.Update(bo.Task{ID: 5, Completed: boolPointer(true)})
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}
		if got.Assignee == nil || *got.Assignee != "Alex" {
			t.Errorf("Update().Assignee = %v, want preserved value %q", got.Assignee, "Alex")
		}
	})

	t.Run("updates completed to true", func(t *testing.T) {
		repository, taskService := newMockTaskService(t)
		daoTask := &dao.Task{ID: 5, Title: "Task"}
		repository.EXPECT().GetByID(5).Return(daoTask, nil)
		repository.EXPECT().
			Update(&dao.Task{ID: 5, Title: "Task", Completed: true}).
			Return(nil)

		_, err := taskService.Update(bo.Task{ID: 5, Completed: boolPointer(true)})
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repository, taskService := newMockTaskService(t)
		repository.EXPECT().
			GetByID(5).
			Return(nil, errors.New("database not found"))

		got, err := taskService.Update(bo.Task{ID: 5})
		if !errors.Is(err, ErrTaskNotFound) {
			t.Errorf("Update() error = %v, want %v", err, ErrTaskNotFound)
		}
		if got != nil {
			t.Errorf("Update() = %+v, want nil", got)
		}
	})

	t.Run("blank title is invalid", func(t *testing.T) {
		repository, taskService := newMockTaskService(t)
		repository.EXPECT().
			GetByID(5).
			Return(&dao.Task{ID: 5, Title: "Original"}, nil)

		got, err := taskService.Update(bo.Task{ID: 5, Title: stringPointer("  ")})
		if err == nil || err.Error() != "title cannot be empty" {
			t.Errorf("Update() error = %v, want title cannot be empty", err)
		}
		if got != nil {
			t.Errorf("Update() = %+v, want nil", got)
		}
	})

	t.Run("repository update error", func(t *testing.T) {
		repository, taskService := newMockTaskService(t)
		repositoryErr := errors.New("update failed")
		repository.EXPECT().
			GetByID(5).
			Return(&dao.Task{ID: 5, Title: "Original"}, nil)
		repository.EXPECT().
			Update(&dao.Task{ID: 5, Title: "Original", Completed: true}).
			Return(repositoryErr)

		got, err := taskService.Update(bo.Task{ID: 5, Completed: boolPointer(true)})
		if !errors.Is(err, repositoryErr) {
			t.Errorf("Update() error = %v, want %v", err, repositoryErr)
		}
		if got != nil {
			t.Errorf("Update() = %+v, want nil", got)
		}
	})
}

func TestTaskServiceDelete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repository, taskService := newMockTaskService(t)
		repository.EXPECT().Delete(8).Return(nil)

		if err := taskService.Delete(8); err != nil {
			t.Fatalf("Delete(8) error = %v", err)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		repository, taskService := newMockTaskService(t)
		repositoryErr := errors.New("delete failed")
		repository.EXPECT().Delete(8).Return(repositoryErr)

		err := taskService.Delete(8)
		if !errors.Is(err, repositoryErr) {
			t.Errorf("Delete(8) error = %v, want %v", err, repositoryErr)
		}
	})
}
