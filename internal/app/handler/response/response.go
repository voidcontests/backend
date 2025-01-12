package response

import (
	"time"

	"github.com/voidcontests/backend/internal/repository/entity"
)

type Problem struct {
	ID            int32  `json:"id"`
	ContestID     int32  `json:"contest_id"`
	Title         string `json:"title"`
	Difficulty    string `json:"difficulty"`
	WriterAddress string `json:"writer_address"`
}

type Contest struct {
	ID             int32            `json:"id"`
	Title          string           `json:"title"`
	Description    string           `json:"description"`
	Problemset     []entity.Problem `json:"problemset"`
	CreatorAddress string           `json:"creator_address"`
	StartingAt     time.Time        `json:"starting_at"`
	DurationMins   int32            `json:"duration_mins"`
	IsDraft        bool             `json:"is_draft"`
	CreatedAt      time.Time        `json:"created_at"`
}
