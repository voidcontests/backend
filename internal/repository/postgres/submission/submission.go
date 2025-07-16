package submission

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/voidcontests/backend/internal/repository/models"
)

const (
	VerdictPending           = "pending"
	VerdictRunning           = "running"
	VerdictOK                = "ok"
	VerdictWrongAnswer       = "wrong_answer"
	VerdictRuntimeError      = "runtime_error"
	VerdictCompilationError  = "compilation_error"
	VerdictTimeLimitExceeded = "time_limit_exceeded"

	defaultLimit = 100
)

type Postgres struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool}
}

func (p *Postgres) Create(ctx context.Context, entryID, problemID int32, verdict, answer, code, language string, passedTestsCount int32, stderr string) (models.Submission, error) {
	query := `
		INSERT INTO submissions (entry_id, problem_id, verdict, answer, code, language, passed_tests_count, stderr)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, entry_id, problem_id,
		          (SELECT kind FROM problems WHERE id = $2) AS problem_kind,
		          verdict, answer, code, language, passed_tests_count, stderr, created_at
	`

	var submission models.Submission
	err := p.pool.QueryRow(ctx, query, entryID, problemID, verdict, answer, code, language, passedTestsCount, stderr).Scan(
		&submission.ID,
		&submission.EntryID,
		&submission.ProblemID,
		&submission.ProblemKind,
		&submission.Verdict,
		&submission.Answer,
		&submission.Code,
		&submission.Language,
		&submission.PassedTestsCount,
		&submission.Stderr,
		&submission.CreatedAt,
	)

	return submission, err
}

func (p *Postgres) CountTestsForProblem(ctx context.Context, problemID int32) (int32, error) {
	var count int32
	err := p.pool.QueryRow(ctx, `SELECT COUNT(*) FROM test_cases WHERE problem_id = $1`, problemID).Scan(&count)
	return count, err
}

func (p *Postgres) GetFailedTest(ctx context.Context, submissionID int32) (models.FailedTest, error) {
	query := `SELECT id, submission_id, input, expected_output, actual_output, created_at FROM failed_tests WHERE submission_id = $1`
	var ft models.FailedTest
	err := p.pool.QueryRow(ctx, query, submissionID).Scan(
		&ft.ID,
		&ft.SubmissionID,
		&ft.Input,
		&ft.ExpectedOutput,
		&ft.ActualOutput,
		&ft.CreatedAt,
	)
	return ft, err
}

func (p *Postgres) GetProblemStatus(ctx context.Context, entryID int32, problemID int32) (string, error) {
	query := `
		SELECT
			CASE
				WHEN COUNT(*) FILTER (WHERE s.verdict = 'ok') > 0 THEN 'accepted'
				WHEN COUNT(*) > 0 THEN 'tried'
				ELSE NULL
			END AS status
		FROM submissions s
		WHERE s.entry_id = $1 AND s.problem_id = $2
	`

	var status sql.NullString
	err := p.pool.QueryRow(ctx, query, entryID, problemID).Scan(&status)
	if err != nil {
		return "", fmt.Errorf("query failed: %w", err)
	}

	if status.Valid {
		return status.String, nil
	}
	return "", nil
}

func (p *Postgres) GetProblemStatuses(ctx context.Context, entryID int32) (map[int32]string, error) {
	query := `
		SELECT
			s.problem_id,
			CASE
				WHEN COUNT(*) FILTER (WHERE s.verdict = 'ok') > 0 THEN 'ok'
				WHEN COUNT(*) > 0 THEN 'tried'
				ELSE NULL
			END AS status
		FROM submissions s
		WHERE s.entry_id = $1
		GROUP BY s.problem_id
	`

	rows, err := p.pool.Query(ctx, query, entryID)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	statuses := make(map[int32]string)

	for rows.Next() {
		var problemID int32
		var status sql.NullString

		if err := rows.Scan(&problemID, &status); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		if status.Valid {
			statuses[problemID] = status.String
		} else {
			statuses[problemID] = "" // or "not_submitted"
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return statuses, nil
}

func (p *Postgres) GetByID(ctx context.Context, userID, submissionID int32) (models.Submission, error) {
	query := `
		SELECT s.id, s.entry_id, s.problem_id, p.kind AS problem_kind, s.verdict,
		       s.answer, s.code, s.language, s.passed_tests_count, s.stderr, s.created_at
		FROM submissions s
		JOIN problems p ON p.id = s.problem_id
		JOIN entries e ON s.entry_id = e.id
		JOIN users u ON e.user_id = u.id
		WHERE s.id = $1 AND u.id = $2
	`

	var s models.Submission
	err := p.pool.QueryRow(ctx, query, submissionID, userID).Scan(
		&s.ID,
		&s.EntryID,
		&s.ProblemID,
		&s.ProblemKind,
		&s.Verdict,
		&s.Answer,
		&s.Code,
		&s.Language,
		&s.PassedTestsCount,
		&s.Stderr,
		&s.CreatedAt,
	)

	return s, err
}

func (p *Postgres) ListByProblem(ctx context.Context, entryID int32, charcode string, limit int, offset int) (items []models.Submission, total int, err error) {
	if limit < 0 {
		limit = defaultLimit
	}

	batch := &pgx.Batch{}

	batch.Queue(`
		SELECT s.id, s.entry_id, s.problem_id, p.kind AS problem_kind, s.verdict,
		       s.answer, s.code, s.language, s.passed_tests_count, s.stderr, s.created_at
		FROM submissions s
		JOIN problems p ON p.id = s.problem_id
		JOIN entries e ON s.entry_id = e.id
		JOIN contest_problems cp ON cp.contest_id = e.contest_id AND cp.problem_id = s.problem_id
		WHERE s.entry_id = $1 AND cp.charcode = $2
		ORDER BY s.created_at DESC LIMIT $3 OFFSET $4
	`, entryID, charcode, limit, offset)

	batch.Queue(`
		SELECT COUNT(*)
		FROM submissions s
		JOIN entries e ON s.entry_id = e.id
		JOIN contest_problems cp ON cp.contest_id = e.contest_id AND cp.problem_id = s.problem_id
		WHERE s.entry_id = $1 AND cp.charcode = $2
	`, entryID, charcode)

	br := p.pool.SendBatch(ctx, batch)
	defer br.Close()

	rows, err := br.Query()
	if err != nil {
		return nil, 0, fmt.Errorf("query rows failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var s models.Submission
		if err := rows.Scan(
			&s.ID,
			&s.EntryID,
			&s.ProblemID,
			&s.ProblemKind,
			&s.Verdict,
			&s.Answer,
			&s.Code,
			&s.Language,
			&s.PassedTestsCount,
			&s.Stderr,
			&s.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("row scan failed: %w", err)
		}
		items = append(items, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("row iteration error: %w", err)
	}

	if err := br.QueryRow().Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count query failed: %w", err)
	}

	return items, total, nil
}
