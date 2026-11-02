package auth

import (
	"context"
	"errors"
	"fmt"
	"pgx-practice/internal/domain"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{
		db: pool,
	}
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

func (repo *Repository) SaveRefreshSession(ctx context.Context, userID int, tokenHash string, familyID uuid.UUID, expiresAt time.Time) (domain.RefreshSession, error) {
	query := `INSERT INTO refresh_sessions (user_id, token_hash, family_id, expires_at) VALUES ($1, $2, $3, $4) RETURNING id, user_id, token_hash, family_id, expires_at, revoked_at, created_at`

	var refreshSession domain.RefreshSession
	err := repo.db.QueryRow(ctx, query, userID, tokenHash, familyID, expiresAt).Scan(
		&refreshSession.ID,
		&refreshSession.UserID,
		&refreshSession.TokenHash,
		&refreshSession.FamilyID,
		&refreshSession.ExpiresAt,
		&refreshSession.RevokedAt,
		&refreshSession.CreatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23505":
				return domain.RefreshSession{}, fmt.Errorf("save refresh session error: %w", ErrUnique)
			case "23503":
				return domain.RefreshSession{}, fmt.Errorf("save resresh session error: %w", ErrForeignKey)
			}
		}

		return domain.RefreshSession{}, fmt.Errorf("save refresh session error: %w", err)
	}

	return refreshSession, nil
}

func (repo *Repository) FindRefreshSessionByHash(ctx context.Context, tokenHash string) (domain.RefreshSession, error) {
	query := `SELECT id, user_id, token_hash, family_id, expires_at, revoked_at, created_at FROM refresh_sessions WHERE token_hash = $1`

	var refreshSession domain.RefreshSession
	err := repo.db.QueryRow(ctx, query, tokenHash).Scan(
		&refreshSession.ID,
		&refreshSession.UserID,
		&refreshSession.TokenHash,
		&refreshSession.FamilyID,
		&refreshSession.ExpiresAt,
		&refreshSession.RevokedAt,
		&refreshSession.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return domain.RefreshSession{}, fmt.Errorf("find refresh session by hash error: %w", ErrRefreshSessionNotFound)
	}

	if err != nil {
		return domain.RefreshSession{}, fmt.Errorf("find refresh session by hash error: %w", err)
	}

	return refreshSession, nil
}

func (repo *Repository) RotateRefreshSession(ctx context.Context, oldTokenHash string, newTokenHash string, newExpiresAt time.Time) (domain.RefreshSession, error) {
	tx, err := repo.db.Begin(ctx)
	if err != nil {
		return domain.RefreshSession{}, fmt.Errorf("begin refresh rotation: %w", err)
	}
	defer tx.Rollback(ctx)

	var oldRefreshSession domain.RefreshSession
	err = tx.QueryRow(ctx, `SELECT id, user_id, token_hash, family_id, expires_at, revoked_at, created_at FROM refresh_sessions WHERE token_hash = $1 FOR UPDATE`, oldTokenHash).Scan(
		&oldRefreshSession.ID,
		&oldRefreshSession.UserID,
		&oldRefreshSession.TokenHash,
		&oldRefreshSession.FamilyID,
		&oldRefreshSession.ExpiresAt,
		&oldRefreshSession.RevokedAt,
		&oldRefreshSession.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.RefreshSession{}, fmt.Errorf("rotate refresh session: %w", ErrRefreshSessionNotFound)
	}

	if err != nil {
		return domain.RefreshSession{}, fmt.Errorf("rotate refresh session: %w", err)
	}

	if oldRefreshSession.RevokedAt != nil {
		return domain.RefreshSession{}, fmt.Errorf("rotate refresh session: %w", ErrRefreshSessionRevoked)
	}

	if time.Now().After(oldRefreshSession.ExpiresAt) {
		return domain.RefreshSession{}, fmt.Errorf("refresh session is finished: %w", ErrRefreshSessionExpired)
	}

	_, err = tx.Exec(ctx, `UPDATE refresh_sessions SET revoked_at = NOW() WHERE id = $1`, oldRefreshSession.ID)
	if err != nil {
		return domain.RefreshSession{}, fmt.Errorf("rotate refresh session: %w", err)
	}

	var newRefreshSession domain.RefreshSession
	err = tx.QueryRow(ctx, `INSERT INTO refresh_sessions (user_id, token_hash, family_id, expires_at) VALUES ($1, $2, $3, $4) RETURNING id, user_id, token_hash, family_id, expires_at, revoked_at, created_at`,
		oldRefreshSession.UserID, newTokenHash, oldRefreshSession.FamilyID, newExpiresAt).Scan(
		&newRefreshSession.ID,
		&newRefreshSession.UserID,
		&newRefreshSession.TokenHash,
		&newRefreshSession.FamilyID,
		&newRefreshSession.ExpiresAt,
		&newRefreshSession.RevokedAt,
		&newRefreshSession.CreatedAt,
	)

	if err != nil {
		return domain.RefreshSession{}, fmt.Errorf("create rotated refresh session: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.RefreshSession{}, fmt.Errorf("commit refresh error: %w", err)
	}

	return newRefreshSession, nil
}

func (repo *Repository) RevokeRefreshSessionByHash(ctx context.Context, tokenHash string) error {
	query := `UPDATE refresh_sessions SET revoked_at = NOW() WHERE token_hash = $1 AND revoked_at IS NULL`
	_, err := repo.db.Exec(ctx, query, tokenHash)
	if err != nil {
		return fmt.Errorf("revoke refresh session: %w", err)
	}

	return nil
}
