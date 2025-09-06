package redis

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
	"github.com/voidcontests/backend/internal/storage/models"
)

type Broker struct {
	client *redis.Client
}

func New(client *redis.Client) *Broker {
	return &Broker{
		client: client,
	}
}

func (b *Broker) PublishSubmission(ctx context.Context, submission models.Submission) error {
	bytes, err := json.Marshal(submission)
	if err != nil {
		return err
	}
	return b.client.Publish(ctx, "submissions", bytes).Err()
}
