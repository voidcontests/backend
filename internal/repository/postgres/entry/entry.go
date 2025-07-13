package entry

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/voidcontests/backend/internal/repository/models"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool}
}

func (p *Postgres) Create(ctx context.Context, contestID int32, userID int32) (int, error) {
	query := `INSERT INTO entries (contest_id, user_id) VALUES ($1, $2) RETURNING id`

	var id int
	err := p.pool.QueryRow(ctx, query, contestID, userID).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (p *Postgres) Get(ctx context.Context, contestID int32, userID int32) (models.Entry, error) {
	query := `SELECT id, contest_id, user_id, created_at FROM entries
	WHERE contest_id = $1 AND user_id = $2`

	var entry models.Entry
	err := p.pool.QueryRow(ctx, query, contestID, userID).Scan(
		&entry.ID,
		&entry.ContestID,
		&entry.UserID,
		&entry.CreatedAt,
	)
	if err != nil {
		return models.Entry{}, err
	}
	return entry, nil
}
