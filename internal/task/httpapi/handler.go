package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PavelFesenkoFirst/task_tracker/internal/task"
)

type Handler struct {
	service task.Service
}

const maxRequestBodyBytes int64 = 1 << 20

type createTaskRequest struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Priority    int       `json:"priority"`
	DueAt       *taskTime `json:"due_at"`
}

type updateTaskRequest struct {
	Title       *string   `json:"title"`
	Description *string   `json:"description"`
	Status      *string   `json:"status"`
	Priority    *int      `json:"priority"`
	DueAt       *taskTime `json:"due_at"`
	ClearDueAt  bool      `json:"clear_due_at"`
}

type errorResponse struct {
	Error string `json:"error"`
	Field string `json:"field,omitempty"`
}

type taskTime struct {
	value string
}

func (t *taskTime) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	t.value = raw
	return nil
}

func NewHandler(service task.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /tasks", h.createTask)
	mux.HandleFunc("GET /tasks", h.listTasks)
	mux.HandleFunc("GET /tasks/{id}", h.getTask)
	mux.HandleFunc("PATCH /tasks/{id}", h.updateTask)
	mux.HandleFunc("DELETE /tasks/{id}", h.deleteTask)
}

func (h *Handler) createTask(w http.ResponseWriter, r *http.Request) {
	var request createTaskRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeDecodeError(w, err)
		return
	}

	dueAt, err := parseOptionalTime(request.DueAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, errorResponse{Error: err.Error(), Field: "due_at"})
		return
	}

	createdTask, err := h.service.Create(r.Context(), task.CreateTaskInput{
		Title:       request.Title,
		Description: request.Description,
		Status:      request.Status,
		Priority:    request.Priority,
		DueAt:       dueAt,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, createdTask)
}

func (h *Handler) listTasks(w http.ResponseWriter, r *http.Request) {
	limit, err := parseQueryInt(r.URL.Query().Get("limit"))
	if err != nil {
		writeError(w, http.StatusBadRequest, errorResponse{Error: "limit must be an integer", Field: "limit"})
		return
	}

	offset, err := parseQueryInt(r.URL.Query().Get("offset"))
	if err != nil {
		writeError(w, http.StatusBadRequest, errorResponse{Error: "offset must be an integer", Field: "offset"})
		return
	}

	tasks, err := h.service.List(r.Context(), task.ListTasksInput{
		Status: r.URL.Query().Get("status"),
		Query:  r.URL.Query().Get("q"),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, tasks)
}

func (h *Handler) getTask(w http.ResponseWriter, r *http.Request) {
	id, ok := parseTaskID(w, r)
	if !ok {
		return
	}

	foundTask, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, foundTask)
}

func (h *Handler) updateTask(w http.ResponseWriter, r *http.Request) {
	id, ok := parseTaskID(w, r)
	if !ok {
		return
	}

	var request updateTaskRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeDecodeError(w, err)
		return
	}

	dueAt, err := parseOptionalTime(request.DueAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, errorResponse{Error: err.Error(), Field: "due_at"})
		return
	}

	updatedTask, err := h.service.Update(r.Context(), id, task.UpdateTaskInput{
		Title:       request.Title,
		Description: request.Description,
		Status:      request.Status,
		Priority:    request.Priority,
		DueAt:       dueAt,
		ClearDueAt:  request.ClearDueAt,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, updatedTask)
}

func (h *Handler) deleteTask(w http.ResponseWriter, r *http.Request) {
	id, ok := parseTaskID(w, r)
	if !ok {
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		writeDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parseTaskID(w http.ResponseWriter, r *http.Request) (uint64, bool) {
	rawID := r.PathValue("id")
	id, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil || id == 0 {
		writeError(w, http.StatusBadRequest, errorResponse{Error: "id must be a positive integer", Field: "id"})
		return 0, false
	}
	return id, true
}

func parseQueryInt(raw string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, nil
	}
	return strconv.Atoi(raw)
}

func parseOptionalTime(value *taskTime) (*time.Time, error) {
	if value == nil {
		return nil, nil
	}

	parsed, err := time.Parse(time.RFC3339, value.value)
	if err != nil {
		return nil, errors.New("due_at must be a valid RFC3339 timestamp")
	}

	return &parsed, nil
}

var errRequestBodyTooLarge = errors.New("request body exceeds maximum size")

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	defer r.Body.Close()

	body := http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return errRequestBodyTooLarge
		}
		return err
	}

	var extra any
	if err := decoder.Decode(&extra); err == nil {
		return errors.New("request body must contain a single JSON object")
	} else if !errors.Is(err, io.EOF) {
		return err
	}

	return nil
}

func writeDomainError(w http.ResponseWriter, err error) {
	var validationErr task.ValidationError
	switch {
	case errors.As(err, &validationErr):
		writeError(w, http.StatusBadRequest, errorResponse{
			Error: validationErr.Message,
			Field: validationErr.Field,
		})
	case errors.Is(err, task.ErrTaskNotFound):
		writeError(w, http.StatusNotFound, errorResponse{Error: task.ErrTaskNotFound.Error()})
	default:
		writeError(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}

func writeDecodeError(w http.ResponseWriter, err error) {
	if errors.Is(err, errRequestBodyTooLarge) {
		writeError(w, http.StatusRequestEntityTooLarge, errorResponse{Error: err.Error()})
		return
	}
	writeError(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
}

func writeError(w http.ResponseWriter, status int, payload errorResponse) {
	writeJSON(w, status, payload)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
