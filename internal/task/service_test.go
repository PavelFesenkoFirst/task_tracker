package task

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type mockRepository struct {
	createParams CreateParams
	updateParams UpdateParams
	listFilter   ListFilter
	getID        uint64
	updateID     uint64
	deleteID     uint64

	createCalled bool
	updateCalled bool
	listCalled   bool
	getCalled    bool
	deleteCalled bool

	createResult Task
	updateResult Task
	listResult   []Task
	getResult    Task

	createErr error
	updateErr error
	listErr   error
	getErr    error
	deleteErr error
}

func (m *mockRepository) Create(_ context.Context, params CreateParams) (Task, error) {
	m.createCalled = true
	m.createParams = params
	if m.createErr != nil {
		return Task{}, m.createErr
	}
	return m.createResult, nil
}

func (m *mockRepository) GetByID(_ context.Context, id uint64) (Task, error) {
	m.getCalled = true
	m.getID = id
	if m.getErr != nil {
		return Task{}, m.getErr
	}
	return m.getResult, nil
}

func (m *mockRepository) List(_ context.Context, filter ListFilter) ([]Task, error) {
	m.listCalled = true
	m.listFilter = filter
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.listResult, nil
}

func (m *mockRepository) Update(_ context.Context, id uint64, params UpdateParams) (Task, error) {
	m.updateCalled = true
	m.updateParams = params
	m.updateID = id
	if m.updateErr != nil {
		return Task{}, m.updateErr
	}
	return m.updateResult, nil
}

func (m *mockRepository) Delete(_ context.Context, id uint64) error {
	m.deleteCalled = true
	m.deleteID = id
	return m.deleteErr
}

func TestServiceCreate_DefaultsAndTrims(t *testing.T) {
	repo := &mockRepository{
		createResult: Task{ID: 10, Title: "Do work"},
	}
	svc := NewService(repo)

	got, err := svc.Create(context.Background(), CreateTaskInput{
		Title:       "  Do work  ",
		Description: "  important task  ",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !repo.createCalled {
		t.Fatal("expected repository create to be called")
	}
	if repo.createParams.Title != "Do work" {
		t.Fatalf("unexpected title: %q", repo.createParams.Title)
	}
	if repo.createParams.Description != "important task" {
		t.Fatalf("unexpected description: %q", repo.createParams.Description)
	}
	if repo.createParams.Status != StatusNew {
		t.Fatalf("unexpected status: %q", repo.createParams.Status)
	}
	if repo.createParams.Priority != defaultPriority {
		t.Fatalf("unexpected priority: %d", repo.createParams.Priority)
	}
	if repo.createParams.DueAt != nil {
		t.Fatalf("expected due_at nil, got %v", *repo.createParams.DueAt)
	}
	if got.ID != 10 {
		t.Fatalf("unexpected created task ID: %d", got.ID)
	}
}

func TestServiceCreate_ValidationErrors(t *testing.T) {
	now := time.Now()
	zero := time.Time{}

	tests := []struct {
		name  string
		input CreateTaskInput
		field string
	}{
		{
			name:  "empty title",
			input: CreateTaskInput{Title: "  "},
			field: "title",
		},
		{
			name:  "too long title",
			input: CreateTaskInput{Title: strings.Repeat("a", maxTitleLength+1)},
			field: "title",
		},
		{
			name:  "invalid status",
			input: CreateTaskInput{Title: "ok", Status: "bad"},
			field: "status",
		},
		{
			name:  "invalid priority",
			input: CreateTaskInput{Title: "ok", Priority: 6},
			field: "priority",
		},
		{
			name:  "zero due_at",
			input: CreateTaskInput{Title: "ok", DueAt: &zero},
			field: "due_at",
		},
		{
			name:  "priority below range",
			input: CreateTaskInput{Title: "ok", Priority: -1},
			field: "priority",
		},
		{
			name:  "invalid status with spaces",
			input: CreateTaskInput{Title: "ok", Status: " donee "},
			field: "status",
		},
		{
			name:  "valid due at should not fail",
			input: CreateTaskInput{Title: "ok", DueAt: &now, Status: "done", Priority: 4},
			field: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockRepository{createResult: Task{ID: 1}}
			svc := NewService(repo)

			_, err := svc.Create(context.Background(), tc.input)
			if tc.field == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if !repo.createCalled {
					t.Fatal("expected repository create to be called")
				}
				return
			}

			if err == nil {
				t.Fatal("expected validation error, got nil")
			}

			var verr ValidationError
			if !errors.As(err, &verr) {
				t.Fatalf("expected ValidationError, got %T", err)
			}
			if verr.Field != tc.field {
				t.Fatalf("expected field %q, got %q", tc.field, verr.Field)
			}
			if repo.createCalled {
				t.Fatal("repository should not be called on validation errors")
			}
		})
	}
}

func TestServiceGetByID_ValidationAndPassThrough(t *testing.T) {
	repo := &mockRepository{getResult: Task{ID: 7}}
	svc := NewService(repo)

	_, err := svc.GetByID(context.Background(), 0)
	if err == nil {
		t.Fatal("expected validation error for zero id")
	}
	if repo.getCalled {
		t.Fatal("repository should not be called for zero id")
	}

	got, err := svc.GetByID(context.Background(), 7)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repo.getCalled {
		t.Fatal("expected repository to be called")
	}
	if repo.getID != 7 {
		t.Fatalf("expected id 7, got %d", repo.getID)
	}
	if got.ID != 7 {
		t.Fatalf("expected result ID 7, got %d", got.ID)
	}
}

func TestServiceList_NormalizationAndValidation(t *testing.T) {
	repo := &mockRepository{
		listResult: []Task{{ID: 1}},
	}
	svc := NewService(repo)

	_, err := svc.List(context.Background(), ListTasksInput{
		Status: "bad",
	})
	if err == nil {
		t.Fatal("expected validation error for bad status")
	}
	if repo.listCalled {
		t.Fatal("repository should not be called for invalid status")
	}

	_, err = svc.List(context.Background(), ListTasksInput{
		Offset: -1,
	})
	if err == nil {
		t.Fatal("expected validation error for negative offset")
	}
	if repo.listCalled {
		t.Fatal("repository should not be called for invalid offset")
	}

	_, err = svc.List(context.Background(), ListTasksInput{
		Status: " done ",
		Query:  "  search text  ",
		Limit:  999,
		Offset: 5,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repo.listCalled {
		t.Fatal("expected repository list to be called")
	}
	if repo.listFilter.Status == nil || *repo.listFilter.Status != StatusDone {
		t.Fatalf("expected status filter %q, got %#v", StatusDone, repo.listFilter.Status)
	}
	if repo.listFilter.Query != "search text" {
		t.Fatalf("unexpected query: %q", repo.listFilter.Query)
	}
	if repo.listFilter.Limit != maxLimit {
		t.Fatalf("expected capped limit %d, got %d", maxLimit, repo.listFilter.Limit)
	}
	if repo.listFilter.Offset != 5 {
		t.Fatalf("expected offset 5, got %d", repo.listFilter.Offset)
	}
}

func TestServiceList_DefaultLimitWhenZero(t *testing.T) {
	repo := &mockRepository{
		listResult: []Task{{ID: 1}},
	}
	svc := NewService(repo)

	_, err := svc.List(context.Background(), ListTasksInput{
		Limit:  0,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repo.listCalled {
		t.Fatal("expected repository list to be called")
	}
	if repo.listFilter.Limit != defaultLimit {
		t.Fatalf("expected default limit %d, got %d", defaultLimit, repo.listFilter.Limit)
	}
}

func TestServiceUpdate_ValidationAndTransformation(t *testing.T) {
	validationRepo := &mockRepository{}
	svc := NewService(validationRepo)

	title := "  "
	_, err := svc.Update(context.Background(), 1, UpdateTaskInput{Title: &title})
	if err == nil {
		t.Fatal("expected validation error for empty title")
	}
	if validationRepo.updateCalled {
		t.Fatal("repository should not be called for invalid title")
	}

	longTitle := strings.Repeat("a", maxTitleLength+1)
	_, err = svc.Update(context.Background(), 1, UpdateTaskInput{Title: &longTitle})
	if err == nil {
		t.Fatal("expected validation error for too long title")
	}
	var longTitleErr ValidationError
	if !errors.As(err, &longTitleErr) || longTitleErr.Field != "title" {
		t.Fatalf("expected title ValidationError, got %v", err)
	}
	if validationRepo.updateCalled {
		t.Fatal("repository should not be called for too long title")
	}

	dueAt := time.Now()
	_, err = svc.Update(context.Background(), 1, UpdateTaskInput{
		DueAt:      &dueAt,
		ClearDueAt: true,
	})
	if err == nil {
		t.Fatal("expected validation error for clear_due_at + due_at")
	}
	if validationRepo.updateCalled {
		t.Fatal("repository should not be called for invalid due_at + clear_due_at")
	}

	zeroDueAt := time.Time{}
	_, err = svc.Update(context.Background(), 1, UpdateTaskInput{DueAt: &zeroDueAt})
	if err == nil {
		t.Fatal("expected validation error for zero due_at")
	}
	var dueAtErr ValidationError
	if !errors.As(err, &dueAtErr) || dueAtErr.Field != "due_at" {
		t.Fatalf("expected due_at ValidationError, got %v", err)
	}
	if validationRepo.updateCalled {
		t.Fatal("repository should not be called for zero due_at")
	}

	_, err = svc.Update(context.Background(), 0, UpdateTaskInput{})
	if err == nil {
		t.Fatal("expected validation error for zero id")
	}
	if validationRepo.updateCalled {
		t.Fatal("repository should not be called for invalid id")
	}

	_, err = svc.Update(context.Background(), 1, UpdateTaskInput{})
	if err == nil {
		t.Fatal("expected validation error for empty update payload")
	}
	if validationRepo.updateCalled {
		t.Fatal("repository should not be called for empty update payload")
	}

	invalidStatus := "broken"
	_, err = svc.Update(context.Background(), 1, UpdateTaskInput{Status: &invalidStatus})
	if err == nil {
		t.Fatal("expected validation error for invalid status")
	}
	var statusErr ValidationError
	if !errors.As(err, &statusErr) || statusErr.Field != "status" {
		t.Fatalf("expected status ValidationError, got %v", err)
	}
	if validationRepo.updateCalled {
		t.Fatal("repository should not be called for invalid status")
	}

	invalidPriority := 6
	_, err = svc.Update(context.Background(), 1, UpdateTaskInput{Priority: &invalidPriority})
	if err == nil {
		t.Fatal("expected validation error for invalid priority")
	}
	var priorityErr ValidationError
	if !errors.As(err, &priorityErr) || priorityErr.Field != "priority" {
		t.Fatalf("expected priority ValidationError, got %v", err)
	}
	if validationRepo.updateCalled {
		t.Fatal("repository should not be called for invalid priority")
	}

	repo := &mockRepository{updateResult: Task{ID: 9}}
	svc = NewService(repo)

	clearDueAtRepo := &mockRepository{updateResult: Task{ID: 8}}
	svc = NewService(clearDueAtRepo)

	cleared, err := svc.Update(context.Background(), 8, UpdateTaskInput{
		ClearDueAt: true,
	})
	if err != nil {
		t.Fatalf("expected no error for clear_due_at only, got %v", err)
	}
	if !clearDueAtRepo.updateCalled {
		t.Fatal("expected repository update to be called for clear_due_at only")
	}
	if clearDueAtRepo.updateID != 8 {
		t.Fatalf("expected update id 8, got %d", clearDueAtRepo.updateID)
	}
	if !clearDueAtRepo.updateParams.ClearDueAt {
		t.Fatal("expected clear_due_at to be true in repository params")
	}
	if clearDueAtRepo.updateParams.DueAt != nil {
		t.Fatalf("expected due_at to be nil when clear_due_at=true, got %v", *clearDueAtRepo.updateParams.DueAt)
	}
	if cleared.ID != 8 {
		t.Fatalf("expected updated task ID 8, got %d", cleared.ID)
	}

	repo = &mockRepository{updateResult: Task{ID: 9}}
	svc = NewService(repo)

	newTitle := "  Updated title  "
	newDescription := "  Updated description  "
	newStatus := "in_progress"
	newPriority := 5
	newDueAt := time.Now()

	got, err := svc.Update(context.Background(), 9, UpdateTaskInput{
		Title:       &newTitle,
		Description: &newDescription,
		Status:      &newStatus,
		Priority:    &newPriority,
		DueAt:       &newDueAt,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !repo.updateCalled {
		t.Fatal("expected repository update to be called")
	}
	if repo.updateID != 9 {
		t.Fatalf("expected update id 9, got %d", repo.updateID)
	}
	if repo.updateParams.Title == nil || *repo.updateParams.Title != "Updated title" {
		t.Fatalf("unexpected title param: %#v", repo.updateParams.Title)
	}
	if repo.updateParams.Description == nil || *repo.updateParams.Description != "Updated description" {
		t.Fatalf("unexpected description param: %#v", repo.updateParams.Description)
	}
	if repo.updateParams.Status == nil || *repo.updateParams.Status != StatusInProgress {
		t.Fatalf("unexpected status param: %#v", repo.updateParams.Status)
	}
	if repo.updateParams.Priority == nil || *repo.updateParams.Priority != 5 {
		t.Fatalf("unexpected priority param: %#v", repo.updateParams.Priority)
	}
	if repo.updateParams.DueAt == nil {
		t.Fatal("expected due_at to be set")
	}
	if !repo.updateParams.DueAt.Equal(newDueAt.UTC()) {
		t.Fatalf("expected due_at UTC %v, got %v", newDueAt.UTC(), *repo.updateParams.DueAt)
	}
	if got.ID != 9 {
		t.Fatalf("expected updated task ID 9, got %d", got.ID)
	}
}

func TestServiceDelete_ValidationAndPassThrough(t *testing.T) {
	repo := &mockRepository{}
	svc := NewService(repo)

	err := svc.Delete(context.Background(), 0)
	if err == nil {
		t.Fatal("expected validation error for zero id")
	}
	if repo.deleteCalled {
		t.Fatal("repository should not be called for zero id")
	}

	err = svc.Delete(context.Background(), 12)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repo.deleteCalled {
		t.Fatal("expected repository delete to be called")
	}
	if repo.deleteID != 12 {
		t.Fatalf("expected delete id 12, got %d", repo.deleteID)
	}
}

func TestService_RepositoryErrorsArePropagated(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		repoErr := errors.New("create failed")
		repo := &mockRepository{createErr: repoErr}
		svc := NewService(repo)

		_, err := svc.Create(context.Background(), CreateTaskInput{Title: "task"})
		if !errors.Is(err, repoErr) {
			t.Fatalf("expected error %v, got %v", repoErr, err)
		}
		if !repo.createCalled {
			t.Fatal("expected repository create to be called")
		}
	})

	t.Run("get by id", func(t *testing.T) {
		repoErr := errors.New("get failed")
		repo := &mockRepository{getErr: repoErr}
		svc := NewService(repo)

		_, err := svc.GetByID(context.Background(), 1)
		if !errors.Is(err, repoErr) {
			t.Fatalf("expected error %v, got %v", repoErr, err)
		}
		if !repo.getCalled {
			t.Fatal("expected repository get to be called")
		}
	})

	t.Run("list", func(t *testing.T) {
		repoErr := errors.New("list failed")
		repo := &mockRepository{listErr: repoErr}
		svc := NewService(repo)

		_, err := svc.List(context.Background(), ListTasksInput{})
		if !errors.Is(err, repoErr) {
			t.Fatalf("expected error %v, got %v", repoErr, err)
		}
		if !repo.listCalled {
			t.Fatal("expected repository list to be called")
		}
	})

	t.Run("update", func(t *testing.T) {
		repoErr := errors.New("update failed")
		repo := &mockRepository{updateErr: repoErr}
		svc := NewService(repo)
		status := "done"

		_, err := svc.Update(context.Background(), 1, UpdateTaskInput{Status: &status})
		if !errors.Is(err, repoErr) {
			t.Fatalf("expected error %v, got %v", repoErr, err)
		}
		if !repo.updateCalled {
			t.Fatal("expected repository update to be called")
		}
	})

	t.Run("delete", func(t *testing.T) {
		repoErr := errors.New("delete failed")
		repo := &mockRepository{deleteErr: repoErr}
		svc := NewService(repo)

		err := svc.Delete(context.Background(), 1)
		if !errors.Is(err, repoErr) {
			t.Fatalf("expected error %v, got %v", repoErr, err)
		}
		if !repo.deleteCalled {
			t.Fatal("expected repository delete to be called")
		}
	})
}
