package request

import "time"

type CreateContestRequest struct {
	Title       string `json:"title" required:"true"`
	Description string `json:"description"`
	Problems    []struct {
		Title      string `json:"title" required:"true"`
		Statement  string `json:"statement" required:"true"`
		Difficulty string `json:"difficulty" required:"true"`
		Input      string `json:"input"`
		Answer     string `json:"answer" required:"true"`
	} `json:"problems"`
	StartingAt   time.Time `json:"starting_at"`
	DurationMins int32     `json:"duration_mins"`
	IsDraft      bool      `json:"is_draft" required:"true"`
}

type CreateSubmissionRequest struct {
	Answer string `json:"answer" required:"true"`
}
