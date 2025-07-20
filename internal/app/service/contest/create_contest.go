package contest

import (
	"context"
	"fmt"

	"github.com/voidcontests/backend/internal/app/handler/dto/request"
	"github.com/voidcontests/backend/internal/app/service"
	"github.com/voidcontests/backend/internal/repository/models"
)

func (s *Service) CreateContest(ctx context.Context, userID int32, body request.CreateContestRequest) (id int32, err error) {
	op := "service.CreateContest"

	userRole, err := s.repo.User.GetRole(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("%s: can't get role: %v", op, err)
	}

	if userRole.Name == models.RoleBanned {
		return 0, service.ErrUserBanned
	}

	if userRole.Name == models.RoleLimited {
		cscount, err := s.repo.User.GetCreatedContestsCount(ctx, userID)
		if err != nil {
			return 0, fmt.Errorf("%s: can't get created contests count: %v", op, err)
		}

		if cscount >= int(userRole.CreatedContestsLimit) {
			return 0, service.ErrContestsLimitExceeded
		}
	}

	id, err = s.repo.Contest.CreateWithProblemIDs(ctx, userID, body.Title, body.Description, body.StartTime, body.EndTime, body.DurationMins, body.MaxEntries, body.AllowLateJoin, body.ProblemsIDs)
	if err != nil {
		return 0, fmt.Errorf("%s: can't create contest: %v", op, err)
	}

	return id, nil
}
