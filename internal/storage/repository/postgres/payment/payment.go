package payment

import (
	"context"
	"fmt"

	"github.com/voidcontests/api/internal/storage/models"
	"github.com/voidcontests/api/internal/storage/repository/postgres"
)

type Postgres struct {
	conn postgres.Transactor
}

func New(conn postgres.Transactor) *Postgres {
	return &Postgres{conn: conn}
}

func (p *Postgres) Create(ctx context.Context, txHash, fromAddress, toAddress string, amountTonNanos uint64, isIncoming bool) (int, error) {
	var paymentID int
	query := `INSERT INTO payments (tx_hash, from_address, to_address, amount_ton_nanos, is_incoming)
			  VALUES ($1, $2, $3, $4, $5) RETURNING id`
	err := p.conn.QueryRow(ctx, query, txHash, fromAddress, toAddress, amountTonNanos, isIncoming).Scan(&paymentID)
	if err != nil {
		return 0, fmt.Errorf("insert payment failed: %w", err)
	}

	return paymentID, nil
}

func (p *Postgres) GetByID(ctx context.Context, paymentID int) (models.Payment, error) {
	var payment models.Payment
	query := `SELECT id, tx_hash, from_address, to_address, amount_ton_nanos, is_incoming, created_at
			  FROM payments WHERE id = $1`
	err := p.conn.QueryRow(ctx, query, paymentID).Scan(
		&payment.ID,
		&payment.TxHash,
		&payment.FromAddress,
		&payment.ToAddress,
		&payment.AmountTonNanos,
		&payment.IsIncoming,
		&payment.CreatedAt,
	)
	if err != nil {
		return models.Payment{}, fmt.Errorf("get payment by id failed: %w", err)
	}

	return payment, nil
}

func (p *Postgres) GetByTxHash(ctx context.Context, txHash string) (models.Payment, error) {
	var payment models.Payment
	query := `SELECT id, tx_hash, from_address, to_address, amount_ton_nanos, is_incoming, created_at
			  FROM payments WHERE tx_hash = $1`
	err := p.conn.QueryRow(ctx, query, txHash).Scan(
		&payment.ID,
		&payment.TxHash,
		&payment.FromAddress,
		&payment.ToAddress,
		&payment.AmountTonNanos,
		&payment.IsIncoming,
		&payment.CreatedAt,
	)
	if err != nil {
		return models.Payment{}, fmt.Errorf("get payment by tx_hash failed: %w", err)
	}

	return payment, nil
}
