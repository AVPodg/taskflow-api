package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"pgx-practice/internal/domain"
	"pgx-practice/internal/middleware"
	"pgx-practice/internal/service"
	"strconv"
	"time"
)

const (
	statusClientCloseRequest = 499
)

type req_user struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type req struct {
	Title  string `json:"title"`
	Status string `json:"status"`
}

type loginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

type TaskService interface {
	CreateTask(context.Context, string, string, int) (domain.Task, error)
	GetTaskByID(context.Context, int, int) (domain.Task, error)
	GetTasks(context.Context, int) ([]domain.Task, error)
	DeleteTask(context.Context, int, int) error
	UpdateTask(context.Context, int, string, string, int) (domain.Task, error)

	CreateUser(context.Context, string, string) (domain.User, error)
	LoginUser(context.Context, string, string) (string, string, error)
}

type handler struct {
	service TaskService
	timeout time.Duration
}

func NewHandler(h TaskService, t time.Duration) *handler {
	return &handler{
		service: h,
		timeout: t,
	}
}

func (h *handler) PostApi(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var u req
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		jsonError(w, http.StatusBadRequest, "bad json")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	task, err := h.service.CreateTask(ctx, u.Title, u.Status, userID)

	if errors.Is(err, context.DeadlineExceeded) {
		jsonError(w, http.StatusGatewayTimeout, "request timeout")
		return
	}

	if errors.Is(err, context.Canceled) {
		jsonError(w, statusClientCloseRequest, "client close request")
		return
	}

	if errors.Is(err, service.ErrUserNotFound) {
		jsonError(w, http.StatusNotFound, "user not found")
		return
	}

	if errors.Is(err, service.ErrInvalidTask) {
		jsonError(w, http.StatusBadRequest, "invalid task error")
		return
	}

	if err != nil {
		slog.Error("Invalid post task operation",
			"error", err,
			"statusCode", http.StatusInternalServerError,
			"request_id", middleware.RequestIDFromContext(r.Context()),
			"user_id", userID,
		)

		jsonError(w, http.StatusInternalServerError, "invalid server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

func (h *handler) GetApiByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idstr := r.PathValue("task_id")
	id, err := strconv.Atoi(idstr)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid URL")
		return
	}

	if id <= 0 {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	task, err := h.service.GetTaskByID(ctx, id, userID)

	if errors.Is(err, context.DeadlineExceeded) {
		jsonError(w, http.StatusGatewayTimeout, "request timeout")
		return
	}

	if errors.Is(err, context.Canceled) {
		jsonError(w, statusClientCloseRequest, "client close request")
		return
	}

	if errors.Is(err, service.ErrTaskNotFound) {
		jsonError(w, http.StatusNotFound, "task not found")
		return
	}

	if err != nil {
		slog.Error("Invalid read by id task operation",
			"error", err,
			"statusCode", http.StatusInternalServerError,
			"request_id", middleware.RequestIDFromContext(r.Context()),
			"user_id", userID,
		)

		jsonError(w, http.StatusInternalServerError, "invalid server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(task)
}

func (h *handler) GetApi(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	tasks, err := h.service.GetTasks(ctx, userID)

	if errors.Is(err, context.DeadlineExceeded) {
		jsonError(w, http.StatusGatewayTimeout, "request timeout")
		return
	}

	if errors.Is(err, context.Canceled) {
		jsonError(w, statusClientCloseRequest, "client close request")
		return
	}

	if err != nil {
		slog.Error("Invalid read tasks operation",
			"error", err,
			"statusCode", http.StatusInternalServerError,
			"request_id", middleware.RequestIDFromContext(r.Context()),
			"user_id", userID,
		)

		jsonError(w, http.StatusInternalServerError, "invalid server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tasks)
}

func (h *handler) DeleteApi(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := r.PathValue("task_id")
	id, err := strconv.Atoi(idStr)

	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid URL")
		return
	}

	if id <= 0 {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	err = h.service.DeleteTask(ctx, id, userID)

	if errors.Is(err, context.DeadlineExceeded) {
		jsonError(w, http.StatusGatewayTimeout, "request timeout")
		return
	}

	if errors.Is(err, context.Canceled) {
		jsonError(w, statusClientCloseRequest, "client close request")
		return
	}

	if errors.Is(err, service.ErrTaskNotFound) {
		jsonError(w, http.StatusNotFound, "task not found")
		return
	}

	if err != nil {
		slog.Error("Invalid delete task operation",
			"error", err,
			"statusCode", http.StatusInternalServerError,
			"request_id", middleware.RequestIDFromContext(r.Context()),
			"user_id", userID,
		)

		jsonError(w, http.StatusInternalServerError, "invalid server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) UpdateApi(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := r.PathValue("task_id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid URL")
		return
	}

	if id <= 0 {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var u req
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		jsonError(w, http.StatusBadRequest, "bad json")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	task, err := h.service.UpdateTask(ctx, id, u.Title, u.Status, userID)

	if errors.Is(err, context.DeadlineExceeded) {
		jsonError(w, http.StatusGatewayTimeout, "request timeout")
		return
	}

	if errors.Is(err, context.Canceled) {
		jsonError(w, statusClientCloseRequest, "client close request")
		return
	}

	if errors.Is(err, service.ErrInvalidTask) {
		jsonError(w, http.StatusBadRequest, "invalid task error")
		return
	}

	if errors.Is(err, service.ErrTaskNotFound) {
		jsonError(w, http.StatusNotFound, "task not found")
		return
	}

	if err != nil {
		slog.Error("Invalid update task operation",
			"error", err,
			"statusCode", http.StatusInternalServerError,
			"request_id", middleware.RequestIDFromContext(r.Context()),
			"user_id", userID,
		)

		jsonError(w, http.StatusInternalServerError, "invalid server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(task)

}

func (h *handler) PostRegister(w http.ResponseWriter, r *http.Request) {
	var u req_user
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		jsonError(w, http.StatusBadRequest, "bad json")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	user, err := h.service.CreateUser(ctx, u.Email, u.Password)

	if errors.Is(err, context.DeadlineExceeded) {
		jsonError(w, http.StatusGatewayTimeout, "request timeout")
		return
	}

	if errors.Is(err, context.Canceled) {
		jsonError(w, statusClientCloseRequest, "client close request")
		return
	}

	if errors.Is(err, service.ErrInvalidUser) {
		jsonError(w, http.StatusBadRequest, "invalid email or password")
		return
	}

	if errors.Is(err, service.ErrUserAlreadyExists) {
		jsonError(w, http.StatusConflict, "email not unique")
		return
	}

	if err != nil {
		slog.Error("Invalid post user operation",
			"error", err, "statusCode", http.StatusInternalServerError, "request_id", middleware.RequestIDFromContext(r.Context()))

		jsonError(w, http.StatusInternalServerError, "invalid server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (h *handler) PostLogin(w http.ResponseWriter, r *http.Request) {
	var u req_user
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		jsonError(w, http.StatusBadRequest, "bad json")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	accessToken, refreshToken, err := h.service.LoginUser(ctx, u.Email, u.Password)

	if errors.Is(err, context.DeadlineExceeded) {
		jsonError(w, http.StatusGatewayTimeout, "request timeout")
		return
	}

	if errors.Is(err, context.Canceled) {
		jsonError(w, statusClientCloseRequest, "client close request")
		return
	}

	if errors.Is(err, service.ErrInvalidUser) {
		jsonError(w, http.StatusBadRequest, "invalid email or password")
		return
	}

	if errors.Is(err, service.ErrInvalidCredentials) {
		jsonError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if err != nil {
		slog.Error("Invalid login user operation",
			"error", err,
			"statusCode", http.StatusInternalServerError,
			"request_id", middleware.RequestIDFromContext(r.Context()),
		)

		jsonError(w, http.StatusInternalServerError, "invalid server error")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(loginResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
	})
}

func jsonError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
