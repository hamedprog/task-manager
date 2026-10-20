package repository

import (
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"task-manager/internal/object/dao"

	"github.com/DATA-DOG/go-sqlmock"
)

func newMockTaskRepository(t *testing.T) (*TaskRepository, sqlmock.Sqlmock) {
	t.Helper()

	db, sqlMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create SQL mock: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	return NewTaskRepository(db), sqlMock
}

func TestTaskRepositoryCreate(t *testing.T) {
	const query = `INSERT INTO tasks (title, completed, assignee) VALUES ($1, $2, $3) RETURNING id`

	t.Run("success assigns ID", func(t *testing.T) {
		taskRepository, sqlMock := newMockTaskRepository(t)
		daoTask := &dao.Task{Title: "Write tests", Completed: true, Assignee: "Alex"}
		sqlMock.ExpectQuery(regexp.QuoteMeta(query)).
			WithArgs(daoTask.Title, daoTask.Completed, daoTask.Assignee).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(42))

		if err := taskRepository.Create(daoTask); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if daoTask.ID != 42 {
			t.Errorf("Create() assigned ID %d, want 42", daoTask.ID)
		}
		if err := sqlMock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet SQL expectations: %v", err)
		}
	})

	t.Run("database error", func(t *testing.T) {
		taskRepository, sqlMock := newMockTaskRepository(t)
		daoTask := &dao.Task{Title: "Write tests"}
		dbErr := errors.New("insert failed")
		sqlMock.ExpectQuery(regexp.QuoteMeta(query)).
			WithArgs(daoTask.Title, daoTask.Completed, daoTask.Assignee).
			WillReturnError(dbErr)

		err := taskRepository.Create(daoTask)
		if !errors.Is(err, dbErr) {
			t.Errorf("Create() error = %v, want wrapped %v", err, dbErr)
		}
		if err := sqlMock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet SQL expectations: %v", err)
		}
	})
}

func TestTaskRepositoryGetByID(t *testing.T) {
	const query = `SELECT id, title, completed, assignee FROM tasks WHERE id = $1`

	t.Run("success", func(t *testing.T) {
		taskRepository, sqlMock := newMockTaskRepository(t)
		want := &dao.Task{ID: 12, Title: "Read docs", Completed: true, Assignee: "Alex"}
		sqlMock.ExpectQuery(regexp.QuoteMeta(query)).
			WithArgs(want.ID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "completed", "assignee"}).
				AddRow(want.ID, want.Title, want.Completed, want.Assignee))

		got, err := taskRepository.GetByID(want.ID)
		if err != nil {
			t.Fatalf("GetByID(%d) error = %v", want.ID, err)
		}
		if *got != *want {
			t.Errorf("GetByID(%d) = %+v, want %+v", want.ID, got, want)
		}
		if err := sqlMock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet SQL expectations: %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		taskRepository, sqlMock := newMockTaskRepository(t)
		sqlMock.ExpectQuery(regexp.QuoteMeta(query)).
			WithArgs(99).
			WillReturnError(sql.ErrNoRows)

		got, err := taskRepository.GetByID(99)
		if !errors.Is(err, ErrTaskNotFound) {
			t.Errorf("GetByID(99) error = %v, want %v", err, ErrTaskNotFound)
		}
		if got != nil {
			t.Errorf("GetByID(99) = %+v, want nil", got)
		}
		if err := sqlMock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet SQL expectations: %v", err)
		}
	})

	t.Run("database error", func(t *testing.T) {
		taskRepository, sqlMock := newMockTaskRepository(t)
		dbErr := errors.New("select failed")
		sqlMock.ExpectQuery(regexp.QuoteMeta(query)).
			WithArgs(12).
			WillReturnError(dbErr)

		_, err := taskRepository.GetByID(12)
		if !errors.Is(err, dbErr) {
			t.Errorf("GetByID(12) error = %v, want wrapped %v", err, dbErr)
		}
		if err := sqlMock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet SQL expectations: %v", err)
		}
	})
}

func TestTaskRepositoryList(t *testing.T) {
	const query = `SELECT id, title, completed, assignee FROM tasks ORDER BY id`
	filter := dao.TaskFilter{}
	cursor := dao.Cursor{}

	t.Run("success", func(t *testing.T) {
		taskRepository, sqlMock := newMockTaskRepository(t)
		sqlMock.ExpectQuery(regexp.QuoteMeta(query)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "completed", "assignee"}).
				AddRow(1, "First", false, "Alex").
				AddRow(2, "Second", true, "Sam"))

		got, err := taskRepository.List(filter, cursor)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		want := []dao.Task{
			{ID: 1, Title: "First", Assignee: "Alex"},
			{ID: 2, Title: "Second", Completed: true, Assignee: "Sam"},
		}
		if len(got) != len(want) {
			t.Fatalf("List() returned %d tasks, want %d", len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("List()[%d] = %+v, want %+v", i, got[i], want[i])
			}
		}
		if err := sqlMock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet SQL expectations: %v", err)
		}
	})

	t.Run("empty result is non-nil", func(t *testing.T) {
		repository, sqlMock := newMockTaskRepository(t)
		sqlMock.ExpectQuery(regexp.QuoteMeta(query)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "completed", "assignee"}))

		got, err := repository.List(filter, cursor)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if got == nil || len(got) != 0 {
			t.Errorf("List() = %#v, want a non-nil empty slice", got)
		}
		if err := sqlMock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet SQL expectations: %v", err)
		}
	})

	t.Run("query error", func(t *testing.T) {
		repository, sqlMock := newMockTaskRepository(t)
		dbErr := errors.New("query failed")
		sqlMock.ExpectQuery(regexp.QuoteMeta(query)).WillReturnError(dbErr)

		_, err := repository.List(filter, cursor)
		if !errors.Is(err, dbErr) {
			t.Errorf("List() error = %v, want wrapped %v", err, dbErr)
		}
		if err := sqlMock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet SQL expectations: %v", err)
		}
	})

	t.Run("scan error", func(t *testing.T) {
		repository, sqlMock := newMockTaskRepository(t)
		sqlMock.ExpectQuery(regexp.QuoteMeta(query)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "completed", "assignee"}).
				AddRow("not an integer", "Task", false, "Alex"))

		_, err := repository.List(filter, cursor)
		if err == nil {
			t.Fatal("List() error = nil, want scan error")
		}
		if err := sqlMock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet SQL expectations: %v", err)
		}
	})

	t.Run("iteration error", func(t *testing.T) {
		repository, sqlMock := newMockTaskRepository(t)
		dbErr := errors.New("row iteration failed")
		rows := sqlmock.NewRows([]string{"id", "title", "completed", "assignee"}).
			AddRow(1, "Task", false, "Alex").
			RowError(0, dbErr)
		sqlMock.ExpectQuery(regexp.QuoteMeta(query)).WillReturnRows(rows)

		_, err := repository.List(filter, cursor)
		if !errors.Is(err, dbErr) {
			t.Errorf("List() error = %v, want wrapped %v", err, dbErr)
		}
		if err := sqlMock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet SQL expectations: %v", err)
		}
	})

	t.Run("filters by assignee and completed", func(t *testing.T) {
		repository, sqlMock := newMockTaskRepository(t)
		assignee := "Alex"
		completed := false
		filter := dao.TaskFilter{Assignee: &assignee, Completed: &completed}
		query := `SELECT id, title, completed, assignee FROM tasks WHERE assignee = $1 AND completed = $2 ORDER BY id`
		sqlMock.ExpectQuery(regexp.QuoteMeta(query)).
			WithArgs(assignee, completed).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "completed", "assignee"}).
				AddRow(1, "First", false, assignee))

		got, err := repository.List(filter, cursor)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(got) != 1 || got[0].Assignee != assignee || got[0].Completed {
			t.Errorf("List() = %+v, want one incomplete task assigned to %s", got, assignee)
		}
		if err := sqlMock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet SQL expectations: %v", err)
		}
	})

	t.Run("applies limit and offset after filters", func(t *testing.T) {
		repository, sqlMock := newMockTaskRepository(t)
		assignee := "Alex"
		completed := false
		limit := 10
		offset := 20
		filter := dao.TaskFilter{Assignee: &assignee, Completed: &completed}
		cursor := dao.Cursor{Limit: &limit, Offset: &offset}
		query := `SELECT id, title, completed, assignee FROM tasks WHERE assignee = $1 AND completed = $2 ORDER BY id LIMIT $3 OFFSET $4`
		sqlMock.ExpectQuery(regexp.QuoteMeta(query)).
			WithArgs(assignee, completed, limit, offset).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "completed", "assignee"}))

		got, err := repository.List(filter, cursor)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(got) != 0 {
			t.Errorf("List() returned %d tasks, want 0", len(got))
		}
		if err := sqlMock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet SQL expectations: %v", err)
		}
	})
}

func TestTaskRepositoryUpdate(t *testing.T) {
	const query = `UPDATE tasks SET title = $1, completed = $2, assignee = $3 WHERE id = $4`

	t.Run("success", func(t *testing.T) {
		taskRepository, sqlMock := newMockTaskRepository(t)
		daoTask := &dao.Task{ID: 7, Title: "Updated task", Completed: true, Assignee: "Alex"}
		sqlMock.ExpectExec(regexp.QuoteMeta(query)).
			WithArgs(daoTask.Title, daoTask.Completed, daoTask.Assignee, daoTask.ID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		if err := taskRepository.Update(daoTask); err != nil {
			t.Fatalf("Update() error = %v", err)
		}
		if err := sqlMock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet SQL expectations: %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		taskRepository, sqlMock := newMockTaskRepository(t)
		daoTask := &dao.Task{ID: 7, Title: "Missing task"}
		sqlMock.ExpectExec(regexp.QuoteMeta(query)).
			WithArgs(daoTask.Title, daoTask.Completed, daoTask.Assignee, daoTask.ID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		if err := taskRepository.Update(daoTask); !errors.Is(err, ErrTaskNotFound) {
			t.Errorf("Update() error = %v, want %v", err, ErrTaskNotFound)
		}
		if err := sqlMock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet SQL expectations: %v", err)
		}
	})

	t.Run("database error", func(t *testing.T) {
		taskRepository, sqlMock := newMockTaskRepository(t)
		task := &dao.Task{ID: 7, Title: "Updated task"}
		dbErr := errors.New("update failed")
		sqlMock.ExpectExec(regexp.QuoteMeta(query)).
			WithArgs(task.Title, task.Completed, task.Assignee, task.ID).
			WillReturnError(dbErr)

		err := taskRepository.Update(task)
		if !errors.Is(err, dbErr) {
			t.Errorf("Update() error = %v, want wrapped %v", err, dbErr)
		}
		if err := sqlMock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet SQL expectations: %v", err)
		}
	})

	t.Run("rows affected error", func(t *testing.T) {
		taskRepository, sqlMock := newMockTaskRepository(t)
		task := &dao.Task{ID: 7, Title: "Updated task"}
		resultErr := errors.New("affected rows unavailable")
		sqlMock.ExpectExec(regexp.QuoteMeta(query)).
			WithArgs(task.Title, task.Completed, task.Assignee, task.ID).
			WillReturnResult(sqlmock.NewErrorResult(resultErr))

		err := taskRepository.Update(task)
		if !errors.Is(err, resultErr) {
			t.Errorf("Update() error = %v, want wrapped %v", err, resultErr)
		}
		if err := sqlMock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet SQL expectations: %v", err)
		}
	})
}

func TestTaskRepositoryDelete(t *testing.T) {
	const query = `DELETE FROM tasks WHERE id = $1`

	t.Run("success", func(t *testing.T) {
		taskRepository, sqlMock := newMockTaskRepository(t)
		sqlMock.ExpectExec(regexp.QuoteMeta(query)).
			WithArgs(7).
			WillReturnResult(sqlmock.NewResult(0, 1))

		if err := taskRepository.Delete(7); err != nil {
			t.Fatalf("Delete(7) error = %v", err)
		}
		if err := sqlMock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet SQL expectations: %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		taskRepository, sqlMock := newMockTaskRepository(t)
		sqlMock.ExpectExec(regexp.QuoteMeta(query)).
			WithArgs(7).
			WillReturnResult(sqlmock.NewResult(0, 0))

		if err := taskRepository.Delete(7); !errors.Is(err, ErrTaskNotFound) {
			t.Errorf("Delete(7) error = %v, want %v", err, ErrTaskNotFound)
		}
		if err := sqlMock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet SQL expectations: %v", err)
		}
	})

	t.Run("database error", func(t *testing.T) {
		taskRepository, sqlMock := newMockTaskRepository(t)
		dbErr := errors.New("delete failed")
		sqlMock.ExpectExec(regexp.QuoteMeta(query)).
			WithArgs(7).
			WillReturnError(dbErr)

		err := taskRepository.Delete(7)
		if !errors.Is(err, dbErr) {
			t.Errorf("Delete(7) error = %v, want wrapped %v", err, dbErr)
		}
		if err := sqlMock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet SQL expectations: %v", err)
		}
	})

	t.Run("rows affected error", func(t *testing.T) {
		taskRepository, sqlMock := newMockTaskRepository(t)
		resultErr := errors.New("affected rows unavailable")
		sqlMock.ExpectExec(regexp.QuoteMeta(query)).
			WithArgs(7).
			WillReturnResult(sqlmock.NewErrorResult(resultErr))

		err := taskRepository.Delete(7)
		if !errors.Is(err, resultErr) {
			t.Errorf("Delete(7) error = %v, want wrapped %v", err, resultErr)
		}
		if err := sqlMock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet SQL expectations: %v", err)
		}
	})
}
