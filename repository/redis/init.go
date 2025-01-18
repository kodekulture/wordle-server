package redis

import (
	"context"
	"fmt"

	"github.com/kodekulture/wordle-server/internal/config"
	redis9 "github.com/redis/go-redis/v9"
)

// NewClient ...
func NewClient(ctx context.Context) (*redis9.Client, error) {
	opts, err := redis9.ParseURL(config.Get("REDIS_URL"))
	if err != nil {
		return nil, err
	}

	cl := redis9.NewClient(opts)
	if err := cl.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return cl, nil
}
