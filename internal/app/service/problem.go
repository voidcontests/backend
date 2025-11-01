package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/voidcontests/api/internal/storage/models"
	"github.com/voidcontests/api/internal/storage/repository"
)

type ProblemService struct {
	repo *repository.Repository
}

func NewProblemService(repo *repository.Repository) *ProblemService {
	return &ProblemService{
		repo: repo,
	}
}

func (s *ProblemService) CreateProblem(ctx context.Context, userID int32, title, statement, difficulty string, timeLimitMS, memoryLimitMB int, checker string, testCases []models.TestCaseDTO) (int32, error) {
	op := "service.ProblemService.CreateProblem"

	userRole, err := s.repo.User.GetRole(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("%s: failed to get user role: %w", op, err)
	}

	if userRole.Name == models.RoleBanned {
		return 0, ErrUserBanned
	}

	if userRole.Name == models.RoleLimited {
		problemsCount, err := s.repo.User.GetCreatedProblemsCount(ctx, userID)
		if err != nil {
			return 0, fmt.Errorf("%s: failed to get created problems count: %w", op, err)
		}

		if problemsCount >= int(userRole.CreatedProblemsLimit) {
			return 0, ErrProblemsLimitExceeded
		}
	}

	if timeLimitMS < 500 || timeLimitMS > 10000 {
		return 0, ErrInvalidTimeLimit
	}

	if memoryLimitMB < 16 || memoryLimitMB > 512 {
		return 0, ErrInvalidMemoryLimit
	}

	examplesCount := 0
	for i := range testCases {
		if testCases[i].IsExample {
			examplesCount++
		}

		if examplesCount > 3 && testCases[i].IsExample {
			testCases[i].IsExample = false
		}
	}

	if checker == "" {
		checker = "tokens"
	}

	problemID, err := s.repo.Problem.CreateWithTCs(ctx, userID, title, statement, difficulty, timeLimitMS, memoryLimitMB, checker, testCases)
	if err != nil {
		return 0, fmt.Errorf("%s: failed to create problem: %w", op, err)
	}

	return problemID, nil
}

type ListProblemsResult struct {
	Problems []models.Problem
	Total    int
}

func (s *ProblemService) GetCreatedProblems(ctx context.Context, writerID int32, limit, offset int) (*ListProblemsResult, error) {
	op := "service.ProblemService.GetCreatedProblems"

	problems, total, err := s.repo.Problem.GetWithWriterID(ctx, writerID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get created problems: %w", op, err)
	}

	return &ListProblemsResult{
		Problems: problems,
		Total:    total,
	}, nil
}

type ContestProblemDetails struct {
	Problem          models.Problem
	Examples         []models.TestCase
	Status           string
	SubmissionWindow SubmissionWindow
}

type SubmissionWindow struct {
	Earliest time.Time
	Deadline time.Time
}

func (s *ProblemService) GetContestProblem(ctx context.Context, contestID int32, userID int32, charcode string) (*ContestProblemDetails, error) {
	op := "service.ProblemService.GetContestProblem"

	if len(charcode) > 2 {
		return nil, ErrInvalidCharcode
	}
	charcode = strings.ToUpper(charcode)

	contest, err := s.repo.Contest.GetByID(ctx, contestID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrContestNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get contest: %w", op, err)
	}

	now := time.Now()
	if contest.StartTime.After(now) {
		return nil, ErrContestNotStarted
	}

	entry, err := s.repo.Entry.Get(ctx, contestID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoEntryForContest
	}
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get entry: %w", op, err)
	}

	problem, err := s.repo.Problem.Get(ctx, contestID, charcode)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrProblemNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get problem: %w", op, err)
	}

	examples, err := s.repo.Problem.GetExampleCases(ctx, problem.ID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get example cases: %w", op, err)
	}

	status, err := s.repo.Submission.GetProblemStatus(ctx, entry.ID, problem.ID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get problem status: %w", op, err)
	}

	earliest, deadline := CalculateSubmissionWindow(contest, entry)

	return &ContestProblemDetails{
		Problem:  problem,
		Examples: examples,
		Status:   status,
		SubmissionWindow: SubmissionWindow{
			Earliest: earliest,
			Deadline: deadline,
		},
	}, nil
}

type ProblemDetails struct {
	Problem  models.Problem
	Examples []models.TestCase
}

func (s *ProblemService) GetProblemByID(ctx context.Context, problemID int32, userID int32) (*ProblemDetails, error) {
	op := "service.ProblemService.GetProblemByID"

	problem, err := s.repo.Problem.GetByID(ctx, problemID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrProblemNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get problem: %w", op, err)
	}

	if problem.WriterID != userID {
		return nil, ErrNotProblemWriter
	}

	examples, err := s.repo.Problem.GetExampleCases(ctx, problem.ID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get example cases: %w", op, err)
	}

	return &ProblemDetails{
		Problem:  problem,
		Examples: examples,
	}, nil
}
