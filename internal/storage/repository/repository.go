package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/voidcontests/api/internal/storage/models"
	"github.com/voidcontests/api/internal/storage/repository/postgres"
	"github.com/voidcontests/api/internal/storage/repository/postgres/contest"
	"github.com/voidcontests/api/internal/storage/repository/postgres/entry"
	"github.com/voidcontests/api/internal/storage/repository/postgres/payment"
	"github.com/voidcontests/api/internal/storage/repository/postgres/problem"
	"github.com/voidcontests/api/internal/storage/repository/postgres/submission"
	"github.com/voidcontests/api/internal/storage/repository/postgres/user"
	"github.com/voidcontests/api/internal/storage/repository/postgres/wallet"
)

type Repository struct {
	User       User
	Contest    Contest
	Problem    Problem
	Entry      Entry
	Submission Submission
	Payment    Payment
	TxManager  *postgres.TxManager
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{
		User:       user.New(pool),
		Contest:    contest.New(pool),
		Problem:    problem.New(pool),
		Entry:      entry.New(pool),
		Submission: submission.New(pool),
		Payment:    payment.New(pool),
		TxManager:  postgres.NewTxManager(pool),
	}
}

type TxRepository struct {
	User       User
	Contest    Contest
	Problem    Problem
	Entry      Entry
	Submission Submission
	Wallet     Wallet
	Payment    Payment
}

func NewTxRepository(tx pgx.Tx) *TxRepository {
	return &TxRepository{
		Contest:    contest.New(tx),
		Wallet:     wallet.New(tx),
		User:       user.New(tx),
		Entry:      entry.New(tx),
		Submission: submission.New(tx),
		Problem:    problem.New(tx),
		Payment:    payment.New(tx),
	}
}

type User interface {
	GetByCredentials(ctx context.Context, username string, passwordHash string) (models.User, error)
	Create(ctx context.Context, username string, passwordHash string) (models.User, error)
	Exists(ctx context.Context, username string) (bool, error)
	GetByID(ctx context.Context, id int) (models.User, error)
	GetByUsername(ctx context.Context, username string) (models.User, error)
	GetRole(ctx context.Context, userID int) (models.Role, error)
	GetCreatedProblemsCount(ctx context.Context, userID int) (int, error)
	GetCreatedContestsCount(ctx context.Context, userID int) (int, error)
	UpdateUser(ctx context.Context, userID int, params models.UpdateUserParams) (models.User, error)
}

type Contest interface {
	Create(ctx context.Context, creatorID int, title, desc, awardType string, entryPriceTonNanos uint64, startTime, endTime time.Time, durationMins, maxEntries int, allowLateJoin bool, problems []models.ProblemCharcode, walletID *int) (int, error)
	GetByID(ctx context.Context, contestID int) (models.Contest, error)
	GetProblemset(ctx context.Context, contestID int) ([]models.Problem, error)
	ListAll(ctx context.Context, limit int, offset int, filters models.ContestFilters) (contests []models.Contest, total int, err error)
	GetWithCreatorID(ctx context.Context, creatorID int, limit, offset int) (contests []models.Contest, total int, err error)
	GetEntriesCount(ctx context.Context, contestID int) (int, error)
	IsTitleOccupied(ctx context.Context, title string) (bool, error)
	GetScores(ctx context.Context, contestID, limit, offset int) (scores []models.ScoresEntry, total int, err error)
	GetWallet(ctx context.Context, walletID int) (models.Wallet, error)
	SetDistributionPaymentID(ctx context.Context, contestID int, paymentID int) error
	GetWithUndistributedAwards(ctx context.Context) ([]models.Contest, error)
	GetWinnerID(ctx context.Context, contestID int) (int, error)
}

type Wallet interface {
	Create(ctx context.Context, address, mnemonic string) (int, error)
}

type Problem interface {
	Create(ctx context.Context, writerID int, title, statement, difficulty string, timeLimitMS, memoryLimitMB int, checker string) (int, error)
	AssociateTestCases(ctx context.Context, problemID int, tcs []models.TestCaseDTO) error
	Get(ctx context.Context, contestID int, charcode string) (models.Problem, error)
	GetByID(ctx context.Context, problemID int) (models.Problem, error)
	GetExampleCases(ctx context.Context, problemID int) ([]models.TestCase, error)
	GetTestCaseByID(ctx context.Context, testCaseID int) (models.TestCase, error)
	GetAll(ctx context.Context) ([]models.Problem, error)
	GetWithWriterID(ctx context.Context, writerID int, limit, offset int) (problems []models.Problem, total int, err error)
}

type Entry interface {
	Create(ctx context.Context, contestID int, userID int) (int, error)
	Get(ctx context.Context, contestID int, userID int) (models.Entry, error)
	SetPaymentID(ctx context.Context, entryID int, paymentID int) error
}

type Submission interface {
	Create(ctx context.Context, entryID int, problemID int, code string, language string) (models.Submission, error)
	GetProblemStatus(ctx context.Context, entryID int, problemID int) (string, error)
	GetProblemStatuses(ctx context.Context, entryID int) (map[int]string, error)
	GetByID(ctx context.Context, submissionID int) (models.Submission, error)
	ListByProblem(ctx context.Context, entryID int, charcode string, limit int, offset int) (items []models.Submission, total int, err error)
	GetTestingReport(ctx context.Context, submissionID int) (models.TestingReport, error)
}

type Payment interface {
	Create(ctx context.Context, txHash, fromAddress, toAddress string, amountTonNanos uint64, isIncoming bool) (int, error)
	GetByID(ctx context.Context, paymentID int) (models.Payment, error)
	GetByTxHash(ctx context.Context, txHash string) (models.Payment, error)
}
