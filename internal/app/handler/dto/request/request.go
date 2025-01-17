package request

import "time"

// TODO: think about draft contests
type CreateContestRequest struct {
	Title       string `json:"title" required:"true"`
	Description string `json:"description"`
	Problems    []struct {
		Title      string `json:"title" required:"true"`
		Statement  string `json:"statement" required:"true"`
		Difficulty string `json:"difficulty" required:"true"`
		Input      string `json:"input"`
		Answer     string `json:"answer" required:"true"`
	} `json:"problems" required:"true"`
	StartingAt   time.Time `json:"starting_at" required:"true"`
	DurationMins int32     `json:"duration_mins" requried:"true"`
}

type CreateSubmissionRequest struct {
	Answer string `json:"answer" required:"true"`
}
