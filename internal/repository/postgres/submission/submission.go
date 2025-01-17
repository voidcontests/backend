package submission

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/voidcontests/backend/internal/repository/models"
)

type Verdict string

const (
	VerdictOK          Verdict = "OK"
	VerdictWrongAnswer Verdict = "WA"
)

type Postgres struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Postgres {
	return &Postgres{db}
}

// entry_id INTEGER NOT NULL REFERENCES entries(id),
// problem_id INTEGER NOT NULL REFERENCES problems(id),
// verdict VARCHAR(10) NOT NULL,
// answer TEXT NOT NULL,

func (p *Postgres) Create(ctx context.Context, entryID int32, problemID int32, verdict Verdict, answer string) (*models.Submission, error) {
	var err error
	var submission models.Submission

	query := `INSERT INTO submissions (entry_id, problem_id, verdict, answer) VALUES ($1, $2, $3, $4) RETURNING *`
	err = p.db.GetContext(ctx, &submission, query, entryID, problemID, verdict, answer)
	if err != nil {
		return nil, err
	}

	return &submission, nil
}

func (p *Postgres) GetForProblem(ctx context.Context, userID int32, entryID int32, problemID int32) ([]models.Submission, error) {
	var err error
	var submissions []models.Submission

	query := `SELECT * FROM submissions WHERE user_id = $1 AND entry_id = $2 AND problem_id = $3`
	err = p.db.SelectContext(ctx, &submissions, query, userID, entryID, problemID)
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
