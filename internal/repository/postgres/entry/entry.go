package entry

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/voidcontests/backend/internal/repository/models"
	"github.com/voidcontests/backend/internal/repository/repoerr"
)

type Postgres struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Postgres {
	return &Postgres{db}
}

func (p *Postgres) Create(ctx context.Context, contestID int32, userID int32) (*models.Entry, error) {
	var err error
	var entry models.Entry

	query := `INSERT INTO entries (contest_id, user_id) VALUES ($1, $2) RETURNING *`
	err = p.db.GetContext(ctx, &entry, query, contestID, userID)
	if err != nil {
		return nil, err
	}

	return &entry, nil
}

func (p *Postgres) Get(ctx context.Context, contestID int32, userID int32) (*models.Entry, error) {
	var err error
	var entry models.Entry

	query := `SELECT * FROM entries WHERE contest_id = $1 AND user_id = $2`
	err = p.db.GetContext(ctx, &entry, query, contestID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repoerr.ErrEntryNotFound
	}
	if err != nil {
		return nil, err
	}

	return &entry, nil
}
