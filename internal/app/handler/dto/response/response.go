package response

import (
	"time"
)

type Pagination[T any] struct {
	Meta  Meta `json:"meta"`
	Items []T  `json:"items"`
}

type Meta struct {
	Total   int  `json:"total"`
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	HasNext bool `json:"has_next"`
	HasPrev bool `json:"has_prev"`
}

type ID struct {
	ID int `json:"id"`
}

type Token struct {
	Token string `json:"token"`
}

type Account struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Role     Role   `json:"role"`
}

type Role struct {
	Name                 string `json:"name"`
	CreatedProblemsLimit int    `json:"created_problems_limit"`
	CreatedContestsLimit int    `json:"created_contests_limit"`
}

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

type ContestDetailed struct {
	ID                 int                      `json:"id"`
	Title              string                   `json:"title"`
	Description        string                   `json:"description"`
	AwardType          string                   `json:"award_type"`
	EntryPriceTonNanos uint64                   `json:"entry_price_ton_nanos"`
	Address            string                   `json:"address,omitempty"`
	Creator            User                     `json:"creator"`
	StartTime          time.Time                `json:"start_time"`
	EndTime            time.Time                `json:"end_time"`
	DurationMins       int                      `json:"duration_mins"`
	MaxEntries         int                      `json:"max_entries,omitempty"`
	Participants       int                      `json:"participants"`
	AllowLateJoin      bool                     `json:"allow_late_join"`
	IsParticipant      bool                     `json:"is_participant,omitempty"`
	SubmissionDeadline *time.Time               `json:"submission_deadline,omitempty"`
	Problems           []ContestProblemListItem `json:"problems"`
	Prizes             Prizes                   `json:"prizes"`
	CreatedAt          time.Time                `json:"created_at"`
}

type Prizes struct {
	Nanos uint64 `json:"ton_nanos"`
}

type ContestListItem struct {
	ID                 int       `json:"id"`
	Creator            User      `json:"creator"`
	Title              string    `json:"title"`
	AwardType          string    `json:"award_type"`
	EntryPriceTonNanos uint64    `json:"entry_price_ton_nanos"`
	StartTime          time.Time `json:"start_time"`
	EndTime            time.Time `json:"end_time"`
	DurationMins       int       `json:"duration_mins"`
	MaxEntries         int       `json:"max_entries,omitempty"`
	Participants       int       `json:"participants"`
	CreatedAt          time.Time `json:"created_at"`
}

type Submission struct {
	ID            int            `json:"id"`
	ProblemID     int            `json:"problem_id"`
	Status        string         `json:"status"`
	Verdict       string         `json:"verdict"`
	Code          string         `json:"code,omitempty"`
	Language      string         `json:"language,omitempty"`
	TestingReport *TestingReport `json:"testing_report,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
}

type TestingReport struct {
	ID               int       `json:"id"`
	PassedTestsCount int       `json:"passed_tests_count"`
	TotalTestsCount  int       `json:"total_tests_count"`
	FailedTest       *Test     `json:"failed_test,omitempty"`
	Stderr           string    `json:"stderr,omitemtpy"`
	CreatedAt        time.Time `json:"created_at"`
}

type Test struct {
	Input          string `json:"input"`
	ExpectedOutput string `json:"expected_output"`
	ActualOutput   string `json:"actual_output"`
}

type ContestProblemDetailed struct {
	ID                 int        `json:"id"`
	Charcode           string     `json:"charcode"`
	ContestID          int        `json:"contest_id"`
	Writer             User       `json:"writer"`
	Title              string     `json:"title"`
	Statement          string     `json:"statement"`
	Examples           []TC       `json:"examples,omitempty"`
	Difficulty         string     `json:"difficulty"`
	Status             string     `json:"status,omitempty"`
	TimeLimitMS        int        `json:"time_limit_ms"`
	MemoryLimitMB      int        `json:"memory_limit_mb"`
	Checker            string     `json:"checker"`
	SubmissionDeadline *time.Time `json:"submission_deadline,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
}

type ContestProblemListItem struct {
	ID            int       `json:"id"`
	Charcode      string    `json:"charcode"`
	Writer        User      `json:"writer"`
	Title         string    `json:"title"`
	Difficulty    string    `json:"difficulty"`
	Status        string    `json:"status,omitempty"`
	TimeLimitMS   int       `json:"time_limit_ms"`
	MemoryLimitMB int       `json:"memory_limit_mb"`
	Checker       string    `json:"checker"`
	CreatedAt     time.Time `json:"created_at"`
}

type ProblemDetailed struct {
	ID            int       `json:"id"`
	Writer        User      `json:"writer"`
	Title         string    `json:"title"`
	Statement     string    `json:"statement"`
	Examples      []TC      `json:"examples,omitempty"`
	Difficulty    string    `json:"difficulty"`
	TimeLimitMS   int       `json:"time_limit_ms"`
	MemoryLimitMB int       `json:"memory_limit_mb"`
	Checker       string    `json:"checker"`
	CreatedAt     time.Time `json:"created_at"`
}

type ProblemListItem struct {
	ID            int       `json:"id"`
	Writer        User      `json:"writer"`
	Title         string    `json:"title"`
	Difficulty    string    `json:"difficulty"`
	TimeLimitMS   int       `json:"time_limit_ms"`
	MemoryLimitMB int       `json:"memory_limit_mb"`
	Checker       string    `json:"checker"`
	CreatedAt     time.Time `json:"created_at"`
}

type TC struct {
	Input  string `json:"input"`
	Output string `json:"output"`
}

type Entry struct {
	ID         int       `json:"id"`
	ContestID  int       `json:"contest_id"`
	UserID     int       `json:"user_id"`
	IsPaid     bool      `json:"is_paid"`
	TxHash     string    `json:"tx_hash,omitempty"`
	IsAdmitted bool      `json:"is_admitted"`
	Message    string    `json:"message,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}
