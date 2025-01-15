package entry

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/voidcontests/backend/internal/repository/models"
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
