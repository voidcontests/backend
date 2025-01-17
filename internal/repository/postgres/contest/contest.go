package contest

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/voidcontests/backend/internal/repository/models"
	"github.com/voidcontests/backend/internal/repository/repoerr"
)

type Postgres struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Postgres {
	return &Postgres{db}
}

func (p *Postgres) Create(ctx context.Context, creatorID int32, title string, description string, startingAt time.Time, durationMins int32, isDraft bool) (*models.Contest, error) {
	var err error
	var contest models.Contest

	query := `INSERT INTO contests (creator_id, title, description, starting_at, duration_mins, is_draft) VALUES ($1, $2, $3, $4, $5, $6) RETURNING *`
	err = p.db.GetContext(ctx, &contest, query, creatorID, title, description, startingAt, durationMins, isDraft)
	if err != nil {
		return nil, err
	}

	return &contest, nil
}

func (p *Postgres) GetByID(ctx context.Context, contestID int32) (*models.Contest, error) {
	var err error
	var contest models.Contest

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

func (p *Postgres) GetProblemset(ctx context.Context, contestID int32) ([]models.Problem, error) {
	query := "SELECT * FROM problems WHERE contest_id = $1"

	var problems []models.Problem
	err := p.db.SelectContext(ctx, &problems, query, contestID)
	if errors.Is(err, sql.ErrNoRows) {
		return problems, nil
	}
	if err != nil {
		return nil, err
	}

	return problems, nil
}

func (p *Postgres) GetAll(ctx context.Context) ([]models.Contest, error) {
	var err error
	var contests []models.Contest

	query := `SELECT * FROM contests`
	err = p.db.SelectContext(ctx, &contests, query)
	if err != nil {
		return nil, err
	}

	return contests, nil
}

func (p *Postgres) IsTitleOccupied(ctx context.Context, title string) (bool, error) {
	var err error
	var count int

	query := `SELECT COUNT(*) FROM contests WHERE LOWER(title) = $1`
	err = p.db.QueryRowContext(ctx, query, strings.ToLower(title)).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}
