package problem

import (
	"context"
	"database/sql"
	"errors"

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

func (p *Postgres) Create(ctx context.Context, contestID int32, writerID int32, title string, statement string, difficulty string, input string, answer string) (int32, error) {
	var id int32
	var err error

	query := `INSERT INTO problems (contest_id, writer_id, title, statement, difficulty, input, answer) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`
	err = p.db.QueryRowContext(ctx, query, contestID, writerID, title, statement, difficulty, input, answer).Scan(&id)

	return id, err
}

func (p *Postgres) GetAnswer(ctx context.Context, id int32) (string, error) {
	var err error
	var answer string

	query := `SELECT answer FROM problems WHERE id = $1`
	err = p.db.GetContext(ctx, &answer, query, id)
	if err != nil {
		return "", err
	}

	return answer, nil
}

func (p *Postgres) Get(ctx context.Context, id int32) (*models.Problem, error) {
	var err error
	var problem models.Problem

	query := `SELECT problems.*, users.address AS creator_address FROM problems JOIN users ON users.id = problems.writer_id WHERE problems.id = $1`
	err = p.db.GetContext(ctx, &problem, query, id)
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

	query := `SELECT problems.*, users.address AS creator_address FROM problems JOIN users ON users.id = problems.writer_id`
	err = p.db.SelectContext(ctx, &problems, query)
	if err != nil {
		return nil, err
	}

	return problems, nil
}
