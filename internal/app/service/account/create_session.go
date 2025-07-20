package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/voidcontests/backend/internal/app/handler/dto/request"
	"github.com/voidcontests/backend/internal/app/service"
	"github.com/voidcontests/backend/internal/hasher"
	"github.com/voidcontests/backend/internal/jwt"
)

func (s *Service) CreateSession(ctx context.Context, body request.CreateSession) (token string, err error) {
	op := "service.CreateSession"

	passwordHash := hasher.Sha256String([]byte(body.Password), []byte(s.config.Security.Salt))
	user, err := s.repo.User.GetByCredentials(ctx, body.Username, passwordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", service.ErrUserNotFound
	}
	if err != nil {
		return "", fmt.Errorf("%s: can't create user: %v", op, err)
	}

	token, err = jwt.GenerateToken(user.ID, s.config.Security.SignatureKey)
	if err != nil {
		return "", fmt.Errorf("%s: can't generate token: %v", op, err)
	}

	return token, nil
}
