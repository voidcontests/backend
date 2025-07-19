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
		return 0, fmt.Errorf("tx begin failed: %w", err)
	}
	defer tx.Rollback(ctx)

	var problemID int32
	err = tx.QueryRow(ctx, `
        INSERT INTO problems (kind, writer_id, title, statement, difficulty, answer, time_limit_ms)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING id
    `, kind, writerID, title, statement, difficulty, answer, timeLimitMS).Scan(&problemID)
	if err != nil {
		return 0, fmt.Errorf("insert problem failed: %w", err)
	}

	if len(tcs) > 0 {
		batch := &pgx.Batch{}
		for _, tc := range tcs {
			batch.Queue(`
                INSERT INTO test_cases (problem_id, input, output, is_example)
                VALUES ($1, $2, $3, $4)
            `, problemID, tc.Input, tc.Output, tc.IsExample)
		}

		br := tx.SendBatch(ctx, batch)

		for i := 0; i < batch.Len(); i++ {
			if _, err := br.Exec(); err != nil {
				br.Close()
				return 0, fmt.Errorf("insert test case %d failed: %w", i, err)
			}
		}

		if err := br.Close(); err != nil {
			return 0, fmt.Errorf("batch close failed: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit failed: %w", err)
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

func (p *Postgres) Get(ctx context.Context, contestID int32, charcode string) (models.Problem, error) {
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

	return problem, err
}

func (p *Postgres) GetByID(ctx context.Context, problemID int32) (models.Problem, error) {
	query := `SELECT
			p.id, p.kind, p.writer_id, p.title, p.statement,
			p.difficulty, p.answer, p.time_limit_ms, p.created_at,
			u.username AS writer_username
		FROM problems p
		JOIN users u ON u.id = p.writer_id
		WHERE p.id = $1`

	row := p.pool.QueryRow(ctx, query, problemID)

	var problem models.Problem
	err := row.Scan(
		&problem.ID, &problem.Kind, &problem.WriterID, &problem.Title, &problem.Statement,
		&problem.Difficulty, &problem.Answer, &problem.TimeLimitMS, &problem.CreatedAt,
		&problem.WriterUsername,
	)

	return problem, err
}

func (p *Postgres) GetTestCases(ctx context.Context, problemID int32) ([]models.TestCase, error) {
	query := `SELECT id, problem_id, input, output, is_example FROM test_cases WHERE problem_id = $1`
	rows, err := p.pool.Query(ctx, query, problemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tcs := make([]models.TestCase, 0)
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

	tcs := make([]models.TestCase, 0)
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

	problems := make([]models.Problem, 0)
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

	rows, err := br.Query()
	if err != nil {
		br.Close()
		return nil, 0, fmt.Errorf("query failed: %w", err)
	}

	problems = make([]models.Problem, 0)
	for rows.Next() {
		var p models.Problem
		if err := rows.Scan(
			&p.ID, &p.Kind, &p.WriterID, &p.Title, &p.Statement, &p.Difficulty,
			&p.Answer, &p.TimeLimitMS, &p.CreatedAt, &p.WriterUsername,
		); err != nil {
			rows.Close()
			br.Close()
			return nil, 0, fmt.Errorf("scan failed: %w", err)
		}
		problems = append(problems, p)
	}
	rows.Close()

	if err := br.QueryRow().Scan(&total); err != nil {
		br.Close()
		return nil, 0, fmt.Errorf("count scan failed: %w", err)
	}

	if err := br.Close(); err != nil {
		return nil, 0, fmt.Errorf("batch close failed: %w", err)
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
