package response

import (
	"time"

	"github.com/voidcontests/backend/internal/repository/entity"
)

type Contests struct {
	Amount   int              `json:"amount"`
	Contests []entity.Contest `json:"contests"`
}

type Problems struct {
	Amount   int              `json:"amount"`
	Problems []entity.Problem `json:"problems"`
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
