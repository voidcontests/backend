package submission

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/voidcontests/backend/internal/repository/models"
)

const (
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

func (p *Postgres) Create(ctx context.Context, entryID int32, problemID int32, verdict string, answer string, code string, language string, passedTestsCount int32) (*models.Submission, error) {
	var err error
	var submission models.Submission

	query := `INSERT INTO submissions (entry_id, problem_id, verdict, answer, code, language, passed_tests_count) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *`
	err = p.db.GetContext(ctx, &submission, query, entryID, problemID, verdict, answer, code, language, passedTestsCount)
	if err != nil {
		return nil, err
	}

	return &submission, nil
}

func (p *Postgres) GetForProblem(ctx context.Context, entryID int32, charcode string) ([]models.Submission, error) {
	var err error
	var submissions []models.Submission

	query := `SELECT s.* FROM submissions s
	JOIN entries e ON s.entry_id = e.id
	JOIN contest_problems cp ON cp.contest_id = e.contest_id AND cp.problem_id = s.problem_id
	WHERE s.entry_id = $1 AND cp.charcode = $2`

	err = p.db.SelectContext(ctx, &submissions, query, entryID, charcode)
	if err != nil {
		return nil, err
	}

	return submissions, nil
}

func (p *Postgres) GetForEntry(ctx context.Context, entryID int32) ([]models.Submission, error) {
	var err error
	var submissions []models.Submission

	query := `SELECT * FROM submissions WHERE entry_id = $1`
	err = p.db.SelectContext(ctx, &submissions, query, entryID)
	if err != nil {
		return nil, err
	}

	return submissions, nil
}
