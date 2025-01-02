package contest

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/voidcontests/backend/internal/repository/entity"
)

type Postgres struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Postgres {
	return &Postgres{db}
}

func (p *Postgres) Create(ctx context.Context, title string, description string, problemIDs []int32, creatorAddress string, startTime time.Time, duration time.Duration, slots int32, isDraft bool) (*entity.Contest, error) {
	return nil, nil
}

func (p *Postgres) Get(ctx context.Context, contestID int32) (*entity.Contest, error) {
	return nil, nil
}

func (p *Postgres) PublishDraft(ctx context.Context, contestID int32) (*entity.Contest, error) {
	return nil, nil
}
