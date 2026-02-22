package task

import "context"

type Repository interface {
	Create(ctx context.Context, params CreateParams) (Task, error)
	GetByID(ctx context.Context, id uint64) (Task, error)
	List(ctx context.Context, filter ListFilter) ([]Task, error)
	Update(ctx context.Context, id uint64, params UpdateParams) (Task, error)
	Delete(ctx context.Context, id uint64) error
}
