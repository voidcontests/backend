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
	ID int32 `json:"id"`
}

type Token struct {
	Token string `json:"token"`
}

type Account struct {
	ID       int32  `json:"id"`
	Username string `json:"username"`
	Role     Role   `json:"role"`
}

type Role struct {
	Name                 string `json:"name"`
	CreatedProblemsLimit int32  `json:"created_problems_limit"`
	CreatedContestsLimit int32  `json:"created_contests_limit"`
}

type User struct {
	ID       int32  `json:"id"`
	Username string `json:"username"`
}

type ContestDetailed struct {
	ID                 int32                    `json:"id"`
	Creator            User                     `json:"creator"`
	Title              string                   `json:"title"`
	Description        string                   `json:"description"`
	StartTime          time.Time                `json:"start_time"`
	EndTime            time.Time                `json:"end_time"`
	DurationMins       int32                    `json:"duration_mins"`
	MaxEntries         int32                    `json:"max_entries,omitempty"`
	Participants       int32                    `json:"participants"`
	AllowLateJoin      bool                     `json:"allow_late_join"`
	IsParticipant      bool                     `json:"is_participant,omitempty"`
	SubmissionDeadline *time.Time               `json:"submission_deadline,omitempty"`
	Problems           []ContestProblemListItem `json:"problems"`
	CreatedAt          time.Time                `json:"created_at"`
}

type ContestListItem struct {
	ID           int32     `json:"id"`
	Creator      User      `json:"creator"`
	Title        string    `json:"title"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	DurationMins int32     `json:"duration_mins"`
	MaxEntries   int32     `json:"max_entries,omitempty"`
	Participants int32     `json:"participants"`
	CreatedAt    time.Time `json:"created_at"`
}

type Submission struct {
	ID            int32          `json:"id"`
	ProblemID     int32          `json:"problem_id"`
	Status        string         `json:"status"`
	Verdict       string         `json:"verdict"`
	Code          string         `json:"code,omitempty"`
	Language      string         `json:"language,omitempty"`
	TestingReport *TestingReport `json:"testing_report,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
}

type TestingReport struct {
	ID               int32     `json:"id"`
	PassedTestsCount int32     `json:"passed_tests_count"`
	TotalTestsCount  int32     `json:"total_tests_count"`
	FailedTest       *Test     `json:"failed_test,omitempty"`
	Stderr           string    `json:"stderr"`
	CreatedAt        time.Time `json:"created_at"`
}

type Test struct {
	Input          string `json:"input"`
	ExpectedOutput string `json:"expected_output"`
	ActualOutput   string `json:"actual_output"`
}

type ContestProblemDetailed struct {
	ID                 int32      `json:"id"`
	Charcode           string     `json:"charcode"`
	ContestID          int32      `json:"contest_id"`
	Writer             User       `json:"writer"`
	Title              string     `json:"title"`
	Statement          string     `json:"statement"`
	Examples           []TC       `json:"examples,omitempty"`
	Difficulty         string     `json:"difficulty"`
	Status             string     `json:"status,omitempty"`
	TimeLimitMS        int32      `json:"time_limit_ms"`
	MemoryLimitMB      int32      `json:"memory_limit_mb"`
	SubmissionDeadline *time.Time `json:"submission_deadline,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
}

type ContestProblemListItem struct {
	ID            int32     `json:"id"`
	Charcode      string    `json:"charcode"`
	Writer        User      `json:"writer"`
	Title         string    `json:"title"`
	Difficulty    string    `json:"difficulty"`
	Status        string    `json:"status,omitempty"`
	TimeLimitMS   int32     `json:"time_limit_ms"`
	MemoryLimitMB int32     `json:"memory_limit_mb"`
	CreatedAt     time.Time `json:"created_at"`
}

type ProblemDetailed struct {
	ID            int32     `json:"id"`
	Writer        User      `json:"writer"`
	Title         string    `json:"title"`
	Statement     string    `json:"statement"`
	Examples      []TC      `json:"examples,omitempty"`
	Difficulty    string    `json:"difficulty"`
	TimeLimitMS   int32     `json:"time_limit_ms"`
	MemoryLimitMB int32     `json:"memory_limit_mb"`
	CreatedAt     time.Time `json:"created_at"`
}

type ProblemListItem struct {
	ID            int32     `json:"id"`
	Writer        User      `json:"writer"`
	Title         string    `json:"title"`
	Difficulty    string    `json:"difficulty"`
	TimeLimitMS   int32     `json:"time_limit_ms"`
	MemoryLimitMB int32     `json:"memory_limit_mb"`
	CreatedAt     time.Time `json:"created_at"`
}

type TC struct {
	Input  string `json:"input"`
	Output string `json:"output"`
}
