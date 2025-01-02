package problem

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/voidcontests/backend/internal/repository/entity"
)

type Postgres struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Postgres {
	return &Postgres{db}
}

func (p *Postgres) Create(ctx context.Context, title string, task string, writerAddress string, input string, answer string) (*entity.Problem, error) {
	return nil, nil
}

func (p *Postgres) Get(ctx context.Context, problemID int32) (*entity.Problem, error) {
	return nil, nil
}
