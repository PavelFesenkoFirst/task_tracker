package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/PavelFesenkoFirst/task_tracker/internal/task"
)

type mockService struct {
	createInput task.CreateTaskInput
	updateInput task.UpdateTaskInput
	listInput   task.ListTasksInput
	getID       uint64
	updateID    uint64
	deleteID    uint64

	createCalled bool
	updateCalled bool
	listCalled   bool
	getCalled    bool
	deleteCalled bool

	createResult task.Task
	updateResult task.Task
	listResult   []task.Task
	getResult    task.Task

	createErr error
	updateErr error
	listErr   error
	getErr    error
	deleteErr error
}

func (m *mockService) Create(_ context.Context, input task.CreateTaskInput) (task.Task, error) {
	m.createCalled = true
	m.createInput = input
	if m.createErr != nil {
		return task.Task{}, m.createErr
	}
	return m.createResult, nil
}

func (m *mockService) GetByID(_ context.Context, id uint64) (task.Task, error) {
	m.getCalled = true
	m.getID = id
	if m.getErr != nil {
		return task.Task{}, m.getErr
	}
	return m.getResult, nil
}

func (m *mockService) List(_ context.Context, input task.ListTasksInput) ([]task.Task, error) {
	m.listCalled = true
	m.listInput = input
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.listResult, nil
}

func (m *mockService) Update(_ context.Context, id uint64, input task.UpdateTaskInput) (task.Task, error) {
	m.updateCalled = true
	m.updateID = id
	m.updateInput = input
	if m.updateErr != nil {
		return task.Task{}, m.updateErr
	}
	return m.updateResult, nil
}

func (m *mockService) Delete(_ context.Context, id uint64) error {
	m.deleteCalled = true
	m.deleteID = id
	return m.deleteErr
}

func TestHandlerCreateTask(t *testing.T) {
	svc := &mockService{
		createResult: task.Task{ID: 1, Title: "Write tests"},
	}
	mux := http.NewServeMux()
	NewHandler(svc).Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewBufferString(`{
		"title":"Write tests",
		"description":"cover handlers",
		"status":"new",
		"priority":4,
		"due_at":"2026-03-04T10:00:00Z"
	}`))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
	if !svc.createCalled {
		t.Fatal("expected create to be called")
	}
	if svc.createInput.Title != "Write tests" {
		t.Fatalf("unexpected title: %q", svc.createInput.Title)
	}
	if svc.createInput.Priority != 4 {
		t.Fatalf("unexpected priority: %d", svc.createInput.Priority)
	}
	if svc.createInput.DueAt == nil {
		t.Fatal("expected due_at to be parsed")
	}
}

func TestHandlerListTasks(t *testing.T) {
	svc := &mockService{
		listResult: []task.Task{{ID: 1}},
	}
	mux := http.NewServeMux()
	NewHandler(svc).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/tasks?status=done&q=report&limit=15&offset=5", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !svc.listCalled {
		t.Fatal("expected list to be called")
	}
	if svc.listInput.Status != "done" || svc.listInput.Query != "report" {
		t.Fatalf("unexpected list input: %+v", svc.listInput)
	}
	if svc.listInput.Limit != 15 || svc.listInput.Offset != 5 {
		t.Fatalf("unexpected pagination input: %+v", svc.listInput)
	}
}

func TestHandlerGetTask_NotFound(t *testing.T) {
	svc := &mockService{getErr: task.ErrTaskNotFound}
	mux := http.NewServeMux()
	NewHandler(svc).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/tasks/42", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
	if !svc.getCalled || svc.getID != 42 {
		t.Fatalf("unexpected get call: called=%v id=%d", svc.getCalled, svc.getID)
	}
}

func TestHandlerGetTask_Success(t *testing.T) {
	expected := task.Task{
		ID:          42,
		Title:       "Write CRUD API",
		Description: "Handler happy path",
		Status:      task.StatusInProgress,
		Priority:    4,
	}

	svc := &mockService{getResult: expected}
	mux := http.NewServeMux()
	NewHandler(svc).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/tasks/42", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !svc.getCalled || svc.getID != 42 {
		t.Fatalf("unexpected get call: called=%v id=%d", svc.getCalled, svc.getID)
	}

	var got task.Task
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got.ID != expected.ID || got.Title != expected.Title || got.Status != expected.Status || got.Priority != expected.Priority {
		t.Fatalf("unexpected response body: %+v", got)
	}
}

func TestHandlerUpdateTask_ClearDueAt(t *testing.T) {
	svc := &mockService{
		updateResult: task.Task{ID: 7},
	}
	mux := http.NewServeMux()
	NewHandler(svc).Register(mux)

	req := httptest.NewRequest(http.MethodPatch, "/tasks/7", bytes.NewBufferString(`{
		"title":"Updated",
		"clear_due_at":true
	}`))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !svc.updateCalled || svc.updateID != 7 {
		t.Fatalf("unexpected update call: called=%v id=%d", svc.updateCalled, svc.updateID)
	}
	if svc.updateInput.Title == nil || *svc.updateInput.Title != "Updated" {
		t.Fatalf("unexpected update title: %#v", svc.updateInput.Title)
	}
	if !svc.updateInput.ClearDueAt {
		t.Fatal("expected clear_due_at=true")
	}
}

func TestHandlerDeleteTask(t *testing.T) {
	svc := &mockService{}
	mux := http.NewServeMux()
	NewHandler(svc).Register(mux)

	req := httptest.NewRequest(http.MethodDelete, "/tasks/9", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}
	if !svc.deleteCalled || svc.deleteID != 9 {
		t.Fatalf("unexpected delete call: called=%v id=%d", svc.deleteCalled, svc.deleteID)
	}
}

func TestHandlerErrors(t *testing.T) {
	mux := http.NewServeMux()
	NewHandler(&mockService{}).Register(mux)

	t.Run("bad due_at", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewBufferString(`{"title":"x","due_at":"bad"}`))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("bad id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/tasks/abc", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("validation error mapping", func(t *testing.T) {
		svc := &mockService{createErr: task.ValidationError{Field: "title", Message: "must not be empty"}}
		mux := http.NewServeMux()
		NewHandler(svc).Register(mux)

		req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewBufferString(`{"title":"x"}`))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}

		var payload errorResponse
		if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if payload.Field != "title" {
			t.Fatalf("expected field title, got %q", payload.Field)
		}
	})

	t.Run("internal error mapping", func(t *testing.T) {
		svc := &mockService{listErr: errors.New("db down")}
		mux := http.NewServeMux()
		NewHandler(svc).Register(mux)

		req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
		}
	})

	t.Run("request body too large", func(t *testing.T) {
		oversized := `{"title":"x","description":"` + strings.Repeat("a", int(maxRequestBodyBytes)) + `"}`
		req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewBufferString(oversized))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("expected status %d, got %d", http.StatusRequestEntityTooLarge, rec.Code)
		}
	})
}

func TestTaskTimeUnmarshalJSON(t *testing.T) {
	var parsed taskTime
	if err := json.Unmarshal([]byte(`"2026-03-04T10:00:00Z"`), &parsed); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed.value != "2026-03-04T10:00:00Z" {
		t.Fatalf("unexpected parsed value: %q", parsed.value)
	}
}

func TestParseOptionalTime(t *testing.T) {
	got, err := parseOptionalTime(&taskTime{value: "2026-03-04T10:00:00Z"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected parsed time")
	}
	if !got.Equal(time.Date(2026, 3, 4, 10, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected time: %v", *got)
	}
}
