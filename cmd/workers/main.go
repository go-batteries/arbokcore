package main

import (
	"arbokcore/core/workers"
	"arbokcore/pkg/config"
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

type ReconcileRunner struct{}

func (s *ReconcileRunner) Run(c *cli.Context) error {
	envDir := c.String(EnvCmd)
	supervisor := c.String(SupervisorName)

	log.Info().Str("reconciler", supervisor).Msg("init reconciler")

	cfg := config.Load(envDir)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return workers.ManualChunkMetadataReconciler(ctx, cfg)
}

type SuperviseRunner struct{}

func (s *SuperviseRunner) Run(c *cli.Context) error {
	envDir := c.String(EnvCmd)
	supervisor := c.String(SupervisorName)

	log.Info().Str("supervisor", supervisor).Msg("init supervisor")

	cfg := config.Load(envDir)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return workers.MetdataSupervisor(ctx, cfg)
}

func main() {
	superviseCmd := &SuperviseRunner{}
	reconcileCmd := &ReconcileRunner{}

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
			{
				Name:  "reconcile",
				Usage: "arbok reconcile -name [name]",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     SupervisorName,
						Required: true},
				},
				Action: reconcileCmd.Run,
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err).Msg("command failed")
	}
}
