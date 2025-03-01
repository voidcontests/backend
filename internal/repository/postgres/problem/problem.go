package problem

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/voidcontests/backend/internal/app/handler/dto/request"
	"github.com/voidcontests/backend/internal/repository/models"
	"github.com/voidcontests/backend/internal/repository/repoerr"
)

type Postgres struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Postgres {
	return &Postgres{db}
}

func (p *Postgres) CreateWithTCs(ctx context.Context, kind string, writerID int32, title string, statement string, difficulty string, input string, answer string, timeLimitMS int32, tcs []request.TC) (int32, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}

	defer tx.Rollback()

	var problemID int32

	query := `INSERT INTO problems (kind, writer_id, title, statement, difficulty, input, answer, time_limit_ms)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`
	if err := tx.QueryRowContext(ctx, query, kind, writerID, title, statement, difficulty, input, answer, timeLimitMS).Scan(&problemID); err != nil {
		return 0, err
	}

	if len(tcs) > 0 {
		query = `INSERT INTO test_cases (problem_id, input, output) VALUES `
		values := make([]interface{}, 0, len(tcs)*3)
		placeholders := make([]string, 0, len(tcs))

		for i, tc := range tcs {
			placeholders = append(placeholders, fmt.Sprintf("($%d, $%d, $%d)", i*3+1, i*3+2, i*3+3))
			values = append(values, problemID, tc.Input, tc.Output)
		}
		query += strings.Join(placeholders, ", ")

		if _, err := tx.ExecContext(ctx, query, values...); err != nil {
			return 0, err
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return problemID, nil
}

func (p *Postgres) Create(ctx context.Context, kind string, writerID int32, title string, statement string, difficulty string, input string, answer string, timeLimitMS int32) (int32, error) {
	var id int32
	var err error

	query := `INSERT INTO problems (kind, writer_id, title, statement, difficulty, input, answer, time_limit_ms) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`
	err = p.db.QueryRowContext(ctx, query, kind, writerID, title, statement, difficulty, input, answer, timeLimitMS).Scan(&id)

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

func (p *Postgres) GetWithWriterID(ctx context.Context, writerID int32) ([]models.Problem, error) {
	var err error
	var problems []models.Problem

	query := `SELECT problems.*, users.address AS writer_address FROM problems JOIN users ON users.id = problems.writer_id WHERE writer_id = $1`
	err = p.db.SelectContext(ctx, &problems, query, writerID)
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
