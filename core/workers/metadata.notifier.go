package workers

import (
	"arbokcore/core/database"
	"arbokcore/core/supervisors"
	"arbokcore/pkg/config"
	"arbokcore/pkg/queuer"
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

func MetadataUpdateNotification(ctx context.Context, cfg config.AppConfig) error {
	redisconn, err := database.NewRedisConnection(cfg.RedisURL)

	if err != nil {
		log.Error().Err(err).Msg("failed to connect to redis")
		return err
	}

	ctx = context.Background()

	nsq := queuer.NewRedisQ(
		redisconn,
		database.MetadataUpdateClientsNotifierQueue,
		1*time.Second,
	)

	producer := supervisors.NewMetadataNotifier(nsq)
	_ = producer

	return nil
}
