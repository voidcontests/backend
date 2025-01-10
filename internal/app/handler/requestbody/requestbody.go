package requestbody

import "time"

type Contest struct {
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Problems     []Problem `json:"problems"`
	StartingAt   time.Time `json:"starting_at"`
	DurationMins int32     `json:"duration_mins"`
	IsDraft      bool      `json:"is_draft"`
}

type Problem struct {
	Title      string `json:"title"`
	Statement  string `json:"statement"`
	Difficulty string `json:"difficulty"`
	Input      string `json:"input"`
	Answer     string `json:"answer"`
}
