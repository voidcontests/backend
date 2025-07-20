package entry

import (
	"github.com/voidcontests/backend/internal/config"
	"github.com/voidcontests/backend/internal/repository"
)

type Service struct {
	config *config.Config
	repo   *repository.Repository
}

func NewService(c *config.Config, r *repository.Repository) *Service {
	return &Service{
		config: c,
		repo:   r,
	}
}
