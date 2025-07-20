package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/voidcontests/backend/internal/app/service"
	"github.com/voidcontests/backend/internal/repository/models"
)

func (s *Service) GetAccount(ctx context.Context, userID int32) (models.User, error) {
	op := "service.GetAccount"

	user, err := s.repo.User.GetByID(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.User{}, service.ErrUserNotFound
	}
	if err != nil {
		return models.User{}, fmt.Errorf("%s: can't get user: %v", op, err)
	}

	return user, nil
}
