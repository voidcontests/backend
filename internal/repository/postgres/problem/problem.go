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

func (p *Postgres) Create(ctx context.Context, contestID int32, title string, statement string, difficulty string, writerAddress string, input string, answer string) (*entity.Problem, error) {
	var err error
	var problem entity.Problem

	query := `INSERT INTO problems (contest_id, title, statement, difficulty, writer_address, input, answer) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *`
	err = p.db.GetContext(ctx, &problem, query, contestID, title, statement, difficulty, writerAddress, input, answer)
	if err != nil {
		return nil, err
	}

	return &problem, nil
}

func (p *Postgres) GetAll(ctx context.Context) ([]entity.Problem, error) {
	var err error
	var problems []entity.Problem

	query := `SELECT * FROM problems`
	err = p.db.SelectContext(ctx, &problems, query)
	if err != nil {
		return nil, err
	}

	return problems, nil
}
