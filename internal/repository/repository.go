package repository

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/voidcontests/backend/internal/repository/models"
	"github.com/voidcontests/backend/internal/repository/postgres/contest"
	"github.com/voidcontests/backend/internal/repository/postgres/entry"
	"github.com/voidcontests/backend/internal/repository/postgres/problem"
	"github.com/voidcontests/backend/internal/repository/postgres/submission"
	"github.com/voidcontests/backend/internal/repository/postgres/user"
)

type User interface {
	Create(ctx context.Context, address string) (*models.User, error)
	GetByAddress(ctx context.Context, address string) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
}

type Contest interface {
	Create(ctx context.Context, creatorID int32, title string, description string, startTime time.Time, endTime time.Time, durationMins int32, isDraft bool) (int32, error)
	GetProblemset(ctx context.Context, contestID int32) ([]models.Problem, error)
	GetByID(ctx context.Context, id int32) (*models.Contest, error)
	IsTitleOccupied(ctx context.Context, title string) (bool, error)
	GetParticipantsCount(ctx context.Context, contestID int32) (int32, error)
	GetAll(ctx context.Context) ([]models.Contest, error)
}

type Problem interface {
	Create(ctx context.Context, contestID int32, writerID int32, title string, statement string, difficulty string, input string, answer string) (int32, error)
	GetAnswer(ctx context.Context, id int32) (string, error)
	Get(ctx context.Context, id int32) (*models.Problem, error)
	GetAll(ctx context.Context) ([]models.Problem, error)
}

type Entry interface {
	Create(ctx context.Context, contestID int32, userID int32) (*models.Entry, error)
	Get(ctx context.Context, contestID int32, userID int32) (*models.Entry, error)
}

type Submission interface {
	Create(ctx context.Context, entryID int32, problemID int32, verdict string, answer string) (*models.Submission, error)
	GetForProblem(ctx context.Context, entryID int32, problemID int32) ([]models.Submission, error)
	GetForEntry(ctx context.Context, entryID int32) ([]models.Submission, error)
}

type Repository struct {
	User
	Contest
	Problem
	Entry
	Submission
}

func New(db *sqlx.DB) *Repository {
	return &Repository{
		User:       user.New(db),
		Contest:    contest.New(db),
		Problem:    problem.New(db),
		Entry:      entry.New(db),
		Submission: submission.New(db),
	}
}
