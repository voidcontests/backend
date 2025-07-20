package entry

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/voidcontests/backend/internal/app/service"
)

func (s *Service) CreateEntry(ctx context.Context, userID, contestID int32) (id int32, err error) {
	op := "service.CreateEntry"

	contest, err := s.repo.Contest.GetByID(ctx, contestID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, service.ErrContestNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("%s: can't get contest: %v", op, err)
	}

	entries, err := s.repo.Contest.GetEntriesCount(ctx, contestID)
	if err != nil {
		return 0, fmt.Errorf("%s: can't get entries: %v", op, err)
	}

	if contest.MaxEntries != 0 && entries >= contest.MaxEntries {
		return 0, service.ErrEntriesLimitReached
	}

	// NOTE: disallow join if: contest already finished or (already started and no late joins)
	if contest.EndTime.Before(time.Now()) || (contest.StartTime.Before(time.Now()) && !contest.AllowLateJoin) {
		return 0, service.ErrApplicationTimeIsOver
	}

	_, err = s.repo.Entry.Get(ctx, contestID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		id, err := s.repo.Entry.Create(ctx, contestID, userID)
		if err != nil {
			return 0, fmt.Errorf("%s: can't create entry: %v", op, err)
		}

		return id, nil
	}
	if err != nil {
		return 0, fmt.Errorf("%s: can't get entry: %v", op, err)
	}

	return 0, service.ErrEntryAlreadyExists
}
