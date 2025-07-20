package contest

import (
	"context"
	"fmt"

	"github.com/voidcontests/backend/internal/repository/models"
)

func (s *Service) GetContests(ctx context.Context, limit, offset int) ([]models.Contest, int, error) {
	op := "service.GetContests"

	contests, total, err := s.repo.Contest.ListAll(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("%s: can't get contests: %v", op, err)
	}

	return contests, total, nil
}
