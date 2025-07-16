package problem

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/voidcontests/backend/internal/app/handler/dto/request"
	"github.com/voidcontests/backend/internal/repository/models"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool}
}

func (p *Postgres) CreateWithTCs(ctx context.Context, kind string, writerID int32, title, statement, difficulty, answer string, timeLimitMS int, tcs []request.TC) (int32, error) {
	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	var problemID int32
	query := `INSERT INTO problems (kind, writer_id, title, statement, difficulty, answer, time_limit_ms)
			VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`

	err = tx.QueryRow(ctx, query, kind, writerID, title, statement, difficulty, answer, timeLimitMS).Scan(&problemID)
	if err != nil {
		return 0, err
	}

	if len(tcs) > 0 {
		batch := &pgx.Batch{}
		for _, tc := range tcs {
			batch.Queue(
				`INSERT INTO test_cases (problem_id, input, output, is_example)
				 VALUES ($1, $2, $3, $4)`,
				problemID, tc.Input, tc.Output, tc.IsExample,
			)
		}

		br := tx.SendBatch(ctx, batch)
		defer br.Close()

		for i := 0; i < len(tcs); i++ {
			if _, err := br.Exec(); err != nil {
				return 0, fmt.Errorf("failed to insert test case %d: %w", i, err)
			}
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return 0, err
	}

	return problemID, nil
}

func (p *Postgres) Create(ctx context.Context, kind string, writerID int32, title, statement, difficulty, answer string, timeLimitMS int32) (int32, error) {
	var id int32
	query := `INSERT INTO problems (kind, writer_id, title, statement, difficulty, answer, time_limit_ms)
	          VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`

	err := p.pool.QueryRow(ctx, query, kind, writerID, title, statement, difficulty, answer, timeLimitMS).Scan(&id)
	return id, err
}

func (p *Postgres) Get(ctx context.Context, contestID int32, charcode string) (*models.Problem, error) {
	query := `SELECT p.*, cp.charcode, u.username AS writer_username
		FROM problems p
		JOIN contest_problems cp ON p.id = cp.problem_id
		JOIN users u ON u.id = p.writer_id
		WHERE cp.contest_id = $1 AND cp.charcode = $2`

	row := p.pool.QueryRow(ctx, query, contestID, charcode)

	var problem models.Problem
	err := row.Scan(
		&problem.ID, &problem.Kind, &problem.WriterID, &problem.Title, &problem.Statement,
		&problem.Difficulty, &problem.Answer, &problem.TimeLimitMS, &problem.CreatedAt,
		&problem.Charcode, &problem.WriterUsername,
	)
	if err != nil {
		return nil, err
	}

	return &problem, nil
}

func (p *Postgres) GetTestCases(ctx context.Context, problemID int32) ([]models.TestCase, error) {
	query := `SELECT id, problem_id, input, output, is_example FROM test_cases WHERE problem_id = $1`
	rows, err := p.pool.Query(ctx, query, problemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tcs []models.TestCase
	for rows.Next() {
		var tc models.TestCase
		if err := rows.Scan(&tc.ID, &tc.ProblemID, &tc.Input, &tc.Output, &tc.IsExample); err != nil {
			return nil, err
		}
		tcs = append(tcs, tc)
	}

	return tcs, nil
}

func (p *Postgres) GetExampleCases(ctx context.Context, problemID int32) ([]models.TestCase, error) {
	query := `SELECT * FROM test_cases WHERE problem_id = $1 AND is_example = true`

	rows, err := p.pool.Query(ctx, query, problemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tcs []models.TestCase
	for rows.Next() {
		var tc models.TestCase
		if err := rows.Scan(&tc.ID, &tc.ProblemID, &tc.Input, &tc.Output, &tc.IsExample); err != nil {
			return nil, err
		}
		tcs = append(tcs, tc)
	}

	return tcs, rows.Err()
}

func (p *Postgres) GetAll(ctx context.Context) ([]models.Problem, error) {
	query := `SELECT problems.*, users.username AS writer_username FROM problems JOIN users ON users.id = problems.writer_id`

	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var problems []models.Problem
	for rows.Next() {
		var p models.Problem
		if err := rows.Scan(
			&p.ID, &p.Kind, &p.WriterID, &p.Title, &p.Statement, &p.Difficulty,
			&p.Answer, &p.TimeLimitMS, &p.CreatedAt, &p.WriterUsername,
		); err != nil {
			return nil, err
		}
		problems = append(problems, p)
	}

	return problems, rows.Err()
}

func (p *Postgres) GetWithWriterID(ctx context.Context, writerID int32, limit, offset int) (problems []models.Problem, total int, err error) {
	batch := &pgx.Batch{}

	batch.Queue(`
		SELECT problems.*, users.username AS writer_username
		FROM problems
		JOIN users ON users.id = problems.writer_id
		WHERE writer_id = $1
		ORDER BY problems.id ASC
		LIMIT $2 OFFSET $3
	`, writerID, limit, offset)

	batch.Queue(`
		SELECT COUNT(*) FROM problems WHERE writer_id = $1
	`, writerID)

	br := p.pool.SendBatch(ctx, batch)
	defer br.Close()

	rows, err := br.Query()
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var p models.Problem
		if err := rows.Scan(
			&p.ID, &p.Kind, &p.WriterID, &p.Title, &p.Statement, &p.Difficulty,
			&p.Answer, &p.TimeLimitMS, &p.CreatedAt, &p.WriterUsername,
		); err != nil {
			return nil, 0, err
		}
		problems = append(problems, p)
	}

	if err := br.QueryRow().Scan(&total); err != nil {
		return nil, 0, err
	}

	return problems, total, nil
}

func (p *Postgres) IsTitleOccupied(ctx context.Context, title string) (bool, error) {
	query := `SELECT COUNT(*) FROM problems WHERE LOWER(title) = $1`

	var count int
	err := p.pool.QueryRow(ctx, query, strings.ToLower(title)).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}
