package repository

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/voidcontests/backend/internal/repository/postgres/contest"
	"github.com/voidcontests/backend/internal/repository/postgres/entry"
	"github.com/voidcontests/backend/internal/repository/postgres/problem"
	"github.com/voidcontests/backend/internal/repository/postgres/submission"
	"github.com/voidcontests/backend/internal/repository/postgres/user"
)

type Repository struct {
	User       *user.Postgres
	Contest    *contest.Postgres
	Problem    *problem.Postgres
	Entry      *entry.Postgres
	Submission *submission.Postgres
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
