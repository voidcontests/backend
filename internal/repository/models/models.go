package models

import "time"

const (
	RoleAdmin     = "admin"
	RoleUnlimited = "unlimited"
	RoleLimited   = "limited"
	RoleBanned    = "banned"
)

const (
	TextAnswerProblem = "text_answer_problem"
	CodingProblem     = "coding_problem"
)

type User struct {
	ID        int32     `db:"id"`
	Address   string    `db:"address"`
	RoleID    int32     `db:"role_id"`
	CreatedAt time.Time `db:"created_at"`
}

type Role struct {
	ID                   int32     `db:"id"`
	Name                 string    `db:"name"`
	CreatedProblemsLimit int32     `db:"created_problems_limit"`
	CreatedContestsLimit int32     `db:"created_contests_limit"`
	IsDefault            bool      `db:"is_default"`
	CreatedAt            time.Time `db:"created_at"`
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
	MaxEntries     int32     `db:"max_entries"`
	AllowLateJoin  bool      `db:"allow_late_join"`
	KeepAsTraining bool      `db:"keep_as_training"`
	IsDraft        bool      `db:"is_draft"`
	Participants   int32     `db:"participants"`
	CreatedAt      time.Time `db:"created_at"`
}

type Problem struct {
	ID            int32     `db:"id"`
	Charcode      string    `db:"charcode"`
	Kind          string    `db:"kind"`
	WriterID      int32     `db:"writer_id"`
	WriterAddress string    `db:"writer_address"`
	Title         string    `db:"title"`
	Statement     string    `db:"statement"`
	Difficulty    string    `db:"difficulty"`
	Input         string    `db:"input"`
	Answer        string    `db:"answer"`
	Language      string    `db:"language"`
	TimeLimitMS   int32     `db:"time_limit_ms"`
	CreatedAt     time.Time `db:"created_at"`
}

type TestCase struct {
	ID        int32  `db:"id"`
	ProblemID int32  `db:"problem_id"`
	Input     string `db:"input"`
	Output    string `db:"output"`
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
