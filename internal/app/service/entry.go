package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/voidcontests/api/internal/storage/repository"
)

type EntryService struct {
	repo *repository.Repository
}

func NewEntryService(repo *repository.Repository) *EntryService {
	return &EntryService{
		repo: repo,
	}
}

func (s *EntryService) CreateEntry(ctx context.Context, contestID int32, userID int32) error {
	op := "service.EntryService.CreateEntry"

	contest, err := s.repo.Contest.GetByID(ctx, contestID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrContestNotFound
	}
	if err != nil {
		return fmt.Errorf("%s: failed to get contest: %w", op, err)
	}

	entriesCount, err := s.repo.Contest.GetEntriesCount(ctx, contestID)
	if err != nil {
		return fmt.Errorf("%s: failed to get entries count: %w", op, err)
	}

	if contest.MaxEntries != 0 && entriesCount >= contest.MaxEntries {
		return ErrMaxSlotsReached
	}

	now := time.Now()
	if contest.EndTime.Before(now) || (contest.StartTime.Before(now) && !contest.AllowLateJoin) {
		return ErrApplicationTimeOver
	}

	_, err = s.repo.Entry.Get(ctx, contestID, userID)
	if err == nil {
		return ErrEntryAlreadyExists
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%s: failed to check existing entry: %w", op, err)
	}

	_, err = s.repo.Entry.Create(ctx, contestID, userID)
	if err != nil {
		return fmt.Errorf("%s: failed to create entry: %w", op, err)
	}

	return nil
}
