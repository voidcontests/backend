package request

import "time"

type CreateAccount struct {
	Username string `json:"username" required:"true"`
	Password string `json:"password" required:"true"`
}

type CreateSession struct {
	Username string `json:"username" required:"true"`
	Password string `json:"password" required:"true"`
}

// TODO: think about draft contests
type CreateContestRequest struct {
	Title         string    `json:"title" required:"true"`
	Description   string    `json:"description"`
	ProblemsIDs   []int32   `json:"problems_ids" required:"true"`
	StartTime     time.Time `json:"start_time" required:"true"`
	EndTime       time.Time `json:"end_time" required:"true"`
	DurationMins  int32     `json:"duration_mins" requried:"true"`
	MaxEntries    int32     `json:"max_entries"`
	AllowLateJoin bool      `json:"allow_late_join"`
}

type CreateProblemRequest struct {
	Title       string `json:"title" required:"true"`
	Kind        string `json:"kind" required:"true"`
	Statement   string `json:"statement" required:"true"`
	Difficulty  string `json:"difficulty" required:"true"`
	TimeLimitMS int    `json:"time_limit_ms"`
	TestCases   []TC   `json:"test_cases"`
	Answer      string `json:"answer"`
}

type TC struct {
	Input     string `json:"input"`
	Output    string `json:"output"`
	IsExample bool   `json:"is_example"`
}

type CreateSubmissionRequest struct {
	ProblemKind string `json:"problem_kind" required:"true"`
	Answer      string `json:"answer"`
	Code        string `json:"code"`
	Language    string `json:"language"`
}
