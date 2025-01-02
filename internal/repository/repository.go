package repository

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/voidcontests/backend/internal/repository/entity"
	"github.com/voidcontests/backend/internal/repository/postgres/contest"
	"github.com/voidcontests/backend/internal/repository/postgres/problem"
)

type Contest interface {
	Create(ctx context.Context, title string, description string, problemIDs []int32, creatorAddress string, startTime time.Time, duration time.Duration, slots int32, isDraft bool) (*entity.Contest, error)
	Get(ctx context.Context, contestID int32) (*entity.Contest, error)
	PublishDraft(ctx context.Context, contestID int32) (*entity.Contest, error)
}

type Problem interface {
	Create(ctx context.Context, title string, task string, writerAddress string, input string, answer string) (*entity.Problem, error)
	Get(ctx context.Context, problemID int32) (*entity.Problem, error)
}

type Repository struct {
	Contest
	Problem
}

func New(db *sqlx.DB) *Repository {
	return &Repository{
		Contest: contest.New(db),
		Problem: problem.New(db),
	}
}
