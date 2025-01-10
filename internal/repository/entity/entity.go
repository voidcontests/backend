package entity

import "time"

type Contest struct {
	ID             int32         `json:"id" db:"id"`
	Title          string        `json:"title" db:"title"`
	Description    string        `json:"description" db:"description"`
	CreatorAddress string        `json:"creator_address" db:"creator_address"`
	StartingAt     time.Time     `json:"start" db:"starting_at"`
	Duration       time.Duration `json:"duration" db:"duration_mins"`
	IsDraft        bool          `json:"is_draft" db:"is_draft"`
	CreatedAt      time.Time     `json:"created_at" db:"created_at"`
}

type Problem struct {
	ID            int32     `json:"id" db:"id"`
	ContestID     int32     `json:"contest_id" db:"contest_id"`
	Title         string    `json:"title" db:"title"`
	Statement     string    `json:"statement" db:"statement"`
	Difficulty    string    `json:"difficulty" db:"difficulty"`
	WriterAddress string    `json:"writer_address" db:"writer_address"`
	Input         string    `json:"input" db:"input"`
	Answer        string    `json:"answer" db:"answer"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}
