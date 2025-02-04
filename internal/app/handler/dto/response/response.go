package response

import "time"

type ContestID struct {
	ID int32 `json:"id"`
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
}

type ProblemListItem struct {
	ID         int32  `json:"id"`
	ContestID  int32  `json:"contest_id"`
	Writer     User   `json:"writer"`
	Title      string `json:"title"`
	Difficulty string `json:"difficulty"`
	Status     string `json:"status,omitempty"`
}

type ContestListItem struct {
	ID           int32     `json:"id"`
	Creator      User      `json:"creator"`
	Title        string    `json:"title"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	DurationMins int32     `json:"duration_mins"`
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
	ID         int32  `json:"id"`
	ContestID  int32  `json:"contest_id"`
	Writer     User   `json:"writer"`
	Title      string `json:"title"`
	Statement  string `json:"statement"`
	Difficulty string `json:"difficulty"`
	Status     string `json:"status,omitempty"`
	Input      string `json:"input,omitempty"`
}
