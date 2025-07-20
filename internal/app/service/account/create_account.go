package account

import (
	"context"
	"fmt"

	"github.com/voidcontests/backend/internal/app/handler/dto/request"
	"github.com/voidcontests/backend/internal/app/service"
	"github.com/voidcontests/backend/internal/hasher"
)

func (s *Service) CreateAccount(ctx context.Context, body request.CreateAccount) (id int32, err error) {
	op := "service.CreateAccount"

	exists, err := s.repo.User.Exists(ctx, body.Username)
	if err != nil {
		return 0, fmt.Errorf("%s: can't verify that user exists or not: %v", op, err)
	}

	if exists {
		return 0, service.ErrUserAlreadyExists
	}

	passwordHash := hasher.Sha256String([]byte(body.Password), []byte(s.config.Security.Salt))
	id, err = s.repo.User.Create(ctx, body.Username, passwordHash)
	if err != nil {
		return 0, fmt.Errorf("%s: failed to create user: %v", op, err)
	}

	return id, nil
}
