package request

import (
	"time"

	"github.com/voidcontests/api/internal/storage/models"
)

type CreateAccount struct {
	Username string `json:"username" required:"true"`
	Password string `json:"password" required:"true"`
}

type CreateSession struct {
	Username string `json:"username" required:"true"`
	Password string `json:"password" required:"true"`
}

type CreateContestRequest struct {
	Title         string    `json:"title" required:"true"`
	Description   string    `json:"description"`
	AwardType     string    `json:"award_type"`
	ProblemsIDs   []int     `json:"problems_ids" required:"true"`
	StartTime     time.Time `json:"start_time" required:"true"`
	EndTime       time.Time `json:"end_time" required:"true"`
	DurationMins  int       `json:"duration_mins" requried:"true"`
	MaxEntries    int       `json:"max_entries"`
	AllowLateJoin bool      `json:"allow_late_join"`
}

type CreateProblemRequest struct {
	Title         string               `json:"title" required:"true"`
	Statement     string               `json:"statement" required:"true"`
	Difficulty    string               `json:"difficulty" required:"true"`
	TimeLimitMS   int                  `json:"time_limit_ms"`
	MemoryLimitMB int                  `json:"memory_limit_mb"`
	Checker       string               `json:"checker"`
	TestCases     []models.TestCaseDTO `json:"test_cases"`
}

type CreateSubmissionRequest struct {
	Code     string `json:"code"`
	Language string `json:"language"`
}
