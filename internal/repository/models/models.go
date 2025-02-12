package models

import "time"

type User struct {
	ID        int32     `db:"id"`
	Address   string    `db:"address"`
	CreatedAt time.Time `db:"created_at"`
}

type Contest struct {
	ID             int32     `db:"id"`
	CreatorID      int32     `db:"creator_id"`
	CreatorAddress string    `db:"creator_address"`
	Title          string    `db:"title"`
	Description    string    `db:"description"`
	StartTime      time.Time `db:"start_time"`
	EndTime        time.Time `db:"end_time"`
	DurationMins   int32     `db:"duration_mins"`
	IsDraft        bool      `db:"is_draft"`
	CreatedAt      time.Time `db:"created_at"`
}

type Problem struct {
	ID            int32     `db:"id"`
	WriterID      int32     `db:"writer_id"`
	WriterAddress string    `db:"writer_address"`
	Title         string    `db:"title"`
	Statement     string    `db:"statement"`
	Difficulty    string    `db:"difficulty"`
	Input         string    `db:"input"`
	Answer        string    `db:"answer"`
	CreatedAt     time.Time `db:"created_at"`
}

type Entry struct {
	ID        int32     `db:"id"`
	ContestID int32     `db:"contest_id"`
	UserID    int32     `db:"user_id"`
	CreatedAt time.Time `db:"created_at"`
}

type Submission struct {
	ID        int32     `db:"id"`
	EntryID   int32     `db:"entry_id"`
	ProblemID int32     `db:"problem_id"`
	Verdict   string    `db:"verdict"`
	Answer    string    `db:"answer"`
	CreatedAt time.Time `db:"created_at"`
}

type LeaderboardEntry struct {
	UserID      int32  `db:"user_id" json:"user_id"`
	UserAddress string `db:"user_address" json:"user_address"`
	Points      int    `db:"points" json:"points"`
}
