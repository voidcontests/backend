package distributor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/big"

	"github.com/jackc/pgx/v5"
	"github.com/voidcontests/api/internal/lib/crypto"
	"github.com/voidcontests/api/internal/storage/models"
	"github.com/voidcontests/api/internal/storage/repository"
	"github.com/voidcontests/api/pkg/ton"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
)

func New(r *repository.Repository, tc *ton.Client, cipher crypto.Cipher) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		contests, err := r.Contest.GetWithUndistributedAwards(ctx)
		if err != nil {
			return err
		}

		for _, c := range contests {
			err := distributeAwardForContest(ctx, r, tc, cipher, c)
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func distributeAwardForContest(ctx context.Context, r *repository.Repository, tc *ton.Client, cipher crypto.Cipher, c models.Contest) error {
	w, err := r.Contest.GetWallet(ctx, *c.WalletID)
	if err != nil {
		return err
	}

	decryptedMnemonic, err := cipher.Decrypt(w.MnemonicEncrypted)
	if err != nil {
		return fmt.Errorf("failed to decrypt mnemonic: %w", err)
	}

	wallet, err := tc.WalletWithSeed(decryptedMnemonic)
	if err != nil {
		return err
	}

	winnerID, err := r.Contest.GetWinnerID(ctx, c.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		slog.Info("no users that submitted at least one ok solution", slog.Int("contest_id", c.ID))
		return nil
	}
	if err != nil {
		return err
	}

	winner, err := r.User.GetByID(ctx, winnerID)
	if err != nil {
		return err
	}

	if winner.Address == "" {
		return fmt.Errorf("user has no wallet")
	}

	recipient, err := address.ParseAddr(winner.Address)
	if err != nil {
		return err
	}

	nanos, err := tc.GetBalance(ctx, wallet.Address())
	if err != nil {
		return err
	}

	// keep 2% for paying gas
	factor := 1 - 0.02
	amount := tlb.FromNanoTON(big.NewInt(int64(float64(nanos) * factor)))
	tx, err := wallet.TransferTo(ctx, recipient, amount, fmt.Sprintf("contests.fckn.engineer: Prize for winning contest #%d", c.ID))
	if err != nil {
		return err
	}

	slog.Info("award distributed", slog.Int("contest_id", c.ID), slog.String("tx", tx))

	paymentID, err := r.Payment.Create(ctx, tx, tc.GetAddress(wallet.Address()), tc.GetAddress(recipient), amount.Nano().Uint64(), false)
	if err != nil {
		return err
	}

	err = r.Contest.SetDistributionPaymentID(ctx, c.ID, paymentID)
	if err != nil {
		return err
	}

	return nil
}
