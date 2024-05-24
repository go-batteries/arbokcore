package workers

import (
	"arbokcore/core/database"
	"arbokcore/core/files"
	"arbokcore/core/notifiers"
	"arbokcore/core/supervisors"
	"arbokcore/pkg/config"
	"arbokcore/pkg/queuer"
	"arbokcore/pkg/squirtle"
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

func MetdataSupervisor(ctx context.Context, cfg config.AppConfig) error {
	conn := database.ConnectSqlite(cfg.DbName)
	dbconn := conn.Connect(ctx)
	redisconn, err := database.NewRedisConnection(cfg.RedisURL)

	if err != nil {
		log.Error().Err(err).Msg("failed to connect to redis")
		return err
	}

	ctx = context.Background()

	qs := squirtle.LoadAll("./config/querystore.yaml")

	metadataQueryStore, err := qs.HydrateQueryStore("file_metadatas")
	if err != nil {
		return err
	}

	repo := files.NewMetadataRepository(
		dbconn,
		metadataQueryStore,
	)

	rsq := queuer.NewRedisQ(
		redisconn,
		database.MetadataFileUpdateQueue,
		1*time.Second,
	)

	nsq := queuer.NewRedisQ(
		redisconn,
		database.MetadataUpdateClientsNotifierQueue,
		1*time.Second,
	)
	notifier := notifiers.NewMedataUpdateStatus(nsq)

	producer := supervisors.NewMetadataChangelog(rsq)

	chunkQueryStore, err := qs.HydrateQueryStore("user_files")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load user files query")
	}

	chunkRepo := files.NewUserFileRespository(dbconn, chunkQueryStore)

	supervisors.MetadataSupervisor(
		ctx,
		repo,
		chunkRepo,
		producer,
		notifier,
	)

	return nil
}
