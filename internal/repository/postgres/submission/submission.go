package submission

import (
	"context"

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
)

type Postgres struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool}
}

func (p *Postgres) Create(ctx context.Context, entryID int32, problemID int32, verdict string, answer string, code string, language string, passedTestsCount int32, stderr string) (models.Submission, error) {
	var submission models.Submission

	query := `
		INSERT INTO submissions (entry_id, problem_id, verdict, answer, code, language, passed_tests_count, stderr)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, entry_id, problem_id, (SELECT kind FROM problems WHERE id = $2) AS problem_kind, verdict, answer, code, language, passed_tests_count, stderr, created_at
	`

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

func (p *Postgres) GetTotalTestsCount(ctx context.Context, problemID int32) (int32, error) {
	var count int32
	query := `SELECT COUNT(*) FROM test_cases WHERE problem_id = $1`
	err := p.pool.QueryRow(ctx, query, problemID).Scan(&count)
	return count, err
}

func (p *Postgres) GetFailedTest(ctx context.Context, submissionID int32) (models.FailedTest, error) {
	var failedTest models.FailedTest

	query := `SELECT id, submission_id, input, expected_output, actual_output, created_at FROM failed_tests WHERE submission_id = $1`
	err := p.pool.QueryRow(ctx, query, submissionID).Scan(
		&failedTest.ID,
		&failedTest.SubmissionID,
		&failedTest.Input,
		&failedTest.ExpectedOutput,
		&failedTest.ActualOutput,
		&failedTest.CreatedAt,
	)

	return failedTest, err
}

func (p *Postgres) GetForProblem(ctx context.Context, entryID int32, charcode string) ([]models.Submission, error) {
	query := `
		SELECT s.id, s.entry_id, s.problem_id, p.kind AS problem_kind, s.verdict, s.answer, s.code, s.language, s.passed_tests_count, s.stderr, s.created_at
		FROM submissions s
		JOIN problems p ON p.id = s.problem_id
		JOIN entries e ON s.entry_id = e.id
		JOIN contest_problems cp ON cp.contest_id = e.contest_id AND cp.problem_id = s.problem_id
		WHERE s.entry_id = $1 AND cp.charcode = $2
	`

	rows, err := p.pool.Query(ctx, query, entryID, charcode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var submissions []models.Submission
	for rows.Next() {
		var s models.Submission
		err := rows.Scan(
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
		if err != nil {
			return nil, err
		}
		submissions = append(submissions, s)
	}

	return submissions, nil
}

func (p *Postgres) GetForEntry(ctx context.Context, entryID int32) ([]models.Submission, error) {
	query := `
		SELECT s.id, s.entry_id, s.problem_id, p.kind AS problem_kind, s.verdict, s.answer, s.code, s.language, s.passed_tests_count, s.stderr, s.created_at
		FROM submissions s
		JOIN problems p ON p.id = s.problem_id
		WHERE s.entry_id = $1
	`

	rows, err := p.pool.Query(ctx, query, entryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var submissions []models.Submission
	for rows.Next() {
		var s models.Submission
		err := rows.Scan(
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
		if err != nil {
			return nil, err
		}
		submissions = append(submissions, s)
	}

	return submissions, nil
}

func (p *Postgres) GetByID(ctx context.Context, userID int32, submissionID int32) (models.Submission, error) {
	var submission models.Submission

	query := `
		SELECT s.id, s.entry_id, s.problem_id, p.kind AS problem_kind, s.verdict, s.answer, s.code, s.language, s.passed_tests_count, s.stderr, s.created_at
		FROM submissions s
		JOIN problems p ON p.id = s.problem_id
		JOIN entries e ON s.entry_id = e.id
		JOIN users u ON e.user_id = u.id
		WHERE s.id = $1 AND u.id = $2
	`

	err := p.pool.QueryRow(ctx, query, submissionID, userID).Scan(
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
