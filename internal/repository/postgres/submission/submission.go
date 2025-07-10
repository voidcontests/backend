package submission

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/voidcontests/backend/internal/repository/models"
)

const (
	VerdictRunning           = "running"
	VerdictOK                = "ok"
	VerdictWrongAnswer       = "wrong_answer"
	VerdictRuntimeError      = "runtime_error"
	VerdictCompilationError  = "compilation_error"
	VerdictTimeLimitExceeded = "time_limit_exceeded"
)

type Postgres struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Postgres {
	return &Postgres{db}
}

func (p *Postgres) Create(ctx context.Context, entryID int32, problemID int32, verdict string, answer string, code string, language string, passedTestsCount int32, stderr string) (models.Submission, error) {
	var submission models.Submission
	query := `INSERT INTO submissions (entry_id, problem_id, verdict, answer, code, language, passed_tests_count, stderr) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING *`
	err := p.db.GetContext(ctx, &submission, query, entryID, problemID, verdict, answer, code, language, passedTestsCount, stderr)
	return submission, err
}

func (p *Postgres) UpdateVerdict(ctx context.Context, id int32, verdict string, passedTestsCount int32, stderr string) error {
	query := "UPDATE submissions SET verdict = $1, passed_tests_count = $2, stderr = $3 WHERE id = $4"
	_, err := p.db.ExecContext(ctx, query, verdict, passedTestsCount, stderr, id)
	return err
}

func (p *Postgres) GetTotalTestsCount(ctx context.Context, problemID int32) (int32, error) {
	query := "SELECT COUNT(*) FROM test_cases WHERE problem_id = $1"
	var count int32
	err := p.db.GetContext(ctx, &count, query, problemID)
	return count, err
}

func (p *Postgres) AddFailedTest(ctx context.Context, submissionID int32, input, expectedOutput, actualOutput, stderr string) error {
	query := "INSERT INTO failed_tests (submission_id, input, expected_output, actual_output) VALUES ($1, $2, $3, $4)"
	_, err := p.db.ExecContext(ctx, query, submissionID, input, expectedOutput, actualOutput)
	return err
}

func (p *Postgres) GetFailedTest(ctx context.Context, submissionID int32) (models.FailedTest, error) {
	query := "SELECT * FROM failed_tests WHERE submission_id = $1"
	var failedTest models.FailedTest
	err := p.db.GetContext(ctx, &failedTest, query, submissionID)
	return failedTest, err
}

func (p *Postgres) GetForProblem(ctx context.Context, entryID int32, charcode string) ([]models.Submission, error) {
	query := `SELECT s.* FROM submissions s
	JOIN entries e ON s.entry_id = e.id
	JOIN contest_problems cp ON cp.contest_id = e.contest_id AND cp.problem_id = s.problem_id
	WHERE s.entry_id = $1 AND cp.charcode = $2`

	var submissions []models.Submission
	err := p.db.SelectContext(ctx, &submissions, query, entryID, charcode)
	if err != nil {
		return nil, err
	}

	return submissions, nil
}

func (p *Postgres) GetForEntry(ctx context.Context, entryID int32) ([]models.Submission, error) {
	query := `SELECT * FROM submissions WHERE entry_id = $1`
	var submissions []models.Submission
	err := p.db.SelectContext(ctx, &submissions, query, entryID)
	if err != nil {
		return nil, err
	}

	return submissions, nil
}

func (p *Postgres) GetByID(ctx context.Context, userID int32, submissionID int32) (models.Submission, error) {
	query := `SELECT s.* FROM submissions s
		JOIN
		    entries e ON s.entry_id = e.id
		JOIN
		    users u ON e.user_id = u.id
		WHERE
		    s.id = $1 AND u.id = $2;`
	var submission models.Submission
	err := p.db.GetContext(ctx, &submission, query, submissionID, userID)
	return submission, err
}
