package task

import (
	"context"
	"strings"
	"time"
)

const (
	defaultPriority uint8 = 3
	minPriority     uint8 = 1
	maxPriority     uint8 = 5
	defaultLimit          = 20
	maxLimit              = 100
	maxTitleLength        = 255
)

type Service interface {
	Create(ctx context.Context, input CreateTaskInput) (Task, error)
	GetByID(ctx context.Context, id uint64) (Task, error)
	List(ctx context.Context, input ListTasksInput) ([]Task, error)
	Update(ctx context.Context, id uint64, input UpdateTaskInput) (Task, error)
	Delete(ctx context.Context, id uint64) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, input CreateTaskInput) (Task, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return Task{}, ValidationError{Field: "title", Message: "must not be empty"}
	}
	if len(title) > maxTitleLength {
		return Task{}, ValidationError{Field: "title", Message: "must be at most 255 characters"}
	}

	status := StatusNew
	if input.Status != "" {
		parsedStatus, err := parseStatus(input.Status)
		if err != nil {
			return Task{}, err
		}
		status = parsedStatus
	}

	priority := defaultPriority
	if input.Priority != 0 {
		validatedPriority, err := validatePriority(input.Priority)
		if err != nil {
			return Task{}, err
		}
		priority = validatedPriority
	}

	dueAt, err := normalizeDueAt(input.DueAt)
	if err != nil {
		return Task{}, err
	}

	return s.repo.Create(ctx, CreateParams{
		Title:       title,
		Description: strings.TrimSpace(input.Description),
		Status:      status,
		Priority:    priority,
		DueAt:       dueAt,
	})
}

func (s *service) GetByID(ctx context.Context, id uint64) (Task, error) {
	if id == 0 {
		return Task{}, ValidationError{Field: "id", Message: "must be greater than 0"}
	}
	return s.repo.GetByID(ctx, id)
}

func (s *service) List(ctx context.Context, input ListTasksInput) ([]Task, error) {
	filter := ListFilter{
		Query:  strings.TrimSpace(input.Query),
		Limit:  input.Limit,
		Offset: input.Offset,
	}

	if input.Status != "" {
		parsedStatus, err := parseStatus(input.Status)
		if err != nil {
			return nil, err
		}
		filter.Status = &parsedStatus
	}

	if filter.Limit <= 0 {
		filter.Limit = defaultLimit
	}
	if filter.Limit > maxLimit {
		filter.Limit = maxLimit
	}
	if filter.Offset < 0 {
		return nil, ValidationError{Field: "offset", Message: "must be greater or equal to 0"}
	}

	return s.repo.List(ctx, filter)
}

func (s *service) Update(ctx context.Context, id uint64, input UpdateTaskInput) (Task, error) {
	if id == 0 {
		return Task{}, ValidationError{Field: "id", Message: "must be greater than 0"}
	}

	if input.ClearDueAt && input.DueAt != nil {
		return Task{}, ValidationError{Field: "due_at", Message: "cannot be provided when clear_due_at is true"}
	}

	params := UpdateParams{}
	fieldsToUpdate := 0

	if input.Title != nil {
		title := strings.TrimSpace(*input.Title)
		if title == "" {
			return Task{}, ValidationError{Field: "title", Message: "must not be empty"}
		}
		if len(title) > maxTitleLength {
			return Task{}, ValidationError{Field: "title", Message: "must be at most 255 characters"}
		}
		params.Title = &title
		fieldsToUpdate++
	}

	if input.Description != nil {
		description := strings.TrimSpace(*input.Description)
		params.Description = &description
		fieldsToUpdate++
	}

	if input.Status != nil {
		status, err := parseStatus(*input.Status)
		if err != nil {
			return Task{}, err
		}
		params.Status = &status
		fieldsToUpdate++
	}

	if input.Priority != nil {
		priority, err := validatePriority(*input.Priority)
		if err != nil {
			return Task{}, err
		}
		params.Priority = &priority
		fieldsToUpdate++
	}

	if input.DueAt != nil {
		dueAt, err := normalizeDueAt(input.DueAt)
		if err != nil {
			return Task{}, err
		}
		params.DueAt = dueAt
		fieldsToUpdate++
	}

	if input.ClearDueAt {
		params.ClearDueAt = true
		fieldsToUpdate++
	}

	if fieldsToUpdate == 0 {
		return Task{}, ValidationError{Field: "body", Message: "at least one field must be provided for update"}
	}

	return s.repo.Update(ctx, id, params)
}

func (s *service) Delete(ctx context.Context, id uint64) error {
	if id == 0 {
		return ValidationError{Field: "id", Message: "must be greater than 0"}
	}
	return s.repo.Delete(ctx, id)
}

func parseStatus(raw string) (Status, error) {
	status := Status(strings.ToLower(strings.TrimSpace(raw)))
	if !status.IsValid() {
		return "", ValidationError{Field: "status", Message: "must be one of: new, in_progress, done"}
	}
	return status, nil
}

func validatePriority(raw int) (uint8, error) {
	if raw < int(minPriority) || raw > int(maxPriority) {
		return 0, ValidationError{Field: "priority", Message: "must be between 1 and 5"}
	}
	return uint8(raw), nil
}

func normalizeDueAt(dueAt *time.Time) (*time.Time, error) {
	if dueAt == nil {
		return nil, nil
	}

	if dueAt.IsZero() {
		return nil, ValidationError{Field: "due_at", Message: "must be a valid timestamp"}
	}

	normalized := dueAt.UTC()
	return &normalized, nil
}
