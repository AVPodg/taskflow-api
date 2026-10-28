package storage

import (
	"context"
	"errors"
	"fmt"
	"pgx-practice/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound   = errors.New("task not found")
	ErrUnique     = errors.New("error unique")
	ErrForeignKey = errors.New("error foreign key")

	ErrInvalidAmount   = errors.New("invalid amount")
	ErrProductNotFound = errors.New("product not found")
	ErrNotEnoughStock  = errors.New("stock < amount")

	ErrUserNotFound = errors.New("user not found")
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{
		db: pool,
	}
}

func (repo *Repository) Save(ctx context.Context, task domain.Task) (domain.Task, error) {
	query := `INSERT INTO tasks (title, status, user_id) VALUES($1, $2, $3) RETURNING id, title, status, created_at, user_id;`

	var createdTask domain.Task

	err := repo.db.QueryRow(ctx, query, task.Title, task.Status, task.UserID).Scan(
		&createdTask.ID,
		&createdTask.Title,
		&createdTask.Status,
		&createdTask.CreatedAt,
		&createdTask.UserID,
	)

	if err != nil {
		var pgxErr *pgconn.PgError
		if errors.As(err, &pgxErr) && pgxErr.Code == "23503" {
			return domain.Task{}, fmt.Errorf("save error: %w", ErrForeignKey)
		}

		return domain.Task{}, fmt.Errorf("save error: %w", err)
	}

	return createdTask, nil
}

func (repo *Repository) ReadByID(ctx context.Context, id int, userID int) (domain.Task, error) {
	query := `SELECT id, title, status, created_at, user_id FROM tasks WHERE id = $1 AND user_id = $2`

	var task domain.Task

	err := repo.db.QueryRow(ctx, query, id, userID).Scan(
		&task.ID,
		&task.Title,
		&task.Status,
		&task.CreatedAt,
		&task.UserID,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Task{}, fmt.Errorf("read task by id: %w", ErrNotFound)
	}

	if err != nil {
		return domain.Task{}, fmt.Errorf("read task by id: %w", err)
	}

	return task, nil
}

func (repo *Repository) Read(ctx context.Context, userID int) ([]domain.Task, error) {
	query := `SELECT id, title, status, created_at, user_id FROM tasks WHERE user_id = $1`

	rows, err := repo.db.Query(ctx, query, userID)

	if err != nil {
		return nil, fmt.Errorf("read tasks: %w", err)
	}

	defer rows.Close()

	tasks := make([]domain.Task, 0)

	for rows.Next() {
		var task domain.Task

		if err := rows.Scan(&task.ID, &task.Title, &task.Status, &task.CreatedAt, &task.UserID); err != nil {
			return nil, fmt.Errorf("read rows scan: %w", err)
		}

		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read rows error: %w", err)
	}

	return tasks, nil
}

func (repo *Repository) Delete(ctx context.Context, id int, userID int) error {
	query := `DELETE FROM tasks WHERE id = $1 AND user_id = $2`

	tag, err := repo.db.Exec(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("delete task: %w", ErrNotFound)
	}

	return nil
}

func (repo *Repository) Update(ctx context.Context, id int, title, status string, userID int) (domain.Task, error) {
	query := `UPDATE tasks SET title = $1, status = $2 WHERE id = $3 AND user_id = $4 RETURNING id, title, status, created_at, user_id`

	var task domain.Task

	err := repo.db.QueryRow(ctx, query, title, status, id, userID).Scan(
		&task.ID,
		&task.Title,
		&task.Status,
		&task.CreatedAt,
		&task.UserID,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Task{}, fmt.Errorf("update task: %w", ErrNotFound)
	}

	if err != nil {
		return domain.Task{}, fmt.Errorf("update task: %w", err)
	}

	return task, nil
}

func (repo *Repository) SaveUser(ctx context.Context, email string, passwordHash string) (domain.User, error) {
	query := `INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id, email, created_at`

	var user domain.User

	err := repo.db.QueryRow(ctx, query, email, passwordHash).Scan(&user.ID, &user.Email, &user.CreatedAt)

	if err != nil {
		var pgxErr *pgconn.PgError
		if errors.As(err, &pgxErr) && pgxErr.Code == "23505" {
			return domain.User{}, fmt.Errorf("save user: %w", ErrUnique)
		}

		return domain.User{}, fmt.Errorf("save user: %w", err)

	}

	return user, nil
}

func (repo *Repository) FindUserByEmail(ctx context.Context, email string) (domain.User, error) {
	query := `SELECT id, email, password_hash FROM users WHERE email=$1`

	var user domain.User

	err := repo.db.QueryRow(ctx, query, email).Scan(&user.ID, &user.Email, &user.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, fmt.Errorf("login error: %w", ErrUserNotFound)
	}

	if err != nil {
		return domain.User{}, fmt.Errorf("login error: %w", err)
	}

	return user, nil
}
