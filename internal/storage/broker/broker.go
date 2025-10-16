package broker

import (
	"context"

	"github.com/voidcontests/api/internal/storage/models"
)

type Broker interface {
	PublishSubmission(ctx context.Context, submission models.Submission) error
}
