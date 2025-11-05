package problem

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/voidcontests/api/internal/storage/models"
	"github.com/voidcontests/api/internal/storage/repository/postgres"
)

type Postgres struct {
	conn postgres.Transactor
}

func New(conn postgres.Transactor) *Postgres {
	return &Postgres{conn: conn}
}

func (p *Postgres) Create(ctx context.Context, writerID int, title, statement, difficulty string, timeLimitMS, memoryLimitMB int, checker string) (int, error) {
	var problemID int
	err := p.conn.QueryRow(ctx, `
        INSERT INTO problems (writer_id, title, statement, difficulty, time_limit_ms, memory_limit_mb, checker)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING id
    `, writerID, title, statement, difficulty, timeLimitMS, memoryLimitMB, checker).Scan(&problemID)
	if err != nil {
		return 0, fmt.Errorf("insert problem failed: %w", err)
	}

	return problemID, nil
}

func (p *Postgres) AssociateTestCases(ctx context.Context, problemID int, tcs []models.TestCaseDTO) error {
	if len(tcs) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for i, tc := range tcs {
		batch.Queue(`
            INSERT INTO test_cases (problem_id, ordinal, input, output, is_example)
            VALUES ($1, $2, $3, $4, $5)
        `, problemID, i+1, tc.Input, tc.Output, tc.IsExample)
	}

	br := p.conn.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < batch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("insert test case %d (ordinal=%d, is_example=%v) failed: %w", i, i+1, tcs[i].IsExample, err)
		}
	}

	if err := br.Close(); err != nil {
		return fmt.Errorf("batch close failed: %w", err)
	}

	return nil
}

func (p *Postgres) Get(ctx context.Context, contestID int, charcode string) (models.Problem, error) {
	query := `
SELECT
	id, writer_id, title, statement, difficulty, time_limit_ms,
	memory_limit_mb, checker, created_at, charcode, writer_username
FROM contest_problemsets
WHERE contest_id = $1 AND charcode = $2`

	row := p.conn.QueryRow(ctx, query, contestID, charcode)

	var problem models.Problem
	err := row.Scan(
		&problem.ID, &problem.WriterID, &problem.Title, &problem.Statement,
		&problem.Difficulty, &problem.TimeLimitMS, &problem.MemoryLimitMB, &problem.Checker, &problem.CreatedAt,
		&problem.Charcode, &problem.WriterUsername,
	)

	return problem, err
}

func (p *Postgres) GetByID(ctx context.Context, problemID int) (models.Problem, error) {
	query := `SELECT
			id, writer_id, title, statement,
			difficulty, time_limit_ms, memory_limit_mb, checker, created_at,
			writer_username
		FROM problem_details
		WHERE id = $1`

	row := p.conn.QueryRow(ctx, query, problemID)

	var problem models.Problem
	err := row.Scan(
		&problem.ID, &problem.WriterID, &problem.Title, &problem.Statement,
		&problem.Difficulty, &problem.TimeLimitMS, &problem.MemoryLimitMB, &problem.Checker, &problem.CreatedAt,
		&problem.WriterUsername,
	)

	return problem, err
}

func (p *Postgres) GetExampleCases(ctx context.Context, problemID int) ([]models.TestCase, error) {
	query := `SELECT id, problem_id, ordinal, input, output, is_example FROM test_cases WHERE problem_id = $1 AND is_example = true`

	rows, err := p.conn.Query(ctx, query, problemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tcs := make([]models.TestCase, 0)
	for rows.Next() {
		var tc models.TestCase
		if err := rows.Scan(&tc.ID, &tc.ProblemID, &tc.Ordinal, &tc.Input, &tc.Output, &tc.IsExample); err != nil {
			return nil, err
		}
		tcs = append(tcs, tc)
	}

	return tcs, rows.Err()
}

func (p *Postgres) GetTestCaseByID(ctx context.Context, testCaseID int) (models.TestCase, error) {
	query := `SELECT id, problem_id, ordinal, input, output, is_example FROM test_cases WHERE id = $1`

	var tc models.TestCase
	err := p.conn.QueryRow(ctx, query, testCaseID).Scan(
		&tc.ID,
		&tc.ProblemID,
		&tc.Ordinal,
		&tc.Input,
		&tc.Output,
		&tc.IsExample,
	)

	if err != nil {
		return models.TestCase{}, fmt.Errorf("failed to get test case by ID: %w", err)
	}

	return tc, nil
}

func (p *Postgres) GetAll(ctx context.Context) ([]models.Problem, error) {
	query := `
SELECT
	id, writer_id, title, statement, difficulty, time_limit_ms,
	memory_limit_mb, checker, created_at, writer_username
FROM problem_details`

	rows, err := p.conn.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	problems := make([]models.Problem, 0)
	for rows.Next() {
		var p models.Problem
		if err := rows.Scan(
			&p.ID, &p.WriterID, &p.Title, &p.Statement, &p.Difficulty,
			&p.TimeLimitMS, &p.MemoryLimitMB, &p.Checker, &p.CreatedAt, &p.WriterUsername,
		); err != nil {
			return nil, err
		}
		problems = append(problems, p)
	}

	return problems, rows.Err()
}

func (p *Postgres) GetWithWriterID(ctx context.Context, writerID int, limit, offset int) (problems []models.Problem, total int, err error) {
	batch := &pgx.Batch{}

	batch.Queue(`
SELECT
	id, writer_id, title, statement, difficulty, time_limit_ms,
	memory_limit_mb, checker, created_at, writer_username
FROM problem_details
WHERE writer_id = $1
ORDER BY id ASC
LIMIT $2 OFFSET $3
	`, writerID, limit, offset)

	batch.Queue(`
		SELECT COUNT(*) FROM problem_details WHERE writer_id = $1
	`, writerID)

	br := p.conn.SendBatch(ctx, batch)

	rows, err := br.Query()
	if err != nil {
		br.Close()
		return nil, 0, fmt.Errorf("query failed: %w", err)
	}

	problems = make([]models.Problem, 0)
	for rows.Next() {
		var p models.Problem
		if err := rows.Scan(
			&p.ID, &p.WriterID, &p.Title, &p.Statement, &p.Difficulty,
			&p.TimeLimitMS, &p.MemoryLimitMB, &p.Checker, &p.CreatedAt, &p.WriterUsername,
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
