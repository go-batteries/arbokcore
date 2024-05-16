package main

import (
	"arbokcore/core/database"
	"arbokcore/core/files"
	"arbokcore/core/supervisors"
	"arbokcore/pkg/config"
	"arbokcore/pkg/queuer"
	"arbokcore/pkg/squirtle"
	"context"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

const (
	EnvCmd         = "env"
	SuperviseCmd   = "supervise"
	SupervisorName = "name"
)

type SuperviseRunner struct{}

func (s *SuperviseRunner) Run(c *cli.Context) error {
	envDir := c.String(EnvCmd)
	supervisor := c.String(SupervisorName)

	log.Info().Str("supervisor", supervisor).Msg("init supervisor")

	cfg := config.Load(envDir)
	conn := database.ConnectSqlite(cfg.DbName)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

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
		database.MetadataRedisQueue,
		5*time.Second,
	)

	producer := supervisors.NewMetadataChangelog(rsq)

	supervisors.MetadataSupervisor(
		ctx,
		repo,
		producer,
	)

	return nil
}

func main() {
	superviseCmd := &SuperviseRunner{}

	app := &cli.App{
		Name:  "arbok",
		Usage: "worker invoker arbok",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    EnvCmd,
				Aliases: []string{"e"},
				Value:   "./config",
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "supervise",
				Usage: "arbok supervise -name [name]",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     SupervisorName,
						Required: true},
				},
				Action: superviseCmd.Run,
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err).Msg("command failed")
	}
}
