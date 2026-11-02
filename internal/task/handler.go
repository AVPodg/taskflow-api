package task

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"pgx-practice/internal/domain"
	"pgx-practice/internal/httputil"
	"pgx-practice/internal/middleware"
	"strconv"
	"time"
)

type TaskService interface {
	CreateTask(context.Context, string, string, int) (domain.Task, error)
	GetTaskByID(context.Context, int, int) (domain.Task, error)
	GetTasks(context.Context, int) ([]domain.Task, error)
	DeleteTask(context.Context, int, int) error
	UpdateTask(context.Context, int, string, string, int) (domain.Task, error)
}

type Handler struct {
	service TaskService
	timeout time.Duration
}

func NewHandler(h TaskService, t time.Duration) *Handler {
	return &Handler{
		service: h,
		timeout: t,
	}
}

type taskRequest struct {
	Title  string `json:"title"`
	Status string `json:"status"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.JSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var u taskRequest
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		httputil.JSONError(w, http.StatusBadRequest, "bad json")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	task, err := h.service.CreateTask(ctx, u.Title, u.Status, userID)

	if errors.Is(err, context.DeadlineExceeded) {
		httputil.JSONError(w, http.StatusGatewayTimeout, "request timeout")
		return
	}

	if errors.Is(err, context.Canceled) {
		httputil.JSONError(w, httputil.StatusClientCloseRequest, "client close request")
		return
	}

	if errors.Is(err, errUserNotFound) {
		httputil.JSONError(w, http.StatusNotFound, "user not found")
		return
	}

	if errors.Is(err, errInvalidTask) {
		httputil.JSONError(w, http.StatusBadRequest, "invalid task error")
		return
	}

	if err != nil {
		slog.Error("Invalid post task operation",
			"error", err,
			"statusCode", http.StatusInternalServerError,
			"request_id", middleware.RequestIDFromContext(r.Context()),
			"user_id", userID,
		)

		httputil.JSONError(w, http.StatusInternalServerError, "invalid server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.JSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idstr := r.PathValue("task_id")
	id, err := strconv.Atoi(idstr)
	if err != nil {
		httputil.JSONError(w, http.StatusBadRequest, "invalid URL")
		return
	}

	if id <= 0 {
		httputil.JSONError(w, http.StatusBadRequest, "invalid id")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	task, err := h.service.GetTaskByID(ctx, id, userID)

	if errors.Is(err, context.DeadlineExceeded) {
		httputil.JSONError(w, http.StatusGatewayTimeout, "request timeout")
		return
	}

	if errors.Is(err, context.Canceled) {
		httputil.JSONError(w, httputil.StatusClientCloseRequest, "client close request")
		return
	}

	if errors.Is(err, errTaskNotFound) {
		httputil.JSONError(w, http.StatusNotFound, "task not found")
		return
	}

	if err != nil {
		slog.Error("Invalid read by id task operation",
			"error", err,
			"statusCode", http.StatusInternalServerError,
			"request_id", middleware.RequestIDFromContext(r.Context()),
			"user_id", userID,
		)

		httputil.JSONError(w, http.StatusInternalServerError, "invalid server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(task)
}

func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.JSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	tasks, err := h.service.GetTasks(ctx, userID)

	if errors.Is(err, context.DeadlineExceeded) {
		httputil.JSONError(w, http.StatusGatewayTimeout, "request timeout")
		return
	}

	if errors.Is(err, context.Canceled) {
		httputil.JSONError(w, httputil.StatusClientCloseRequest, "client close request")
		return
	}

	if err != nil {
		slog.Error("Invalid read tasks operation",
			"error", err,
			"statusCode", http.StatusInternalServerError,
			"request_id", middleware.RequestIDFromContext(r.Context()),
			"user_id", userID,
		)

		httputil.JSONError(w, http.StatusInternalServerError, "invalid server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tasks)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.JSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := r.PathValue("task_id")
	id, err := strconv.Atoi(idStr)

	if err != nil {
		httputil.JSONError(w, http.StatusBadRequest, "invalid URL")
		return
	}

	if id <= 0 {
		httputil.JSONError(w, http.StatusBadRequest, "invalid id")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	err = h.service.DeleteTask(ctx, id, userID)

	if errors.Is(err, context.DeadlineExceeded) {
		httputil.JSONError(w, http.StatusGatewayTimeout, "request timeout")
		return
	}

	if errors.Is(err, context.Canceled) {
		httputil.JSONError(w, httputil.StatusClientCloseRequest, "client close request")
		return
	}

	if errors.Is(err, errTaskNotFound) {
		httputil.JSONError(w, http.StatusNotFound, "task not found")
		return
	}

	if err != nil {
		slog.Error("Invalid delete task operation",
			"error", err,
			"statusCode", http.StatusInternalServerError,
			"request_id", middleware.RequestIDFromContext(r.Context()),
			"user_id", userID,
		)

		httputil.JSONError(w, http.StatusInternalServerError, "invalid server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.JSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := r.PathValue("task_id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		httputil.JSONError(w, http.StatusBadRequest, "invalid URL")
		return
	}

	if id <= 0 {
		httputil.JSONError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var u taskRequest
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		httputil.JSONError(w, http.StatusBadRequest, "bad json")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	task, err := h.service.UpdateTask(ctx, id, u.Title, u.Status, userID)

	if errors.Is(err, context.DeadlineExceeded) {
		httputil.JSONError(w, http.StatusGatewayTimeout, "request timeout")
		return
	}

	if errors.Is(err, context.Canceled) {
		httputil.JSONError(w, httputil.StatusClientCloseRequest, "client close request")
		return
	}

	if errors.Is(err, errInvalidTask) {
		httputil.JSONError(w, http.StatusBadRequest, "invalid task error")
		return
	}

	if errors.Is(err, errTaskNotFound) {
		httputil.JSONError(w, http.StatusNotFound, "task not found")
		return
	}

	if err != nil {
		slog.Error("Invalid update task operation",
			"error", err,
			"statusCode", http.StatusInternalServerError,
			"request_id", middleware.RequestIDFromContext(r.Context()),
			"user_id", userID,
		)

		httputil.JSONError(w, http.StatusInternalServerError, "invalid server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(task)

}
