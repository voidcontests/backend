package response

import (
	"time"

	"github.com/voidcontests/backend/internal/app/runner"
)

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

type ContestDetailed struct {
	ID            int32             `json:"id"`
	Creator       User              `json:"creator"`
	Title         string            `json:"title"`
	Description   string            `json:"description"`
	StartTime     time.Time         `json:"start_time"`
	EndTime       time.Time         `json:"end_time"`
	DurationMins  int32             `json:"duration_mins"`
	MaxEntries    int32             `json:"max_entries,omitempty"`
	Participants  int32             `json:"participants"`
	AllowLateJoin bool              `json:"allow_late_join"`
	IsParticipant bool              `json:"is_participant,omitempty"`
	Problems      []ProblemListItem `json:"problems"`
	CreatedAt     time.Time         `json:"created_at"`
}

type ProblemListItem struct {
	ID         int32     `json:"id"`
	Charcode   string    `json:"charcode,omitempty"`
	ContestID  int32     `json:"contest_id,omitempty"`
	Writer     User      `json:"writer"`
	Title      string    `json:"title"`
	Difficulty string    `json:"difficulty"`
	Status     string    `json:"status,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
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

type User struct {
	ID       int32  `json:"id"`
	Username string `json:"username"`
}

type Submission struct {
	ID            int32         `json:"id"`
	ProblemID     int32         `json:"problem_id"`
	Verdict       string        `json:"verdict"`
	Answer        string        `json:"answer,omitempty"`
	Code          string        `json:"code,omitempty"`
	Language      string        `json:"language,omitempty"`
	TestingReport TestingReport `json:"testing_report,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
}

type TestingReport struct {
	Passed     int                `json:"passed"`
	Total      int                `json:"total"`
	Stderr     string             `json:"stderr,omitempty"`
	FailedTest *runner.FailedTest `json:"failed_test,omitempty"`
}

type ProblemDetailed struct {
	ID          int32     `json:"id"`
	Charcode    string    `json:"charcode,omitempty"`
	ContestID   int32     `json:"contest_id,omitempty"`
	Writer      User      `json:"writer"`
	Kind        string    `json:"kind"`
	Title       string    `json:"title"`
	Statement   string    `json:"statement"`
	Examples    []TC      `json:"examples,omitempty"`
	Difficulty  string    `json:"difficulty"`
	Status      string    `json:"status,omitempty"`
	TimeLimitMS int32     `json:"time_limit_ms,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type TC struct {
	Input  string `json:"input"`
	Output string `json:"output"`
}
