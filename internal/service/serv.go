package service

import (
	"context"
	"errors"
	"fmt"
	"pgx-practice/internal/domain"
	"pgx-practice/internal/storage"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidTask  = errors.New("invalid task")
	ErrTaskNotFound = errors.New("task not found")
	ErrUserNotFound = errors.New("user not found")

	ErrInvalidUser        = errors.New("invalid user")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserAlreadyExists  = errors.New("error unique")
)

type Storage interface {
	Save(context.Context, domain.Task) (domain.Task, error)
	ReadByID(context.Context, int, int) (domain.Task, error)
	Read(context.Context, int) ([]domain.Task, error)
	Delete(context.Context, int, int) error
	Update(context.Context, int, string, string, int) (domain.Task, error)

	FindUserByEmail(context.Context, string) (domain.User, error)
	SaveUser(context.Context, string, string) (domain.User, error)
}

type TokenManager interface {
	Generate(int) (string, error)
}

type Service struct {
	repo         Storage
	tokenManager TokenManager
}

func NewService(s Storage, t TokenManager) *Service {
	return &Service{
		repo:         s,
		tokenManager: t,
	}
}

func (s *Service) CreateTask(ctx context.Context, title, status string, userID int) (domain.Task, error) {
	if title == "" {
		return domain.Task{}, fmt.Errorf("Невалидный title: %w", ErrInvalidTask)
	}

	if status == "" {
		return domain.Task{}, fmt.Errorf("Невалидный status: %w", ErrInvalidTask)
	}

	task, err := s.repo.Save(ctx, domain.Task{Title: title, Status: status, UserID: userID})
	if errors.Is(err, storage.ErrForeignKey) {
		return domain.Task{}, fmt.Errorf("create task: %w", ErrUserNotFound)
	}

	if err != nil {
		return domain.Task{}, fmt.Errorf("create task: %w", err)
	}

	return task, nil
}

func (s *Service) GetTaskByID(ctx context.Context, id int, userID int) (domain.Task, error) {
	task, err := s.repo.ReadByID(ctx, id, userID)
	if errors.Is(err, storage.ErrNotFound) {
		return domain.Task{}, fmt.Errorf("Read task by id: %w", ErrTaskNotFound)
	}

	if err != nil {
		return domain.Task{}, fmt.Errorf("Read task by id: %w", err)
	}

	return task, nil
}

func (s *Service) GetTasks(ctx context.Context, userID int) ([]domain.Task, error) {
	tasks, err := s.repo.Read(ctx, userID)

	if err != nil {
		return nil, fmt.Errorf("Read tasks: %w", err)
	}

	return tasks, nil
}

func (s *Service) DeleteTask(ctx context.Context, id int, userID int) error {
	err := s.repo.Delete(ctx, id, userID)

	if errors.Is(err, storage.ErrNotFound) {
		return fmt.Errorf("delete task: %w", ErrTaskNotFound)
	}

	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	return nil
}

func (s *Service) UpdateTask(ctx context.Context, id int, title, status string, userID int) (domain.Task, error) {
	if title == "" {
		return domain.Task{}, fmt.Errorf("Неверный title: %w", ErrInvalidTask)
	}

	if status == "" {
		return domain.Task{}, fmt.Errorf("Неверный status: %w", ErrInvalidTask)
	}

	task, err := s.repo.Update(ctx, id, title, status, userID)

	if errors.Is(err, storage.ErrNotFound) {
		return domain.Task{}, fmt.Errorf("update task: %w", ErrTaskNotFound)
	}

	if err != nil {
		return domain.Task{}, fmt.Errorf("update task: %w", err)
	}

	return task, nil
}

func (s *Service) CreateUser(ctx context.Context, email string, password string) (domain.User, error) {
	if email == "" || len(password) < 8 {
		return domain.User{}, fmt.Errorf("invalid user data: %w", ErrInvalidUser)
	}

	passHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return domain.User{}, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.repo.SaveUser(ctx, email, string(passHash))

	if errors.Is(err, storage.ErrUnique) {
		return domain.User{}, fmt.Errorf("create user: %w", ErrUserAlreadyExists)
	}

	if err != nil {
		return domain.User{}, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

func (s *Service) LoginUser(ctx context.Context, email string, password string) (string, error) {
	if email == "" || len(password) < 8 {
		return "", fmt.Errorf("invalid credentials: %w", ErrInvalidUser)
	}

	user, err := s.repo.FindUserByEmail(ctx, email)
	if errors.Is(err, storage.ErrUserNotFound) {
		return "", fmt.Errorf("login user error: %w", ErrInvalidCredentials)
	}

	if err != nil {
		return "", fmt.Errorf("login user error: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return "", fmt.Errorf("login user error: %w", ErrInvalidCredentials)
	}

	if err != nil {
		return "", fmt.Errorf("login user: %w", err)
	}

	token, err := s.tokenManager.Generate(user.ID)
	if err != nil {
		return "", fmt.Errorf("generate access token: %w", err)
	}

	return token, nil
}
