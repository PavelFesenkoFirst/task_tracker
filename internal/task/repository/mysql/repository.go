package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/PavelFesenkoFirst/task_tracker/internal/task"
)

type Repository struct {
	db *sql.DB
}

var _ task.Repository = (*Repository)(nil)

func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, params task.CreateParams) (task.Task, error) {
	const query = `
		INSERT INTO tasks (title, description, status, priority, due_at)
		VALUES (?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		params.Title,
		params.Description,
		params.Status,
		params.Priority,
		asNullableTime(params.DueAt),
	)
	if err != nil {
		return task.Task{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return task.Task{}, err
	}

	return r.GetByID(ctx, uint64(id))
}

func (r *Repository) GetByID(ctx context.Context, id uint64) (task.Task, error) {
	const query = `
		SELECT id, title, description, status, priority, due_at, created_at, updated_at
		FROM tasks
		WHERE id = ?
	`

	row := r.db.QueryRowContext(ctx, query, id)
	foundTask, err := scanTask(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return task.Task{}, task.ErrTaskNotFound
		}
		return task.Task{}, err
	}

	return foundTask, nil
}

func (r *Repository) List(ctx context.Context, filter task.ListFilter) ([]task.Task, error) {
	var queryBuilder strings.Builder
	queryBuilder.WriteString(`
		SELECT id, title, description, status, priority, due_at, created_at, updated_at
		FROM tasks
	`)

	args := make([]any, 0, 6)
	conditions := make([]string, 0, 2)

	if filter.Status != nil {
		conditions = append(conditions, "status = ?")
		args = append(args, *filter.Status)
	}

	if filter.Query != "" {
		conditions = append(conditions, "(title LIKE ? OR description LIKE ?)")
		likeExpr := "%" + filter.Query + "%"
		args = append(args, likeExpr, likeExpr)
	}

	if len(conditions) > 0 {
		queryBuilder.WriteString(" WHERE ")
		queryBuilder.WriteString(strings.Join(conditions, " AND "))
	}

	queryBuilder.WriteString(" ORDER BY created_at DESC LIMIT ? OFFSET ?")
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, queryBuilder.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]task.Task, 0, filter.Limit)
	for rows.Next() {
		taskItem, scanErr := scanTask(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		tasks = append(tasks, taskItem)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

func (r *Repository) Update(ctx context.Context, id uint64, params task.UpdateParams) (task.Task, error) {
	setClauses := make([]string, 0, 5)
	args := make([]any, 0, 6)

	if params.Title != nil {
		setClauses = append(setClauses, "title = ?")
		args = append(args, *params.Title)
	}
	if params.Description != nil {
		setClauses = append(setClauses, "description = ?")
		args = append(args, *params.Description)
	}
	if params.Status != nil {
		setClauses = append(setClauses, "status = ?")
		args = append(args, *params.Status)
	}
	if params.Priority != nil {
		setClauses = append(setClauses, "priority = ?")
		args = append(args, *params.Priority)
	}
	if params.DueAt != nil {
		setClauses = append(setClauses, "due_at = ?")
		args = append(args, params.DueAt.UTC())
	}
	if params.ClearDueAt {
		setClauses = append(setClauses, "due_at = NULL")
	}

	if len(setClauses) == 0 {
		return task.Task{}, task.ValidationError{Field: "body", Message: "at least one field must be provided for update"}
	}

	query := fmt.Sprintf("UPDATE tasks SET %s WHERE id = ?", strings.Join(setClauses, ", "))
	args = append(args, id)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return task.Task{}, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return task.Task{}, err
	}
	if rowsAffected == 0 {
		return task.Task{}, task.ErrTaskNotFound
	}

	return r.GetByID(ctx, id)
}

func (r *Repository) Delete(ctx context.Context, id uint64) error {
	const query = `DELETE FROM tasks WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return task.ErrTaskNotFound
	}

	return nil
}

type sqlScanner interface {
	Scan(dest ...any) error
}

func scanTask(scanner sqlScanner) (task.Task, error) {
	var (
		foundTask    task.Task
		description  sql.NullString
		dueAt        sql.NullTime
		createdAtRaw time.Time
		updatedAtRaw time.Time
	)

	err := scanner.Scan(
		&foundTask.ID,
		&foundTask.Title,
		&description,
		&foundTask.Status,
		&foundTask.Priority,
		&dueAt,
		&createdAtRaw,
		&updatedAtRaw,
	)
	if err != nil {
		return task.Task{}, err
	}

	if description.Valid {
		foundTask.Description = description.String
	}

	if dueAt.Valid {
		normalized := dueAt.Time.UTC()
		foundTask.DueAt = &normalized
	}

	foundTask.CreatedAt = createdAtRaw.UTC()
	foundTask.UpdatedAt = updatedAtRaw.UTC()

	return foundTask, nil
}

func asNullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC()
}
