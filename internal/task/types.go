package task

import "time"

type Status string

const (
	StatusNew        Status = "new"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

func (s Status) IsValid() bool {
	switch s {
	case StatusNew, StatusInProgress, StatusDone:
		return true
	default:
		return false
	}
}

type Task struct {
	ID          uint64     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      Status     `json:"status"`
	Priority    uint8      `json:"priority"`
	DueAt       *time.Time `json:"due_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type CreateTaskInput struct {
	Title       string
	Description string
	Status      string
	Priority    int
	DueAt       *time.Time
}

type UpdateTaskInput struct {
	Title       *string
	Description *string
	Status      *string
	Priority    *int
	DueAt       *time.Time
	ClearDueAt  bool
}

type ListTasksInput struct {
	Status string
	Query  string
	Limit  int
	Offset int
}

type ListFilter struct {
	Status *Status
	Query  string
	Limit  int
	Offset int
}

type CreateParams struct {
	Title       string
	Description string
	Status      Status
	Priority    uint8
	DueAt       *time.Time
}

type UpdateParams struct {
	Title       *string
	Description *string
	Status      *Status
	Priority    *uint8
	DueAt       *time.Time
	ClearDueAt  bool
}
