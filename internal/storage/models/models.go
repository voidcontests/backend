package models

import "time"

const (
	RoleAdmin     = "admin"
	RoleUnlimited = "unlimited"
	RoleLimited   = "limited"
	RoleBanned    = "banned"
)

type User struct {
	ID           int       `db:"id"`
	Username     string    `db:"username"`
	PasswordHash string    `db:"password_hash"`
	RoleID       int       `db:"role_id"`
	Address      *string   `db:"address"`
	CreatedAt    time.Time `db:"created_at"`
}

type Role struct {
	ID                   int       `db:"id"`
	Name                 string    `db:"name"`
	CreatedProblemsLimit int       `db:"created_problems_limit"`
	CreatedContestsLimit int       `db:"created_contests_limit"`
	IsDefault            bool      `db:"is_default"`
	CreatedAt            time.Time `db:"created_at"`
}

type Contest struct {
	ID                    int       `db:"id"`
	CreatorID             int       `db:"creator_id"`
	CreatorUsername       string    `db:"creator_username"`
	Title                 string    `db:"title"`
	Description           string    `db:"description"`
	AwardType             string    `db:"award_type"`
	EntryPriceTonNanos    uint64    `db:"entry_price_ton_nanos"`
	StartTime             time.Time `db:"start_time"`
	EndTime               time.Time `db:"end_time"`
	DurationMins          int       `db:"duration_mins"`
	MaxEntries            int       `db:"max_entries"`
	AllowLateJoin         bool      `db:"allow_late_join"`
	ParticipantsCount     int       `db:"participants"`
	WalletID              *int      `db:"wallet_id"`
	DistributionPaymentID *int      `db:"distribution_payment_id"`
	CreatedAt             time.Time `db:"created_at"`
}

type ContestFilters struct {
	CreatorID int
	Title     string
}

type ProblemCharcode struct {
	ProblemID int
	Charcode  string
}

type Wallet struct {
	ID        int       `db:"id"`
	Address   string    `db:"address"`
	Mnemonic  string    `db:"mnemonic"`
	CreatedAt time.Time `db:"created_at"`
}

type Payment struct {
	ID             int       `db:"id"`
	TxHash         string    `db:"tx_hash"`
	FromAddress    string    `db:"from_address"`
	ToAddress      string    `db:"to_address"`
	AmountTonNanos uint64    `db:"amount_ton_nanos"`
	IsIncoming     bool      `db:"is_incoming"`
	CreatedAt      time.Time `db:"created_at"`
}

type Problem struct {
	ID             int       `db:"id"`
	Charcode       string    `db:"charcode"`
	WriterID       int       `db:"writer_id"`
	WriterUsername string    `db:"writer_username"`
	Title          string    `db:"title"`
	Statement      string    `db:"statement"`
	Difficulty     string    `db:"difficulty"`
	TimeLimitMS    int       `db:"time_limit_ms"`
	MemoryLimitMB  int       `db:"memory_limit_mb"`
	Checker        string    `db:"checker"`
	CreatedAt      time.Time `db:"created_at"`
}

type TestCase struct {
	ID        int    `db:"id"`
	ProblemID int    `db:"problem_id"`
	Ordinal   int    `db:"ordinal"`
	Input     string `db:"input"`
	Output    string `db:"output"`
	IsExample bool   `db:"is_example"`
}

type TestCaseDTO struct {
	Input     string `json:"input"`
	Output    string `json:"output"`
	IsExample bool   `json:"is_example"`
}

type Entry struct {
	ID        int       `db:"id"`
	ContestID int       `db:"contest_id"`
	UserID    int       `db:"user_id"`
	PaymentID *int      `db:"payment_id"`
	CreatedAt time.Time `db:"created_at"`
}

type Submission struct {
	ID        int       `db:"id"`
	EntryID   int       `db:"entry_id"`
	ProblemID int       `db:"problem_id"`
	ContestID int       `db:"contest_id"`
	UserID    int       `db:"user_id"`
	Username  string    `db:"username"`
	Status    string    `db:"status"`
	Verdict   string    `db:"verdict"`
	Code      string    `db:"code"`
	Language  string    `db:"language"`
	CreatedAt time.Time `db:"created_at"`
}

type TestingReport struct {
	ID                    int       `db:"id"`
	SubmissionID          int       `db:"submission_id"`
	PassedTestsCount      int       `db:"passed_tests_count"`
	TotalTestsCount       int       `db:"total_tests_count"`
	FirstFailedTestID     *int      `db:"first_failed_test_id"`
	FirstFailedTestOutput *string   `db:"first_failed_test_output"`
	Stderr                string    `db:"stderr"`
	CreatedAt             time.Time `db:"created_at"`
}

type LeaderboardEntry struct {
	UserID   int    `db:"user_id" json:"user_id"`
	Username string `db:"username" json:"username"`
	Points   int    `db:"points" json:"points"`
}

type FailedTest struct {
	ID             int       `db:"id"`
	SubmissionID   int       `db:"submission_id"`
	Input          string    `db:"input"`
	ExpectedOutput string    `db:"expected_output"`
	ActualOutput   string    `db:"actual_output"`
	CreatedAt      time.Time `db:"created_at"`
}
