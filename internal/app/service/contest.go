package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/voidcontests/api/internal/storage/models"
	"github.com/voidcontests/api/internal/storage/models/award"
	"github.com/voidcontests/api/internal/storage/repository"
	"github.com/voidcontests/api/pkg/ton"
	"github.com/xssnick/tonutils-go/address"
)

type ContestService struct {
	repo *repository.Repository
	ton  *ton.Client
}

func NewContestService(repo *repository.Repository, tc *ton.Client) *ContestService {
	return &ContestService{
		repo: repo,
		ton:  tc,
	}
}

// CreateContestParams contains parameters for creating a contest
type CreateContestParams struct {
	UserID        int
	Title         string
	Description   string
	AwardType     string
	StartTime     time.Time
	EndTime       time.Time
	DurationMins  int
	MaxEntries    int
	AllowLateJoin bool
	ProblemIDs    []int
}

func (s *ContestService) CreateContest(ctx context.Context, params CreateContestParams) (int, error) {
	op := "service.ContestService.CreateContest"

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
	if params.AwardType == award.Pool || params.AwardType == award.Sponsored {
		w, err := s.ton.CreateWallet()
		if err != nil {
			return 0, err
		}

		address := w.Address.String()
		mnemonic := strings.Join(w.Mnemonic, " ")

		err = s.repo.TxManager.WithinTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
			repo := repository.NewTxRepository(tx)

			walletID, err := repo.Wallet.Create(ctx, address, mnemonic)
			if err != nil {
				return fmt.Errorf("create wallet: %w", err)
			}

			contestID, err = repo.Contest.Create(
				ctx,
				params.UserID,
				params.Title,
				params.Description,
				params.AwardType,
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
	} else {
		contestID, err = s.repo.Contest.Create(
			ctx,
			params.UserID,
			params.Title,
			params.Description,
			award.No,
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
	}

	return contestID, nil
}

type ContestDetails struct {
	Contest            models.Contest
	Problems           []models.Problem
	IsParticipant      bool
	SubmissionDeadline *time.Time
	ProblemStatuses    map[int]string
	PrizeNanosTON      uint64
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

	now := time.Now()
	if contest.EndTime.Before(now) {
		if !authenticated || userID != contest.CreatorID {
			return nil, ErrContestFinished
		}
	}

	problems, err := s.repo.Contest.GetProblemset(ctx, contestID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get problemset: %w", op, err)
	}

	details := &ContestDetails{
		Contest:  contest,
		Problems: problems,
	}

	if contest.WalletID != nil && (contest.AwardType == award.Pool || contest.AwardType == award.Sponsored) {
		wallet, err := s.repo.Contest.GetWallet(ctx, *contest.WalletID)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to get wallet: %w", op, err)
		}

		addr, err := address.ParseAddr(wallet.Address)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to parse wallet address: %w", op, err)
		}

		details.PrizeNanosTON, err = s.ton.GetBalance(ctx, addr)
		if err != nil {
			// TODO: maybe on this error, just return balance = 0 (?)
			return nil, fmt.Errorf("%s: failed to get wallet balance: %w", op, err)
		}
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

	_, deadline := CalculateSubmissionWindow(contest, entry)
	if contest.StartTime.Before(now) {
		details.SubmissionDeadline = &deadline
	}

	statuses, err := s.repo.Submission.GetProblemStatuses(ctx, entry.ID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get problem statuses: %w", op, err)
	}
	details.ProblemStatuses = statuses

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

type LeaderboardResult struct {
	Leaderboard []models.LeaderboardEntry
	Total       int
}

func (s *ContestService) GetLeaderboard(ctx context.Context, contestID int, limit, offset int) (*LeaderboardResult, error) {
	op := "service.ContestService.GetLeaderboard"

	_, err := s.repo.Contest.GetByID(ctx, contestID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrContestNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get contest: %w", op, err)
	}

	leaderboard, total, err := s.repo.Contest.GetLeaderboard(ctx, contestID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get leaderboard: %w", op, err)
	}

	return &LeaderboardResult{
		Leaderboard: leaderboard,
		Total:       total,
	}, nil
}
