package task

import (
	"context"
	"errors"
	"fmt"
	"pgx-practice/internal/domain"
)

type Storage interface {
	Save(context.Context, domain.Task) (domain.Task, error)
	ReadByID(context.Context, int, int) (domain.Task, error)
	Read(context.Context, int) ([]domain.Task, error)
	Delete(context.Context, int, int) error
	Update(context.Context, int, string, string, int) (domain.Task, error)
}

type Service struct {
	repo Storage
}

func NewService(s Storage) *Service {
	return &Service{
		repo: s,
	}
}

func (s *Service) CreateTask(ctx context.Context, title, status string, userID int) (domain.Task, error) {
	if title == "" {
		return domain.Task{}, fmt.Errorf("невалидный title: %w", errInvalidTask)
	}

	if status == "" {
		return domain.Task{}, fmt.Errorf("невалидный status: %w", errInvalidTask)
	}

	task, err := s.repo.Save(ctx, domain.Task{Title: title, Status: status, UserID: userID})
	if errors.Is(err, errForeignKey) {
		return domain.Task{}, fmt.Errorf("create task: %w", errUserNotFound)
	}

	if err != nil {
		return domain.Task{}, fmt.Errorf("create task: %w", err)
	}

	return task, nil
}

func (s *Service) GetTaskByID(ctx context.Context, id int, userID int) (domain.Task, error) {
	task, err := s.repo.ReadByID(ctx, id, userID)
	if errors.Is(err, errNotFound) {
		return domain.Task{}, fmt.Errorf("read task by id: %w", errTaskNotFound)
	}

	if err != nil {
		return domain.Task{}, fmt.Errorf("read task by id: %w", err)
	}

	return task, nil
}

func (s *Service) GetTasks(ctx context.Context, userID int) ([]domain.Task, error) {
	tasks, err := s.repo.Read(ctx, userID)

	if err != nil {
		return nil, fmt.Errorf("read tasks: %w", err)
	}

	return tasks, nil
}

func (s *Service) DeleteTask(ctx context.Context, id int, userID int) error {
	err := s.repo.Delete(ctx, id, userID)

	if errors.Is(err, errNotFound) {
		return fmt.Errorf("delete task: %w", errTaskNotFound)
	}

	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	return nil
}

func (s *Service) UpdateTask(ctx context.Context, id int, title, status string, userID int) (domain.Task, error) {
	if title == "" {
		return domain.Task{}, fmt.Errorf("неверный title: %w", errInvalidTask)
	}

	if status == "" {
		return domain.Task{}, fmt.Errorf("неверный status: %w", errInvalidTask)
	}

	task, err := s.repo.Update(ctx, id, title, status, userID)

	if errors.Is(err, errNotFound) {
		return domain.Task{}, fmt.Errorf("update task: %w", errTaskNotFound)
	}

	if err != nil {
		return domain.Task{}, fmt.Errorf("update task: %w", err)
	}

	return task, nil
}
