package main

import (
	"arbokcore/core/database"
	"arbokcore/pkg/config"
	"context"
	"flag"
	"time"

	"github.com/rs/zerolog/log"
)

func main() {
	var envDir string

	flag.StringVar(&envDir, "env", "./config", "envfile path")
	flag.Parse()

	log.Info().Str("env_file", envDir).Msg("using env file from")

	cfg := config.Load(envDir)

	conn := database.ConnectSqlite(cfg.DbName)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn.Connect(ctx)

	ctx = context.Background()
	if err := conn.Setup(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to setup db schema")
	}

	log.Info().Msg("database schema migration success")
}
