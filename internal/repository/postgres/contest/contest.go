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

func (p *Postgres) Create(ctx context.Context, title string, description string, creatorAddress string, startingAt time.Time, durationMins int32, isDraft bool) (*entity.Contest, error) {
	var err error
	var contest entity.Contest

	query := `INSERT INTO contests (title, description, creator_address, starting_at, duration_mins, is_draft) VALUES ($1, $2, $3, $4, $5, $6) RETURNING *`
	err = p.db.GetContext(ctx, &contest, query, title, description, creatorAddress, startingAt, durationMins, isDraft)
	if err != nil {
		return nil, err
	}

	return &contest, nil
}
