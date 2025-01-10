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
	Create(ctx context.Context, title string, description string, creatorAddress string, startingAt time.Time, durationMins int32, isDraft bool) (*entity.Contest, error)
	GetAll(ctx context.Context) ([]entity.Contest, error)
}

type Problem interface {
	Create(ctx context.Context, contestID int32, title string, statement string, difficulty string, writerAddress string, input string, answer string) (*entity.Problem, error)
	GetAll(ctx context.Context) ([]entity.Problem, error)
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
