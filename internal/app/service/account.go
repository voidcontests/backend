package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/voidcontests/api/internal/config"
	"github.com/voidcontests/api/internal/hasher"
	"github.com/voidcontests/api/internal/jwt"
	"github.com/voidcontests/api/internal/storage/models"
	"github.com/voidcontests/api/internal/storage/repository"
)

type AccountService struct {
	config *config.Config
	repo   *repository.Repository
}

func NewAccountService(cfg *config.Config, repo *repository.Repository) *AccountService {
	return &AccountService{
		config: cfg,
		repo:   repo,
	}
}

func (s *AccountService) CreateAccount(ctx context.Context, username, password string) (int32, error) {
	op := "service.AccountService.CreateAccount"

	exists, err := s.repo.User.Exists(ctx, username)
	if err != nil {
		return 0, fmt.Errorf("%s: can't verify that user exists: %w", op, err)
	}

	if exists {
		return 0, ErrUserAlreadyExists
	}

	passwordHash := hasher.Sha256String([]byte(password), []byte(s.config.Security.Salt))

	user, err := s.repo.User.Create(ctx, username, passwordHash)
	if err != nil {
		return 0, fmt.Errorf("%s: failed to create user: %w", op, err)
	}

	return user.ID, nil
}

func (s *AccountService) CreateSession(ctx context.Context, username, password string) (string, error) {
	op := "service.AccountService.CreateSession"

	passwordHash := hasher.Sha256String([]byte(password), []byte(s.config.Security.Salt))

	user, err := s.repo.User.GetByCredentials(ctx, username, passwordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrInvalidCredentials
	}
	if err != nil {
		return "", fmt.Errorf("%s: failed to get user by credentials: %w", op, err)
	}

	token, err := jwt.GenerateToken(user.ID, s.config.Security.SignatureKey)
	if err != nil {
		return "", fmt.Errorf("%s: %w: %v", op, ErrTokenGeneration, err)
	}

	return token, nil
}

type AccountInfo struct {
	User models.User
	Role models.Role
}

func (s *AccountService) GetAccount(ctx context.Context, userID int32) (*AccountInfo, error) {
	op := "service.AccountService.GetAccount"

	user, err := s.repo.User.GetByID(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvalidToken
	}
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get user: %w", op, err)
	}

	role, err := s.repo.User.GetRole(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get role: %w", op, err)
	}

	return &AccountInfo{
		User: user,
		Role: role,
	}, nil
}
