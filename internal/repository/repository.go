package repository

import (
	"github.com/jmoiron/sqlx"
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

func New(db *sqlx.DB) *Repository {
	return &Repository{
		User:       user.New(db),
		Contest:    contest.New(db),
		Problem:    problem.New(db),
		Entry:      entry.New(db),
		Submission: submission.New(db),
	}
}
