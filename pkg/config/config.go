package config

import (
	"os"

	"github.com/go-batteries/diaper"
	"github.com/rs/zerolog/log"
)

type AppConfig struct {
	Dsn    string
	DbName string
}

func Load(envFile string) AppConfig {
	providers := diaper.BuildProviders(diaper.EnvProvider{})
	loader := diaper.DiaperConfig{
		DefaultEnvFile: "app.env",
		Providers:      providers,
	}

	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "dev"
	}

	cfgMap, err := loader.ReadFromFile(env, envFile)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config from " + envFile)
	}

	return AppConfig{
		Dsn:    cfgMap.MustGet("dsn").(string),
		DbName: cfgMap.MustGet("db_name").(string),
	}
}
