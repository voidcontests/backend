package wallet

import (
	"context"
	"fmt"

	"github.com/voidcontests/api/internal/storage/repository/postgres"
)

type Postgres struct {
	conn postgres.Transactor
}

func New(conn postgres.Transactor) *Postgres {
	return &Postgres{conn: conn}
}

func (p *Postgres) Create(ctx context.Context, address, mnemonic string) (int, error) {
	var walletID int
	query := `INSERT INTO wallets (address, mnemonic) VALUES ($1, $2) RETURNING id`
	err := p.conn.QueryRow(ctx, query, address, mnemonic).Scan(&walletID)
	if err != nil {
		return 0, fmt.Errorf("insert wallet failed: %w", err)
	}

	return walletID, nil
}
