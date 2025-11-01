package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/voidcontests/api/internal/storage/models"
	"github.com/voidcontests/api/internal/storage/repository"
)

type ContestService struct {
	repo *repository.Repository
}

func NewContestService(repo *repository.Repository) *ContestService {
	return &ContestService{
		repo: repo,
	}
}

func (s *ContestService) CreateContest(ctx context.Context, userID int, title, description string, startTime, endTime time.Time, durationMins, maxEntries int, allowLateJoin bool, problemIDs []int) (int, error) {
	op := "service.ContestService.CreateContest"

	userRole, err := s.repo.User.GetRole(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("%s: failed to get user role: %w", op, err)
	}

	if userRole.Name == models.RoleBanned {
		return 0, ErrUserBanned
	}

	if userRole.Name == models.RoleLimited {
		contestsCount, err := s.repo.User.GetCreatedContestsCount(ctx, userID)
		if err != nil {
			return 0, fmt.Errorf("%s: failed to get created contests count: %w", op, err)
		}

		if contestsCount >= int(userRole.CreatedContestsLimit) {
			return 0, ErrContestsLimitExceeded
		}
	}

	contestID, err := s.repo.Contest.CreateWithProblemIDs(ctx, userID, title, description, startTime, endTime, durationMins, maxEntries, allowLateJoin, problemIDs)
	if err != nil {
		return 0, fmt.Errorf("%s: failed to create contest: %w", op, err)
	}

	return contestID, nil
}

type ContestDetails struct {
	Contest            models.Contest
	Problems           []models.Problem
	IsParticipant      bool
	SubmissionDeadline *time.Time
	ProblemStatuses    map[int]string
}

func (s *ContestService) GetContestByID(ctx context.Context, contestID int, userID int, authenticated bool) (*ContestDetails, error) {
	op := "service.ContestService.GetContestByID"

	contest, err := s.repo.Contest.GetByID(ctx, contestID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrContestNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get contest: %w", op, err)
	}

	now := time.Now()
	if contest.EndTime.Before(now) {
		if !authenticated || userID != contest.CreatorID {
			return nil, ErrContestFinished
		}
	}

	problems, err := s.repo.Contest.GetProblemset(ctx, contestID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get problemset: %w", op, err)
	}

	details := &ContestDetails{
		Contest:  contest,
		Problems: problems,
	}

	if !authenticated {
		return details, nil
	}

	entry, err := s.repo.Entry.Get(ctx, contestID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return details, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get entry: %w", op, err)
	}

	details.IsParticipant = true

	_, deadline := CalculateSubmissionWindow(contest, entry)
	if contest.StartTime.Before(now) {
		details.SubmissionDeadline = &deadline
	}

	statuses, err := s.repo.Submission.GetProblemStatuses(ctx, entry.ID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get problem statuses: %w", op, err)
	}
	details.ProblemStatuses = statuses

	return details, nil
}

type ListContestsResult struct {
	Contests []models.Contest
	Total    int
}

func (s *ContestService) ListCreatedContests(ctx context.Context, creatorID int, limit, offset int) (*ListContestsResult, error) {
	op := "service.ContestService.ListCreatedContests"

	contests, total, err := s.repo.Contest.GetWithCreatorID(ctx, creatorID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get created contests: %w", op, err)
	}

	return &ListContestsResult{
		Contests: contests,
		Total:    total,
	}, nil
}

func (s *ContestService) ListAllContests(ctx context.Context, limit, offset int) (*ListContestsResult, error) {
	op := "service.ContestService.ListAllContests"

	contests, total, err := s.repo.Contest.ListAll(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to list all contests: %w", op, err)
	}

	return &ListContestsResult{
		Contests: contests,
		Total:    total,
	}, nil
}

type LeaderboardResult struct {
	Leaderboard []models.LeaderboardEntry
	Total       int
}

func (s *ContestService) GetLeaderboard(ctx context.Context, contestID int, limit, offset int) (*LeaderboardResult, error) {
	op := "service.ContestService.GetLeaderboard"

	_, err := s.repo.Contest.GetByID(ctx, int(contestID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrContestNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get contest: %w", op, err)
	}

	leaderboard, total, err := s.repo.Contest.GetLeaderboard(ctx, contestID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get leaderboard: %w", op, err)
	}

	return &LeaderboardResult{
		Leaderboard: leaderboard,
		Total:       total,
	}, nil
}
