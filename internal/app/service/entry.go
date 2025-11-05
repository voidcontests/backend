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

func (s *EntryService) GetEntry(ctx context.Context, contestID int, userID int) (models.Entry, error) {
	op := "service.EntryService.GetEntry"

	entry, err := s.repo.Entry.Get(ctx, contestID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Entry{}, ErrEntryNotFound
	}
	if err != nil {
		return models.Entry{}, fmt.Errorf("%s: failed to get entry: %w", op, err)
	}

	if entry.IsPaid {
		return entry, nil
	}

	contest, err := s.repo.Contest.GetByID(ctx, contestID)
	if err != nil {
		return models.Entry{}, fmt.Errorf("%s: failed to get contest: %w", op, err)
	}

	if contest.AwardType != award.Pool {
		return entry, nil
	}

	user, err := s.repo.User.GetByID(ctx, userID)
	if err != nil {
		return models.Entry{}, fmt.Errorf("%s: failed to get user: %w", op, err)
	}

	if user.Address == nil || *user.Address == "" {
		return entry, nil
	}

	if contest.WalletID == nil {
		return entry, nil
	}

	wallet, err := s.repo.Contest.GetWallet(ctx, *contest.WalletID)
	if err != nil {
		return models.Entry{}, fmt.Errorf("%s: failed to get wallet: %w", op, err)
	}

	from, err := address.ParseAddr(*user.Address)
	if err != nil {
		return models.Entry{}, fmt.Errorf("%s: failed to parse sender address: %w", op, err)
	}

	to, err := address.ParseAddr(wallet.Address)
	if err != nil {
		return models.Entry{}, fmt.Errorf("%s: failed to parse recepient address: %w", op, err)
	}

	// TODO: unhardcode, move to contest settings
	amount := tlb.FromNanoTON(big.NewInt(500000000))

	exists := s.ton.LookupTx(ctx, from, to, amount)

	if exists {
		err = s.repo.Entry.MarkAsPaid(ctx, entry.ID)
		if err != nil {
			return models.Entry{}, fmt.Errorf("%s: failed to mark entry as paid: %w", op, err)
		}
		entry.IsPaid = true
	}

	return entry, nil
}
