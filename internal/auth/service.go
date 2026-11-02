package auth

import (
	"context"
	"errors"
	"fmt"
	"pgx-practice/internal/domain"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Storage interface {
	FindUserByEmail(context.Context, string) (domain.User, error)
	SaveUser(context.Context, string, string) (domain.User, error)
	SaveRefreshSession(context.Context, int, string, uuid.UUID, time.Time) (domain.RefreshSession, error)
	RotateRefreshSession(context.Context, string, string, time.Time) (domain.RefreshSession, error)
	RevokeRefreshSessionByHash(context.Context, string) error
}

type TokenManagerService interface {
	Generate(int) (string, error)
	GenerateRefreshToken() (string, error)
	HashRefreshToken(string) string
}

type Service struct {
	repo         Storage
	tokenManager TokenManagerService
	refreshTTL   time.Duration
}

func NewService(s Storage, t TokenManagerService, r time.Duration) *Service {
	return &Service{
		repo:         s,
		tokenManager: t,
		refreshTTL:   r,
	}
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

	if errors.Is(err, ErrUnique) {
		return domain.User{}, fmt.Errorf("create user: %w", ErrUserAlreadyExists)
	}

	if err != nil {
		return domain.User{}, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

func (s *Service) LoginUser(ctx context.Context, email string, password string) (string, string, error) {
	if email == "" || len(password) < 8 {
		return "", "", fmt.Errorf("invalid credentials: %w", ErrInvalidUser)
	}

	user, err := s.repo.FindUserByEmail(ctx, email)
	if errors.Is(err, ErrUserNotFound) {
		return "", "", fmt.Errorf("login user error: %w", ErrInvalidCredentials)
	}

	if err != nil {
		return "", "", fmt.Errorf("login user error: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return "", "", fmt.Errorf("login user error: %w", ErrInvalidCredentials)
	}

	if err != nil {
		return "", "", fmt.Errorf("login user: %w", err)
	}

	accessToken, err := s.tokenManager.Generate(user.ID)
	if err != nil {
		return "", "", fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := s.tokenManager.GenerateRefreshToken()
	if err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}

	hashRefreshToken := s.tokenManager.HashRefreshToken(refreshToken)

	expiresAt := time.Now().Add(s.refreshTTL)

	familyID := uuid.New()

	_, err = s.repo.SaveRefreshSession(ctx, user.ID, hashRefreshToken, familyID, expiresAt)
	if err != nil {
		return "", "", fmt.Errorf("save refresh session: %w", err)
	}

	return accessToken, refreshToken, nil
}

func (s *Service) LogoutUser(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return fmt.Errorf("invalid hash: %w", ErrInvalidRefreshToken)
	}

	hash := s.tokenManager.HashRefreshToken(refreshToken)

	err := s.repo.RevokeRefreshSessionByHash(ctx, hash)
	if err != nil {
		return fmt.Errorf("logout user error: %w", err)
	}

	return nil
}

func (s *Service) RefreshTokens(ctx context.Context, refreshToken string) (string, string, error) {
	if refreshToken == "" {
		return "", "", fmt.Errorf("invalid input string: %w", ErrInvalidRefreshToken)
	}

	oldHash := s.tokenManager.HashRefreshToken(refreshToken)
	newRefreshToken, err := s.tokenManager.GenerateRefreshToken()
	if err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}

	newHash := s.tokenManager.HashRefreshToken(newRefreshToken)
	newExpiresAt := time.Now().Add(s.refreshTTL)

	refreshSession, err := s.repo.RotateRefreshSession(ctx, oldHash, newHash, newExpiresAt)
	if errors.Is(err, ErrRefreshSessionNotFound) ||
		errors.Is(err, ErrRefreshSessionRevoked) ||
		errors.Is(err, ErrRefreshSessionExpired) {

		return "", "", fmt.Errorf(
			"refresh tokens: %w",
			ErrInvalidRefreshToken,
		)
	}

	if err != nil {
		return "", "", fmt.Errorf("error refresh token: %w", err)
	}

	accessToken, err := s.tokenManager.Generate(refreshSession.UserID)
	if err != nil {
		return "", "", fmt.Errorf("generate access token: %w", err)
	}

	return accessToken, newRefreshToken, nil
}
