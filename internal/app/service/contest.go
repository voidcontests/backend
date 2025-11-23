package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/voidcontests/api/internal/lib/crypto"
	"github.com/voidcontests/api/internal/storage/models"
	"github.com/voidcontests/api/internal/storage/models/award"
	"github.com/voidcontests/api/internal/storage/repository"
	"github.com/voidcontests/api/pkg/ton"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
)

type ContestService struct {
	repo   *repository.Repository
	ton    *ton.Client
	cipher crypto.Cipher
}

func NewContestService(repo *repository.Repository, tc *ton.Client, cipher crypto.Cipher) *ContestService {
	return &ContestService{
		repo:   repo,
		ton:    tc,
		cipher: cipher,
	}
}

// CreateContestParams contains parameters for creating a contest
type CreateContestParams struct {
	UserID             int
	Title              string
	Description        string
	AwardType          string
	EntryPriceTonNanos uint64
	StartTime          time.Time
	EndTime            time.Time
	DurationMins       int
	MaxEntries         int
	AllowLateJoin      bool
	ProblemIDs         []int
}

func (s *ContestService) CreateContest(ctx context.Context, params CreateContestParams) (int, error) {
	op := "service.ContestService.CreateContest"

	now := time.Now()
	if params.StartTime.Before(now) {
		return 0, ErrInvalidContestTiming
	}
	if !params.StartTime.Before(params.EndTime) {
		return 0, ErrInvalidContestTiming
	}
	if params.DurationMins < 0 {
		return 0, ErrInvalidContestTiming
	}

	contestLengthMins := int(params.EndTime.Sub(params.StartTime).Minutes())
	if params.DurationMins > 0 && params.DurationMins > contestLengthMins {
		return 0, ErrInvalidContestTiming
	}

	userRole, err := s.repo.User.GetRole(ctx, params.UserID)
	if err != nil {
		return 0, fmt.Errorf("%s: failed to get user role: %w", op, err)
	}

	if userRole.Name == models.RoleBanned {
		return 0, ErrUserBanned
	}

	if userRole.Name == models.RoleLimited {
		contestsCount, err := s.repo.User.GetCreatedContestsCount(ctx, params.UserID)
		if err != nil {
			return 0, fmt.Errorf("%s: failed to get created contests count: %w", op, err)
		}

		if contestsCount >= userRole.CreatedContestsLimit {
			return 0, ErrContestsLimitExceeded
		}
	}

	const charcodes = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	if len(params.ProblemIDs) > len(charcodes) {
		return 0, fmt.Errorf("too many problems: got %d, max %d", len(params.ProblemIDs), len(charcodes))
	}

	problems := make([]models.ProblemCharcode, len(params.ProblemIDs))
	for i, problemID := range params.ProblemIDs {
		problems[i] = models.ProblemCharcode{
			ProblemID: problemID,
			Charcode:  string(charcodes[i]),
		}
	}

	// NOTE: if award type is not `paid_entry` or `sponsored` - use `no_prize` by default
	var contestID int
	switch params.AwardType {
	case award.Sponsored, award.Pool:
		w, err := s.ton.CreateWallet()
		if err != nil {
			return 0, err
		}

		address := w.Address().String()
		mnemonic := strings.Join(w.Mnemonic, " ")

		// Encrypt the mnemonic before storing
		encryptedMnemonic, err := s.cipher.Encrypt(mnemonic)
		if err != nil {
			return 0, fmt.Errorf("%s: failed to encrypt mnemonic: %w", op, err)
		}

		err = s.repo.TxManager.WithinTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
			repo := repository.NewTxRepository(tx)

			walletID, err := repo.Wallet.Create(ctx, address, encryptedMnemonic)
			if err != nil {
				return fmt.Errorf("create wallet: %w", err)
			}

			contestID, err = repo.Contest.Create(
				ctx,
				params.UserID,
				params.Title,
				params.Description,
				params.AwardType,
				params.EntryPriceTonNanos,
				params.StartTime,
				params.EndTime,
				params.DurationMins,
				params.MaxEntries,
				params.AllowLateJoin,
				problems,
				&walletID,
			)
			if err != nil {
				return fmt.Errorf("create contest: %w", err)
			}

			return nil
		})
		if err != nil {
			return 0, fmt.Errorf("%s: failed to create contest: %w", op, err)
		}
	case award.No:
		contestID, err = s.repo.Contest.Create(
			ctx,
			params.UserID,
			params.Title,
			params.Description,
			award.No,
			0,
			params.StartTime,
			params.EndTime,
			params.DurationMins,
			params.MaxEntries,
			params.AllowLateJoin,
			problems,
			nil,
		)
		if err != nil {
			return 0, fmt.Errorf("%s: failed to create contest: %w", op, err)
		}
	default:
		return 0, ErrUnknownAwardType
	}

	return contestID, nil
}

type ContestDetails struct {
	Contest             models.Contest
	IsRegistrationOpen  bool
	Problems            []models.Problem
	IsParticipant       bool
	ProblemStatuses     map[int]string
	WalletAddress       string
	PrizeNanosTON       uint64
	EntryDetails        *EntryDetails
	DistributionPayment *models.Payment
}

func (s *ContestService) GetContestByID(ctx context.Context, contestID int, userID int, authenticated bool) (*ContestDetails, error) {
	op := "service.ContestService.GetContestByID"

	contest, err := s.repo.Contest.GetByID(ctx, contestID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrContestNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get contest: %w", op, err)
	}

	problems, err := s.repo.Contest.GetProblemset(ctx, contestID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get problemset: %w", op, err)
	}

	now := time.Now()

	// Registration is closed if:
	//   1. Contest has ended
	//   2. Contest has started and late join is not allowed
	isRegistrationOpen := true
	if contest.EndTime.Before(now) || (contest.StartTime.Before(now) && !contest.AllowLateJoin) {
		isRegistrationOpen = false
	}
	details := &ContestDetails{
		IsRegistrationOpen: isRegistrationOpen,
		Contest:            contest,
		Problems:           problems,
	}

	if contest.WalletID != nil && (contest.AwardType == award.Pool || contest.AwardType == award.Sponsored) {
		wallet, err := s.repo.Contest.GetWallet(ctx, *contest.WalletID)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to get wallet: %w", op, err)
		}

		details.WalletAddress = wallet.Address

		addr, err := address.ParseAddr(wallet.Address)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to parse wallet address: %w", op, err)
		}

		details.PrizeNanosTON, err = s.ton.GetBalanceCached(ctx, addr)
		if err != nil {
			// TODO: maybe on this error, just return balance = 0 (?)
			return nil, fmt.Errorf("%s: failed to get wallet balance: %w", op, err)
		}
	}

	if contest.DistributionPaymentID != nil {
		payment, err := s.repo.Payment.GetByID(ctx, *contest.DistributionPaymentID)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to get distribution payment: %w", op, err)
		}
		details.DistributionPayment = &payment
	}

	if !authenticated {
		return details, nil
	}

	entry, err := s.repo.Entry.Get(ctx, contestID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return details, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get entry: %w", op, err)
	}

	details.IsParticipant = true

	statuses, err := s.repo.Submission.GetProblemStatuses(ctx, entry.ID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get problem statuses: %w", op, err)
	}
	details.ProblemStatuses = statuses

	entryDetails, err := s.getEntryDetails(ctx, entry, contest)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get entry details: %w", op, err)
	}
	details.EntryDetails = &entryDetails

	if details.EntryDetails != nil {
		_, deadline := CalculateSubmissionWindow(contest, entry)
		if contest.StartTime.Before(now) {
			details.EntryDetails.SubmissionDeadline = &deadline
		}
	}

	return details, nil
}

type ListContestsResult struct {
	Contests []models.Contest
	Total    int
}

func (s *ContestService) ListCreatedContests(ctx context.Context, creatorID int, limit, offset int) (*ListContestsResult, error) {
	op := "service.ContestService.ListCreatedContests"

	contests, total, err := s.repo.Contest.GetWithCreatorID(ctx, creatorID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get created contests: %w", op, err)
	}

	return &ListContestsResult{
		Contests: contests,
		Total:    total,
	}, nil
}

func (s *ContestService) ListAllContests(ctx context.Context, limit, offset int, filters models.ContestFilters) (*ListContestsResult, error) {
	op := "service.ContestService.ListAllContests"

	contests, total, err := s.repo.Contest.ListAll(ctx, limit, offset, filters)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to list all contests: %w", op, err)
	}

	return &ListContestsResult{
		Contests: contests,
		Total:    total,
	}, nil
}

type ScoresResult struct {
	Scores []models.ScoresEntry
	Total  int
}

func (s *ContestService) GetScores(ctx context.Context, contestID int, limit, offset int) (*ScoresResult, error) {
	op := "service.ContestService.GetScores"

	_, err := s.repo.Contest.GetByID(ctx, contestID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrContestNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get contest: %w", op, err)
	}

	scores, total, err := s.repo.Contest.GetScores(ctx, contestID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get scores: %w", op, err)
	}

	return &ScoresResult{
		Scores: scores,
		Total:  total,
	}, nil
}

type EntryDetails struct {
	Entry              models.Entry
	IsAdmitted         bool
	SubmissionDeadline *time.Time
	Message            string
	Payment            *models.Payment
}

func (s *ContestService) getEntryDetails(ctx context.Context, entry models.Entry, contest models.Contest) (EntryDetails, error) {
	op := "service.ContestService.getEntryDetails"

	if entry.PaymentID != nil {
		payment, err := s.repo.Payment.GetByID(ctx, *entry.PaymentID)
		if err != nil {
			return EntryDetails{}, fmt.Errorf("%s: failed to get payment: %w", op, err)
		}

		return EntryDetails{
			Entry:      entry,
			IsAdmitted: true,
			Payment:    &payment,
		}, nil
	}

	if contest.AwardType != award.Pool {
		return EntryDetails{
			Entry:      entry,
			IsAdmitted: true,
		}, nil
	}

	user, err := s.repo.User.GetByID(ctx, entry.UserID)
	if err != nil {
		return EntryDetails{}, fmt.Errorf("%s: failed to get user: %w", op, err)
	}

	if user.Address == "" {
		return EntryDetails{
			Entry:      entry,
			IsAdmitted: false,
			Message:    "address connected required to user account, to check payment",
		}, nil
	}

	from, err := address.ParseAddr(user.Address)
	if err != nil {
		return EntryDetails{}, fmt.Errorf("%s: failed to parse user address: %w", op, err)
	}

	if contest.WalletID == nil {
		return EntryDetails{}, errors.New("prized contest has no wallet")
	}

	wallet, err := s.repo.Contest.GetWallet(ctx, *contest.WalletID)
	if err != nil {
		return EntryDetails{}, fmt.Errorf("%s: failed to get wallet: %w", op, err)
	}

	to, err := address.ParseAddr(wallet.Address)
	if err != nil {
		return EntryDetails{}, fmt.Errorf("%s: failed to parse user address: %w", op, err)
	}

	amount := tlb.FromNanoTONU(contest.EntryPriceTonNanos)
	transaction, exists := s.ton.LookupTx(ctx, from, to, amount)
	if !exists {
		return EntryDetails{
			Entry:      entry,
			IsAdmitted: false,
			Message:    "payment required to participate in this contest",
		}, nil
	}

	var payment models.Payment
	err = s.repo.TxManager.WithinTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		repo := repository.NewTxRepository(tx)

		pid, err := repo.Payment.Create(ctx, transaction, s.ton.GetAddress(from), wallet.Address, contest.EntryPriceTonNanos, true)
		if err != nil {
			return fmt.Errorf("failed to create payment: %w", err)
		}

		err = repo.Entry.SetPaymentID(ctx, entry.ID, pid)
		if err != nil {
			return fmt.Errorf("failed to set payment ID for entry: %w", err)
		}
		entry.PaymentID = &pid

		payment, err = repo.Payment.GetByID(ctx, pid)
		if err != nil {
			return fmt.Errorf("failed to get payment by ID: %w", err)
		}

		return nil
	})

	if err != nil {
		return EntryDetails{}, fmt.Errorf("%s: %w", op, err)
	}

	return EntryDetails{
		Entry:      entry,
		IsAdmitted: true,
		Payment:    &payment,
	}, nil
}

func (s *ContestService) CreateEntry(ctx context.Context, contestID int, userID int) (int, error) {
	op := "service.ContestService.CreateEntry"

	contest, err := s.repo.Contest.GetByID(ctx, contestID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrContestNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("%s: failed to get contest: %w", op, err)
	}

	if contest.CreatorID == userID {
		return 0, ErrCannotJoinOwnContest
	}

	now := time.Now()
	if contest.EndTime.Before(now) || (contest.StartTime.Before(now) && !contest.AllowLateJoin) {
		return 0, ErrApplicationTimeOver
	}

	var entryID int
	err = s.repo.TxManager.WithinTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		repo := repository.NewTxRepository(tx)

		_, err := repo.Entry.Get(ctx, contestID, userID)
		if err == nil {
			return ErrEntryAlreadyExists
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("failed to check existing entry: %w", err)
		}

		entriesCount, err := repo.Contest.GetEntriesCount(ctx, contestID)
		if err != nil {
			return fmt.Errorf("failed to get entries count: %w", err)
		}

		if contest.MaxEntries != 0 && entriesCount >= contest.MaxEntries {
			return ErrMaxSlotsReached
		}

		entryID, err = repo.Entry.Create(ctx, contestID, userID)
		if err != nil {
			return fmt.Errorf("failed to create entry: %w", err)
		}

		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return entryID, nil
}
