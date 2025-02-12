package problem

import (
	"context"
	"database/sql"
	"errors"
	"strings"

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

func (p *Postgres) Create(ctx context.Context, writerID int32, title string, statement string, difficulty string, input string, answer string) (int32, error) {
	var id int32
	var err error

	query := `INSERT INTO problems (writer_id, title, statement, difficulty, input, answer) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`
	err = p.db.QueryRowContext(ctx, query, writerID, title, statement, difficulty, input, answer).Scan(&id)

	return id, err
}

func (p *Postgres) Get(ctx context.Context, contestID int32, charcode string) (*models.Problem, error) {
	var problem models.Problem

	query := `SELECT p.*, cp.charcode, u.address AS writer_address
		FROM problems p
		JOIN contest_problems cp ON p.id = cp.problem_id
		JOIN users u ON u.id = p.writer_id
		WHERE cp.contest_id = $1 AND cp.charcode = $2`
	err := p.db.GetContext(ctx, &problem, query, contestID, charcode)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repoerr.ErrProblemNotFound
	}
	if err != nil {
		return nil, err
	}

	return &problem, nil
}

func (p *Postgres) GetAll(ctx context.Context) ([]models.Problem, error) {
	var err error
	var problems []models.Problem

	query := `SELECT problems.*, users.address AS writer_address FROM problems JOIN users ON users.id = problems.writer_id`
	err = p.db.SelectContext(ctx, &problems, query)
	if err != nil {
		return nil, err
	}

	return problems, nil
}

func (p *Postgres) IsTitleOccupied(ctx context.Context, title string) (bool, error) {
	var err error
	var count int

	query := `SELECT COUNT(*) FROM problems WHERE LOWER(title) = $1`
	err = p.db.QueryRowContext(ctx, query, strings.ToLower(title)).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}
