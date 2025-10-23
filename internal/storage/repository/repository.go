package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/voidcontests/api/internal/storage/models"
	"github.com/voidcontests/api/internal/storage/repository/postgres/contest"
	"github.com/voidcontests/api/internal/storage/repository/postgres/entry"
	"github.com/voidcontests/api/internal/storage/repository/postgres/problem"
	"github.com/voidcontests/api/internal/storage/repository/postgres/submission"
	"github.com/voidcontests/api/internal/storage/repository/postgres/user"
)

type Repository struct {
	User       User
	Contest    Contest
	Problem    Problem
	Entry      Entry
	Submission Submission
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{
		User:       user.New(pool),
		Contest:    contest.New(pool),
		Problem:    problem.New(pool),
		Entry:      entry.New(pool),
		Submission: submission.New(pool),
	}
}

type User interface {
	GetByCredentials(ctx context.Context, username string, passwordHash string) (models.User, error)
	Create(ctx context.Context, username string, passwordHash string) (models.User, error)
	Exists(ctx context.Context, username string) (bool, error)
	GetByID(ctx context.Context, id int32) (models.User, error)
	GetRole(ctx context.Context, userID int32) (models.Role, error)
	GetCreatedProblemsCount(ctx context.Context, userID int32) (int, error)
	GetCreatedContestsCount(ctx context.Context, userID int32) (int, error)
}

type Contest interface {
	Create(ctx context.Context, creatorID int32, title, description string, startTime, endTime time.Time, durationMins, maxEntries int32, allowLateJoin bool) (int32, error)
	CreateWithProblemIDs(ctx context.Context, creatorID int32, title, desc string, startTime, endTime time.Time, durationMins, maxEntries int32, allowLateJoin bool, problemIDs []int32) (int32, error)
	GetByID(ctx context.Context, contestID int32) (models.Contest, error)
	GetProblemset(ctx context.Context, contestID int32) ([]models.Problem, error)
	ListAll(ctx context.Context, limit int, offset int) (contests []models.Contest, total int, err error)
	GetWithCreatorID(ctx context.Context, creatorID int32, limit, offset int) (contests []models.Contest, total int, err error)
	GetEntriesCount(ctx context.Context, contestID int32) (int32, error)
	IsTitleOccupied(ctx context.Context, title string) (bool, error)
	GetLeaderboard(ctx context.Context, contestID, limit, offset int) (leaderboard []models.LeaderboardEntry, total int, err error)
}

type Problem interface {
	CreateWithTCs(ctx context.Context, kind string, writerID int32, title, statement, difficulty, answer string, timeLimitMS int, tcs []models.TestCaseDTO) (int32, error)
	Create(ctx context.Context, kind string, writerID int32, title, statement, difficulty, answer string, timeLimitMS int32) (int32, error)
	Get(ctx context.Context, contestID int32, charcode string) (models.Problem, error)
	GetByID(ctx context.Context, problemID int32) (models.Problem, error)
	GetExampleCases(ctx context.Context, problemID int32) ([]models.TestCase, error)
	GetTestCaseByID(ctx context.Context, testCaseID int32) (models.TestCase, error)
	GetAll(ctx context.Context) ([]models.Problem, error)
	GetWithWriterID(ctx context.Context, writerID int32, limit, offset int) (problems []models.Problem, total int, err error)
	IsTitleOccupied(ctx context.Context, title string) (bool, error)
}

type Entry interface {
	Create(ctx context.Context, contestID int32, userID int32) (int, error)
	Get(ctx context.Context, contestID int32, userID int32) (models.Entry, error)
}

type Submission interface {
	Create(ctx context.Context, entryID int32, problemID int32, code string, language string) (models.Submission, error)
	GetProblemStatus(ctx context.Context, entryID int32, problemID int32) (string, error)
	GetProblemStatuses(ctx context.Context, entryID int32) (map[int32]string, error)
	GetByID(ctx context.Context, submissionID int32) (models.Submission, error)
	ListByProblem(ctx context.Context, entryID int32, charcode string, limit int, offset int) (items []models.Submission, total int, err error)
	GetTestingReport(ctx context.Context, submissionID int32) (models.TestingReport, error)
}
