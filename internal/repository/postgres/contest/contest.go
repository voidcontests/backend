package contest

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

func (p *Postgres) Create(ctx context.Context, creatorID int32, title string, description string, startTime time.Time, endTime time.Time, durationMins int32, isDraft bool) (int32, error) {
	var id int32
	var err error

	query := `INSERT INTO contests (creator_id, title, description, start_time, end_time, duration_mins, is_draft) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`
	err = p.db.QueryRowContext(ctx, query, creatorID, title, description, startTime, endTime, durationMins, isDraft).Scan(&id)

	return id, err
}

func (p *Postgres) AddProblems(ctx context.Context, contestID int32, problemIDs ...int32) error {
	if len(problemIDs) == 0 {
		return nil
	}

	query := `INSERT INTO contest_problems (contest_id, problem_id) VALUES `
	values := make([]interface{}, 0, len(problemIDs)*2)
	placeholders := make([]string, 0, len(problemIDs))

	for i, problemID := range problemIDs {
		placeholders = append(placeholders, fmt.Sprintf("($%d, $%d)", i*2+1, i*2+2))
		values = append(values, contestID, problemID)
	}

	query += strings.Join(placeholders, ", ")

	_, err := p.db.ExecContext(ctx, query, values...)
	return err
}

func (p *Postgres) GetByID(ctx context.Context, contestID int32) (*models.Contest, error) {
	var err error
	var contest models.Contest

	query := `SELECT contests.*, users.address AS creator_address FROM contests JOIN users ON users.id = contests.creator_id WHERE contests.id = $1`
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
	var problems []models.Problem

	query := `SELECT p.*, u.address AS writer_address
		FROM problems p
		JOIN contest_problems cp ON p.id = cp.problem_id
		JOIN users u ON u.id = p.writer_id
		WHERE cp.contest_id = $1`

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

	query := `SELECT contests.*, users.address AS creator_address FROM contests JOIN users ON users.id = contests.creator_id`
	err = p.db.SelectContext(ctx, &contests, query)
	if err != nil {
		return nil, err
	}

	return contests, nil
}

func (p *Postgres) GetParticipantsCount(ctx context.Context, contestID int32) (int32, error) {
	var err error
	var count int32

	query := `SELECT COUNT(*) FROM entries WHERE contest_id = $1`
	err = p.db.QueryRowContext(ctx, query, contestID).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
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
