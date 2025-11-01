package submission

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/voidcontests/api/internal/storage/models"
)

const (
	defaultLimit = 100
)

type Postgres struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool}
}

func (p *Postgres) Create(ctx context.Context, entryID int, problemID int, code string, language string) (models.Submission, error) {
	query := `INSERT INTO submissions (entry_id, problem_id, code, language)
		VALUES ($1, $2, $3, $4)
		RETURNING id, entry_id, problem_id, status, verdict, code, language, created_at`

	var submission models.Submission
	err := p.pool.QueryRow(ctx, query, entryID, problemID, code, language).Scan(
		&submission.ID,
		&submission.EntryID,
		&submission.ProblemID,
		&submission.Status,
		&submission.Verdict,
		&submission.Code,
		&submission.Language,
		&submission.CreatedAt,
	)

	return submission, err
}

func (p *Postgres) GetProblemStatus(ctx context.Context, entryID int, problemID int) (string, error) {
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

func (p *Postgres) GetProblemStatuses(ctx context.Context, entryID int) (map[int]string, error) {
	query := `
		SELECT
			s.problem_id,
			CASE
				WHEN COUNT(*) FILTER (WHERE s.verdict = 'ok') > 0 THEN 'accepted'
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

	statuses := make(map[int]string)

	for rows.Next() {
		var problemID int
		var status sql.NullString

		if err := rows.Scan(&problemID, &status); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		if status.Valid {
			statuses[problemID] = status.String
		} else {
			statuses[problemID] = ""
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return statuses, nil
}

func (p *Postgres) GetByID(ctx context.Context, submissionID int) (models.Submission, error) {
	query := `SELECT s.id, s.entry_id, s.problem_id, s.status, s.verdict, s.code, s.language, s.created_at
		FROM submissions s WHERE s.id = $1`

	var s models.Submission
	err := p.pool.QueryRow(ctx, query, submissionID).Scan(
		&s.ID,
		&s.EntryID,
		&s.ProblemID,
		&s.Status,
		&s.Verdict,
		&s.Code,
		&s.Language,
		&s.CreatedAt,
	)

	return s, err
}

func (p *Postgres) ListByProblem(ctx context.Context, entryID int, charcode string, limit int, offset int) (items []models.Submission, total int, err error) {
	if limit < 0 {
		limit = defaultLimit
	}

	query := `
		SELECT s.id, s.entry_id, s.problem_id, s.status, s.verdict, s.code, s.language, s.created_at, COUNT(*) OVER() as total_count
		FROM submissions s
		JOIN problems p ON p.id = s.problem_id
		JOIN entries e ON s.entry_id = e.id
		JOIN contest_problems cp ON cp.contest_id = e.contest_id AND cp.problem_id = s.problem_id
		WHERE s.entry_id = $1 AND cp.charcode = $2
		ORDER BY s.created_at DESC
		LIMIT $3 OFFSET $4`

	rows, err := p.pool.Query(ctx, query, entryID, charcode, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query rows failed: %w", err)
	}
	defer rows.Close()

	items = make([]models.Submission, 0)
	for rows.Next() {
		var s models.Submission
		if err := rows.Scan(
			&s.ID,
			&s.EntryID,
			&s.ProblemID,
			&s.Status,
			&s.Verdict,
			&s.Code,
			&s.Language,
			&s.CreatedAt,
			&total,
		); err != nil {
			return nil, 0, fmt.Errorf("row scan failed: %w", err)
		}
		items = append(items, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("row iteration error: %w", err)
	}

	return items, total, nil
}

func (p *Postgres) GetTestingReport(ctx context.Context, submissionID int) (models.TestingReport, error) {
	query := `SELECT id, submission_id, passed_tests_count, total_tests_count,
		first_failed_test_id, first_failed_test_output, stderr, created_at
		FROM testing_reports WHERE submission_id = $1`

	var report models.TestingReport
	err := p.pool.QueryRow(ctx, query, submissionID).Scan(
		&report.ID,
		&report.SubmissionID,
		&report.PassedTestsCount,
		&report.TotalTestsCount,
		&report.FirstFailedTestID,
		&report.FirstFailedTestOutput,
		&report.Stderr,
		&report.CreatedAt,
	)

	if err != nil {
		return models.TestingReport{}, err
	}

	return report, nil
}
