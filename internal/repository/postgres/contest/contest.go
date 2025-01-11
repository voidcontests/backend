package contest

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/voidcontests/backend/internal/repository/entity"
	repoerr "github.com/voidcontests/backend/internal/repository/errors"
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

func (p *Postgres) GetByID(ctx context.Context, contestID int32) (*entity.Contest, error) {
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

func (p *Postgres) GetProblemset(ctx context.Context, contestID int32) ([]entity.Problem, error) {
	query := "SELECT * FROM problems WHERE contest_id = $1"

	var problems []entity.Problem
	err := p.db.SelectContext(ctx, &problems, query, contestID)
	if errors.Is(err, sql.ErrNoRows) {
		return problems, nil
	}
	if err != nil {
		return nil, err
	}

	return problems, nil
}

func (p *Postgres) GetAll(ctx context.Context) ([]entity.Contest, error) {
	var err error
	var contests []entity.Contest

	query := `SELECT * FROM contests`
	err = p.db.SelectContext(ctx, &contests, query)
	if err != nil {
		return nil, err
	}

	return contests, nil
}
