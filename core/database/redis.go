package database

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	MetadataFileUpdateQueue = "q::metadata::update::file"

	// Per user queue
	MetadataUpdateClientsNotifierQueue = "q::metadata::update::clients::%s"
)

func NewRedisConnection(redisURL string) (*redis.Client, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	// redis.NewClusterClient()
	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		2*time.Second,
	)
	defer cancel()

	err = client.Ping(ctx).Err()
	if err != nil {
		return nil, err
	}

	return client, nil
}
