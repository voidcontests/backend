package responsebody

import "github.com/voidcontests/backend/internal/repository/entity"

type Contests struct {
	Amount   int              `json:"amount"`
	Contests []entity.Contest `json:"contests"`
}

type Problems struct {
	Amount   int              `json:"amount"`
	Problems []entity.Problem `json:"problems"`
}
