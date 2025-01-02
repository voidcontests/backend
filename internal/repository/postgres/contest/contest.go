package contest

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/voidcontests/backend/internal/repository/entity"
	repoerr "github.com/voidcontests/backend/internal/repository/errors"
)

type Postgres struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Postgres {
	return &Postgres{db}
}

func (p *Postgres) Create(ctx context.Context, title string, description string, problemIDs []int32, creatorAddress string, startTime time.Time, duration time.Duration, slots int32, isDraft bool) (*entity.Contest, error) {
	var err error
	var contest entity.Contest

	query := `INSERT INTO contests (title, description, problem_ids, creator_address, start_time, duration, slots, is_draft) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING *`
	err = p.db.GetContext(ctx, &contest, query, title, description, pq.Array(problemIDs), creatorAddress, startTime, duration, slots, isDraft)
	if err != nil {
		return nil, err
	}

	return &contest, nil
}

func (p *Postgres) Get(ctx context.Context, contestID int32) (*entity.Contest, error) {
	var err error
	var contest entity.Contest

	query := `SELECT * FROM contests WHERE id = $1`

	err = p.db.GetContext(ctx, &contest, query, contestID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repoerr.ErrContestNotFound
	}
	if err != nil {
		return nil, err
	}

	return &contest, nil
}

func (p *Postgres) PublishDraft(ctx context.Context, contestID int32) (*entity.Contest, error) {
	return nil, nil
}
