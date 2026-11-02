package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"pgx-practice/internal/domain"
	"pgx-practice/internal/httputil"
	"pgx-practice/internal/middleware"
	"time"
)

type req_user struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

type TaskService interface {
	CreateUser(context.Context, string, string) (domain.User, error)
	LoginUser(context.Context, string, string) (string, string, error)
	RefreshTokens(context.Context, string) (string, string, error)
	LogoutUser(context.Context, string) error
}

type handler struct {
	service        TaskService
	timeout        time.Duration
	authRefreshTTL time.Duration
}

func NewHandler(h TaskService, t time.Duration, a time.Duration) *handler {
	return &handler{
		service:        h,
		timeout:        t,
		authRefreshTTL: a,
	}
}

func (h *handler) PostRegister(w http.ResponseWriter, r *http.Request) {
	var u req_user
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		httputil.JSONError(w, http.StatusBadRequest, "bad json")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	user, err := h.service.CreateUser(ctx, u.Email, u.Password)

	if errors.Is(err, context.DeadlineExceeded) {
		httputil.JSONError(w, http.StatusGatewayTimeout, "request timeout")
		return
	}

	if errors.Is(err, context.Canceled) {
		httputil.JSONError(w, httputil.StatusClientCloseRequest, "client close request")
		return
	}

	if errors.Is(err, ErrInvalidUser) {
		httputil.JSONError(w, http.StatusBadRequest, "invalid email or password")
		return
	}

	if errors.Is(err, ErrUserAlreadyExists) {
		httputil.JSONError(w, http.StatusConflict, "email not unique")
		return
	}

	if err != nil {
		slog.Error("Invalid post user operation",
			"error", err, "statusCode", http.StatusInternalServerError, "request_id", middleware.RequestIDFromContext(r.Context()))

		httputil.JSONError(w, http.StatusInternalServerError, "invalid server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (h *handler) PostLogin(w http.ResponseWriter, r *http.Request) {
	var u req_user
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		httputil.JSONError(w, http.StatusBadRequest, "bad json")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	accessToken, refreshToken, err := h.service.LoginUser(ctx, u.Email, u.Password)

	if errors.Is(err, context.DeadlineExceeded) {
		httputil.JSONError(w, http.StatusGatewayTimeout, "request timeout")
		return
	}

	if errors.Is(err, context.Canceled) {
		httputil.JSONError(w, httputil.StatusClientCloseRequest, "client close request")
		return
	}

	if errors.Is(err, ErrInvalidUser) {
		httputil.JSONError(w, http.StatusBadRequest, "invalid email or password")
		return
	}

	if errors.Is(err, ErrInvalidCredentials) {
		httputil.JSONError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if err != nil {
		slog.Error("Invalid login user operation",
			"error", err,
			"statusCode", http.StatusInternalServerError,
			"request_id", middleware.RequestIDFromContext(r.Context()),
		)

		httputil.JSONError(w, http.StatusInternalServerError, "invalid server error")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Expires:  time.Now().Add(h.authRefreshTTL),
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

func (h *handler) PostLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if errors.Is(err, http.ErrNoCookie) {
		httputil.JSONError(w, http.StatusUnauthorized, "cookie error")
		return
	}
	if err != nil {
		httputil.JSONError(w, http.StatusUnauthorized, "error")
		return
	}

	refreshToken := cookie.Value

	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	err = h.service.LogoutUser(ctx, refreshToken)
	if err != nil {
		httputil.JSONError(w, http.StatusUnauthorized, "error")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})

	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) PostRefresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if errors.Is(err, http.ErrNoCookie) {
		httputil.JSONError(w, http.StatusUnauthorized, "no cookie")
		return
	}
	if err != nil {
		httputil.JSONError(w, http.StatusUnauthorized, "invalid...")
		return
	}

	refreshToken := cookie.Value

	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	accessToken, newRefreshToken, err := h.service.RefreshTokens(ctx, refreshToken)
	if err != nil {
		httputil.JSONError(w, http.StatusUnauthorized, "error")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    newRefreshToken,
		Expires:  time.Now().Add(h.authRefreshTTL),
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
