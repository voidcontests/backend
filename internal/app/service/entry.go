package service

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/voidcontests/api/internal/storage/models"
	"github.com/voidcontests/api/internal/storage/models/award"
	"github.com/voidcontests/api/internal/storage/repository"
	"github.com/voidcontests/api/pkg/ton"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
)

type EntryService struct {
	repo *repository.Repository
	ton  *ton.Client
}

func NewEntryService(repo *repository.Repository, tc *ton.Client) *EntryService {
	return &EntryService{
		repo: repo,
		ton:  tc,
	}
}

func (s *EntryService) CreateEntry(ctx context.Context, contestID int, userID int) error {
	op := "service.EntryService.CreateEntry"

	contest, err := s.repo.Contest.GetByID(ctx, contestID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrContestNotFound
	}
	if err != nil {
		return fmt.Errorf("%s: failed to get contest: %w", op, err)
	}

	entriesCount, err := s.repo.Contest.GetEntriesCount(ctx, contestID)
	if err != nil {
		return fmt.Errorf("%s: failed to get entries count: %w", op, err)
	}

	if contest.MaxEntries != 0 && entriesCount >= contest.MaxEntries {
		return ErrMaxSlotsReached
	}

	now := time.Now()
	if contest.EndTime.Before(now) || (contest.StartTime.Before(now) && !contest.AllowLateJoin) {
		return ErrApplicationTimeOver
	}

	_, err = s.repo.Entry.Get(ctx, contestID, userID)
	if err == nil {
		return ErrEntryAlreadyExists
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%s: failed to check existing entry: %w", op, err)
	}

	_, err = s.repo.Entry.Create(ctx, contestID, userID)
	if err != nil {
		return fmt.Errorf("%s: failed to create entry: %w", op, err)
	}

	return nil
}

type EntryDetails struct {
	Entry      models.Entry
	IsAdmitted bool
	Message    string
}

func (s *EntryService) GetEntry(ctx context.Context, contestID int, userID int) (EntryDetails, error) {
	op := "service.EntryService.GetEntry"

	entry, err := s.repo.Entry.Get(ctx, contestID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return EntryDetails{}, ErrEntryNotFound
	}
	if err != nil {
		return EntryDetails{}, fmt.Errorf("%s: failed to get entry: %w", op, err)
	}

	if entry.IsPaid {
		return EntryDetails{
			Entry:      entry,
			IsAdmitted: true,
		}, nil
	}

	contest, err := s.repo.Contest.GetByID(ctx, contestID)
	if err != nil {
		return EntryDetails{}, fmt.Errorf("%s: failed to get contest: %w", op, err)
	}

	if contest.AwardType != award.Pool {
		return EntryDetails{
			Entry:      entry,
			IsAdmitted: true,
		}, nil
	}

	user, err := s.repo.User.GetByID(ctx, userID)
	if err != nil {
		return EntryDetails{}, fmt.Errorf("%s: failed to get user: %w", op, err)
	}

	if user.Address == nil || *user.Address == "" {
		return EntryDetails{
			Entry:      entry,
			IsAdmitted: false,
			Message:    "connect wallet to your account, and pay entry price to contest's wallet",
		}, nil
	}

	// TODO: maybe this is a 5xx
	if contest.WalletID == nil {
		return EntryDetails{
			Entry:      entry,
			IsAdmitted: false,
			Message:    "contest has ho wallet associated",
		}, nil
	}

	wallet, err := s.repo.Contest.GetWallet(ctx, *contest.WalletID)
	if err != nil {
		return EntryDetails{}, fmt.Errorf("%s: failed to get wallet: %w", op, err)
	}

	from, err := address.ParseAddr(*user.Address)
	if err != nil {
		return EntryDetails{}, fmt.Errorf("%s: failed to parse sender address: %w", op, err)
	}

	to, err := address.ParseAddr(wallet.Address)
	if err != nil {
		return EntryDetails{}, fmt.Errorf("%s: failed to parse recepient address: %w", op, err)
	}

	amount := tlb.FromNanoTON(big.NewInt(int64(contest.EntryPriceTonNanos)))

	tx, exists := s.ton.LookupTx(ctx, from, to, amount)
	if !exists {
		return EntryDetails{
			Entry:      entry,
			IsAdmitted: false,
			Message:    "tx not found",
		}, nil
	}

	err = s.repo.Entry.MarkAsPaid(ctx, entry.ID, tx)
	if err != nil {
		return EntryDetails{}, fmt.Errorf("%s: failed to mark entry as paid: %w", op, err)
	}

	entry.IsPaid = true
	entry.TxHash = tx

	return EntryDetails{
		Entry:      entry,
		IsAdmitted: true,
	}, nil
}
