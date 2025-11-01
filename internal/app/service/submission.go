package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/voidcontests/api/internal/storage/broker"
	"github.com/voidcontests/api/internal/storage/models"
	"github.com/voidcontests/api/internal/storage/models/status"
	"github.com/voidcontests/api/internal/storage/repository"
)

type SubmissionService struct {
	repo   *repository.Repository
	broker broker.Broker
}

func NewSubmissionService(repo *repository.Repository, broker broker.Broker) *SubmissionService {
	return &SubmissionService{
		repo:   repo,
		broker: broker,
	}
}

type CreateSubmissionParams struct {
	ContestID int
	UserID    int
	Charcode  string
	Code      string
	Language  string
}

type CreateSubmissionResult struct {
	Submission models.Submission
}

func (s *SubmissionService) CreateSubmission(ctx context.Context, params CreateSubmissionParams) (*CreateSubmissionResult, error) {
	op := "service.SubmissionService.CreateSubmission"

	if len(params.Charcode) > 2 {
		return nil, ErrInvalidCharcode
	}
	charcode := strings.ToUpper(params.Charcode)

	contest, err := s.repo.Contest.GetByID(ctx, params.ContestID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrContestNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get contest: %w", op, err)
	}

	entry, err := s.repo.Entry.Get(ctx, params.ContestID, params.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoEntryForContest
	}
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get entry: %w", op, err)
	}

	now := time.Now()
	earliest, deadline := CalculateSubmissionWindow(contest, entry)
	if earliest.After(now) || deadline.Before(now) {
		return nil, ErrSubmissionWindowClosed
	}

	problem, err := s.repo.Problem.Get(ctx, params.ContestID, charcode)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrProblemNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get problem: %w", op, err)
	}

	submission, err := s.repo.Submission.Create(ctx, entry.ID, problem.ID, params.Code, params.Language)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to create submission: %w", op, err)
	}

	if err := s.broker.PublishSubmission(ctx, submission); err != nil {
		return nil, fmt.Errorf("%s: failed to publish submission: %w", op, err)
	}

	return &CreateSubmissionResult{
		Submission: submission,
	}, nil
}

type SubmissionDetails struct {
	Submission    models.Submission
	TestingReport *models.TestingReport
	FailedTest    *models.TestCase
}

func (s *SubmissionService) GetSubmissionByID(ctx context.Context, submissionID int) (*SubmissionDetails, error) {
	op := "service.SubmissionService.GetSubmissionByID"

	submission, err := s.repo.Submission.GetByID(ctx, submissionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSubmissionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get submission: %w", op, err)
	}

	details := &SubmissionDetails{
		Submission: submission,
	}

	if submission.Status != status.Success {
		return details, nil
	}

	testingReport, err := s.repo.Submission.GetTestingReport(ctx, submission.ID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get testing report: %w", op, err)
	}
	details.TestingReport = &testingReport

	if testingReport.FirstFailedTestID != nil {
		failedTest, err := s.repo.Problem.GetTestCaseByID(ctx, *testingReport.FirstFailedTestID)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to get failed test case: %w", op, err)
		}
		details.FailedTest = &failedTest
	}

	return details, nil
}

type ListSubmissionsResult struct {
	Submissions []models.Submission
	Total       int
}

func (s *SubmissionService) ListSubmissions(ctx context.Context, contestID int, userID int, charcode string, limit, offset int) (*ListSubmissionsResult, error) {
	op := "service.SubmissionService.ListSubmissions"

	if len(charcode) > 2 {
		return nil, ErrInvalidCharcode
	}
	charcode = strings.ToUpper(charcode)

	entry, err := s.repo.Entry.Get(ctx, contestID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoEntryForContest
	}
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get entry: %w", op, err)
	}

	submissions, total, err := s.repo.Submission.ListByProblem(ctx, entry.ID, charcode, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to list submissions: %w", op, err)
	}

	return &ListSubmissionsResult{
		Submissions: submissions,
		Total:       total,
	}, nil
}

func CalculateSubmissionWindow(contest models.Contest, entry models.Entry) (earliest time.Time, deadline time.Time) {
	if contest.DurationMins == 0 {
		return contest.StartTime, contest.EndTime
	}

	earliest = entry.CreatedAt
	if contest.StartTime.After(earliest) {
		earliest = contest.StartTime
	}

	personalDeadline := earliest.Add(time.Duration(contest.DurationMins) * time.Minute)

	if personalDeadline.Before(contest.EndTime) {
		deadline = personalDeadline
	} else {
		deadline = contest.EndTime
	}

	return earliest, deadline
}
