package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/voidcontests/api/internal/storage/models"
	"github.com/voidcontests/api/internal/storage/models/award"
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

// CreateProblemParams contains parameters for creating a problem
type CreateProblemParams struct {
	UserID        int
	Title         string
	Statement     string
	Difficulty    string
	TimeLimitMS   int
	MemoryLimitMB int
	Checker       string
	TestCases     []models.TestCaseDTO
}

func (s *ProblemService) CreateProblem(ctx context.Context, params CreateProblemParams) (int, error) {
	op := "service.ProblemService.CreateProblem"

	userRole, err := s.repo.User.GetRole(ctx, params.UserID)
	if err != nil {
		return 0, fmt.Errorf("%s: failed to get user role: %w", op, err)
	}

	if userRole.Name == models.RoleBanned {
		return 0, ErrUserBanned
	}

	if userRole.Name == models.RoleLimited {
		problemsCount, err := s.repo.User.GetCreatedProblemsCount(ctx, params.UserID)
		if err != nil {
			return 0, fmt.Errorf("%s: failed to get created problems count: %w", op, err)
		}

		if problemsCount >= userRole.CreatedProblemsLimit {
			return 0, ErrProblemsLimitExceeded
		}
	}

	if params.TimeLimitMS < 500 || params.TimeLimitMS > 10000 {
		return 0, ErrInvalidTimeLimit
	}

	if params.MemoryLimitMB < 16 || params.MemoryLimitMB > 512 {
		return 0, ErrInvalidMemoryLimit
	}

	examplesCount := 0
	for i := range params.TestCases {
		if params.TestCases[i].IsExample {
			examplesCount++
		}

		if examplesCount > 3 && params.TestCases[i].IsExample {
			params.TestCases[i].IsExample = false
		}
	}

	checker := params.Checker
	if checker == "" {
		checker = "tokens"
	}

	const MAX_TEST_CASES = 100
	if len(params.TestCases) > MAX_TEST_CASES {
		return 0, fmt.Errorf("too many test cases: got %d, max %d", len(params.TestCases), MAX_TEST_CASES)
	}

	for i, tc := range params.TestCases {
		if tc.Input == "" && tc.Output == "" {
			return 0, fmt.Errorf("test case %d: both input and output are empty", i)
		}
	}

	var problemID int
	err = s.repo.TxManager.WithinTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		repo := repository.NewTxRepository(tx)

		problemID, err = repo.Problem.Create(ctx, params.UserID, params.Title, params.Statement, params.Difficulty, params.TimeLimitMS, params.MemoryLimitMB, checker)
		if err != nil {
			return fmt.Errorf("create problem: %w", err)
		}

		err = repo.Problem.AssociateTestCases(ctx, problemID, params.TestCases)
		if err != nil {
			return fmt.Errorf("associate test cases: %w", err)
		}

		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("%s: failed to create problem: %w", op, err)
	}

	return problemID, nil
}

type ListProblemsResult struct {
	Problems []models.Problem
	Total    int
}

func (s *ProblemService) GetCreatedProblems(ctx context.Context, writerID int, limit, offset int) (*ListProblemsResult, error) {
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

func (s *ProblemService) GetContestProblem(ctx context.Context, contestID int, userID int, charcode string) (*ContestProblemDetails, error) {
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

	if contest.AwardType == award.Pool && !entry.IsPaid {
		return nil, ErrEntryNotPaid
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

func (s *ProblemService) GetProblemByID(ctx context.Context, problemID int, userID int) (*ProblemDetails, error) {
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
