package response

import (
	"time"

	"github.com/voidcontests/backend/internal/ton"
)

type ContestID struct {
	ID int32 `json:"id"`
}

type Account struct {
	ID         int32       `json:"id"`
	TonAccount ton.Account `json:"ton_account"`
	Role       Role        `json:"role"`
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
	Participants  int32             `json:"participants"`
	IsDraft       bool              `json:"is_draft,omitempty"`
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
	Participants int32     `json:"participants"`
	CreatedAt    time.Time `json:"created_at"`
}

type User struct {
	ID      int32  `json:"id"`
	Address string `json:"address"`
}

type SubmissionListItem struct {
	ID        int32     `json:"id"`
	ProblemID int32     `json:"problem_id"`
	Verdict   string    `json:"verdict"`
	CreatedAt time.Time `json:"created_at"`
}

type ProblemDetailed struct {
	ID         int32     `json:"id"`
	Charcode   string    `json:"charcode,omitempty"`
	ContestID  int32     `json:"contest_id,omitempty"`
	Writer     User      `json:"writer"`
	Title      string    `json:"title"`
	Statement  string    `json:"statement"`
	Difficulty string    `json:"difficulty"`
	Status     string    `json:"status,omitempty"`
	Input      string    `json:"input,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}
