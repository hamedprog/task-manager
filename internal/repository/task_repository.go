package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"task-manager/internal/object/dao"
)

var ErrTaskNotFound = errors.New("task not found")

type TaskRepository struct {
	sqlDB *sql.DB
}

func NewTaskRepository(sqlDB *sql.DB) *TaskRepository {
	return &TaskRepository{sqlDB: sqlDB}
}

func (r *TaskRepository) Create(daoTask *dao.Task) error {
	const query = `INSERT INTO tasks (title, completed, assignee) VALUES ($1, $2, $3) RETURNING id`
	if err := r.sqlDB.QueryRow(query, daoTask.Title, daoTask.Completed, daoTask.Assignee).Scan(&daoTask.ID); err != nil {
		return fmt.Errorf("create task: %w", err)
	}

	return nil
}

func (r *TaskRepository) GetByID(id int) (*dao.Task, error) {
	const query = `SELECT id, title, completed, assignee FROM tasks WHERE id = $1`
	daoTask := &dao.Task{}
	if err := r.sqlDB.QueryRow(query, id).Scan(&daoTask.ID, &daoTask.Title, &daoTask.Completed, &daoTask.Assignee); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("get task by ID: %w", err)
	}

	return daoTask, nil
}

func (r *TaskRepository) List(daoTaskFilter dao.TaskFilter, daoCursor dao.Cursor) ([]dao.Task, error) {
	query := `SELECT id, title, completed, assignee FROM tasks`
	var conditions []string
	var args []any
	if daoTaskFilter.Assignee != nil {
		args = append(args, *daoTaskFilter.Assignee)
		conditions = append(conditions, fmt.Sprintf("assignee = $%d", len(args)))
	}
	if daoTaskFilter.Completed != nil {
		args = append(args, *daoTaskFilter.Completed)
		conditions = append(conditions, fmt.Sprintf("completed = $%d", len(args)))
	}
	if len(conditions) != 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY id"
	if daoCursor.Limit != nil {
		args = append(args, *daoCursor.Limit)
		query += fmt.Sprintf(" LIMIT $%d", len(args))
	}
	if daoCursor.Offset != nil {
		args = append(args, *daoCursor.Offset)
		query += fmt.Sprintf(" OFFSET $%d", len(args))
	}

	rows, err := r.sqlDB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	daoTasks := make([]dao.Task, 0)
	for rows.Next() {
		var task dao.Task
		if err := rows.Scan(&task.ID, &task.Title, &task.Completed, &task.Assignee); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		daoTasks = append(daoTasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks: %w", err)
	}

	return daoTasks, nil
}

func (r *TaskRepository) Update(daoTask *dao.Task) error {
	const query = `UPDATE tasks SET title = $1, completed = $2, assignee = $3 WHERE id = $4`
	sqlResult, err := r.sqlDB.Exec(query, daoTask.Title, daoTask.Completed, daoTask.Assignee, daoTask.ID)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	rowsAffected, err := sqlResult.RowsAffected()
	if err != nil {
		return fmt.Errorf("get updated task count: %w", err)
	}
	if rowsAffected == 0 {
		return ErrTaskNotFound
	}

	return nil
}

func (r *TaskRepository) Delete(id int) error {
	const query = `DELETE FROM tasks WHERE id = $1`
	sqlResult, err := r.sqlDB.Exec(query, id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	rowsAffected, err := sqlResult.RowsAffected()
	if err != nil {
		return fmt.Errorf("get deleted task count: %w", err)
	}
	if rowsAffected == 0 {
		return ErrTaskNotFound
	}

	return nil
}
