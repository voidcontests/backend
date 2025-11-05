package entry

import (
	"context"

	"github.com/voidcontests/api/internal/storage/models"
	"github.com/voidcontests/api/internal/storage/repository/postgres"
)

type Postgres struct {
	conn postgres.Transactor
}

func New(conn postgres.Transactor) *Postgres {
	return &Postgres{conn}
}

func (p *Postgres) Create(ctx context.Context, contestID int, userID int) (int, error) {
	query := `INSERT INTO entries (contest_id, user_id) VALUES ($1, $2) RETURNING id`

	var id int
	err := p.conn.QueryRow(ctx, query, contestID, userID).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (p *Postgres) Get(ctx context.Context, contestID int, userID int) (models.Entry, error) {
	query := `SELECT id, contest_id, user_id, is_paid, tx_hash, created_at FROM entries
	WHERE contest_id = $1 AND user_id = $2`

	var entry models.Entry
	err := p.conn.QueryRow(ctx, query, contestID, userID).Scan(
		&entry.ID,
		&entry.ContestID,
		&entry.UserID,
		&entry.IsPaid,
		&entry.TxHash,
		&entry.CreatedAt,
	)
	if err != nil {
		return models.Entry{}, err
	}
	return entry, nil
}

func (p *Postgres) MarkAsPaid(ctx context.Context, entryID int, txHash string) error {
	query := `UPDATE entries SET is_paid = true, tx_hash = $1 WHERE id = $2`
	_, err := p.conn.Exec(ctx, query, txHash, entryID)
	return err
}
