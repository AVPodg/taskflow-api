package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"pgx-practice/internal/auth"
	"pgx-practice/internal/config"
	"pgx-practice/internal/handler"
	"pgx-practice/internal/middleware"
	"pgx-practice/internal/service"
	"pgx-practice/internal/storage"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config error", "error", err)
		os.Exit(1)
	}

	var logLevel slog.Level

	switch cfg.Log.Level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	ctx := context.Background()

	pgConfig, err := pgxpool.ParseConfig(cfg.Database.URL)
	if err != nil {
		slog.Error("config to postgres", "error", err)
		os.Exit(1)
	}

	pgConfig.MaxConns = cfg.Database.MaxConns
	pgConfig.MinConns = cfg.Database.MinConns

	pool, err := pgxpool.NewWithConfig(ctx, pgConfig)
	if err != nil {
		slog.Error("connect to postgres", "error", err)
		os.Exit(1)
	}

	defer pool.Close()

	if err = pool.Ping(ctx); err != nil {
		slog.Error("ping postgres", "error", err)
		os.Exit(1)
	}

	tokenManager := auth.NewTokenManager([]byte(cfg.JWT.Secret), cfg.JWT.TTL)

	repository := storage.NewRepository(pool)

	svc := service.NewService(repository, tokenManager)

	h := handler.NewHandler(svc, cfg.HTTP.Timeout)

	mux := http.NewServeMux()
	httpHandler := middleware.RequestIDMiddleware(middleware.LoggingMiddleware(middleware.RecoveryMiddleware(mux)))

	mux.Handle("POST /tasks", middleware.AuthMiddleware(http.HandlerFunc(h.PostApi), tokenManager))
	mux.Handle("GET /tasks/{task_id}", middleware.AuthMiddleware(http.HandlerFunc(h.GetApiByID), tokenManager))
	mux.Handle("GET /tasks", middleware.AuthMiddleware(http.HandlerFunc(h.GetApi), tokenManager))
	mux.Handle("DELETE /tasks/{task_id}", middleware.AuthMiddleware(http.HandlerFunc(h.DeleteApi), tokenManager))
	mux.Handle("PUT /tasks/{task_id}", middleware.AuthMiddleware(http.HandlerFunc(h.UpdateApi), tokenManager))

	mux.HandleFunc("POST /register", h.PostRegister)
	mux.HandleFunc("POST /login", h.PostLogin)

	stopCtx, stop := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	server := &http.Server{
		Addr:              ":" + cfg.HTTP.Port,
		Handler:           httpHandler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http server error", "error", err)
		}
	}()

	<-stopCtx.Done()

	shutdownCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)

		if err := server.Close(); err != nil {
			slog.Error("server close error", "error", err)
		}
	}
}
