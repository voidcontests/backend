package response

import "time"

type ContestID struct {
	ID int32 `json:"id"`
}

type ContestDetailed struct {
	ID            int32             `json:"id"`
	CreatorID     int32             `json:"creator_id"`
	Title         string            `json:"title"`
	Description   string            `json:"description"`
	StartingAt    time.Time         `json:"starting_at"`
	DurationMins  int32             `json:"duration_mins"`
	IsDraft       bool              `json:"is_draft,omitempty"`
	IsParticipant bool              `json:"is_participant,omitempty"`
	Problems      []ProblemListItem `json:"problems"`
}

type ProblemListItem struct {
	ID         int32  `json:"id"`
	ContestID  int32  `json:"contest_id"`
	WriterID   int32  `json:"writer_id"`
	Title      string `json:"title"`
	Difficulty string `json:"difficulty"`
	Status     string `json:"status,omitempty"`
}

type ContestListItem struct {
	ID           int32     `json:"id"`
	CreatorID    int32     `json:"creator_id"`
	Title        string    `json:"title"`
	StartingAt   time.Time `json:"starting_at"`
	DurationMins int32     `json:"duration_mins"`
}

type SubmissionListItem struct {
	ID        int32     `json:"id"`
	ProblemID int32     `json:"problem_id"`
	Verdict   string    `json:"verdict"`
	CreatedAt time.Time `json:"created_at"`
}
