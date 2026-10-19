package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"task-manager/internal/object/dao"
)

var ErrTaskNotFound = errors.New("task not found")

type TaskRepository struct {
	db *sql.DB
}

func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Create(task *dao.Task) error {
	const query = `INSERT INTO tasks (title, completed, assignee) VALUES ($1, $2, $3) RETURNING id`
	if err := r.db.QueryRow(query, task.Title, task.Completed, task.Assignee).Scan(&task.ID); err != nil {
		return fmt.Errorf("create task: %w", err)
	}

	return nil
}

func (r *TaskRepository) GetByID(id int) (*dao.Task, error) {
	const query = `SELECT id, title, completed, assignee FROM tasks WHERE id = $1`
	task := &dao.Task{}
	if err := r.db.QueryRow(query, id).Scan(&task.ID, &task.Title, &task.Completed, &task.Assignee); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("get task by ID: %w", err)
	}

	return task, nil
}

func (r *TaskRepository) List() ([]dao.Task, error) {
	const query = `SELECT id, title, completed, assignee FROM tasks ORDER BY id`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]dao.Task, 0)
	for rows.Next() {
		var task dao.Task
		if err := rows.Scan(&task.ID, &task.Title, &task.Completed, &task.Assignee); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks: %w", err)
	}

	return tasks, nil
}

func (r *TaskRepository) Update(task *dao.Task) error {
	const query = `UPDATE tasks SET title = $1, completed = $2, assignee = $3 WHERE id = $4`
	result, err := r.db.Exec(query, task.Title, task.Completed, task.Assignee, task.ID)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
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
	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get deleted task count: %w", err)
	}
	if rowsAffected == 0 {
		return ErrTaskNotFound
	}

	return nil
}
