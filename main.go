package main

import (
	"github.com/getsentry/sentry-go"
	"github.com/urfave/cli/v2"
	"os"
	"time"
	"unitski-backup/unitski/commands"
)

const sentryFlagKey = "sentry"
const configFlagKey = "config"

var sentryIsInit bool

func main() {
	// Build the CLI app
	sentryFlag := &cli.StringFlag{
		Name:    sentryFlagKey,
		Aliases: []string{"s"},
		Usage:   "Sentry DSN",
	}
	configFlag := &cli.StringFlag{
		Name:      configFlagKey,
		Aliases:   []string{"c"},
		Usage:     "path to the config file",
		Required:  true,
		TakesFile: true,
	}

	app := &cli.App{
		Name:        "Unitski Backup",
		Version:     "1.0.0",
		Usage:       "This version better work cause this project will be unmaintined in like 1 month",
		Description: "A simple program for making back-ups of MySQL docker containers & create tar balls of file(paths).",
		Commands: []*cli.Command{
			{
				Name:  "backup",
				Usage: "run the backup using the given config",
				Flags: []cli.Flag{
					configFlag,
					sentryFlag,
				},
				Action: func(ctx *cli.Context) error {
					initSentry(ctx)
					commands.Sync(ctx.String(configFlagKey))

					return nil
				},
			},
			{
				Name:    "test-config",
				Aliases: []string{"test"},
				Usage:   "test the given config file",
				Flags: []cli.Flag{
					configFlag,
				},
				Action: func(ctx *cli.Context) error {
					panic("Not implemented yet")
				},
			},
		},
		EnableBashCompletion: true,
		Authors: []*cli.Author{
			{
				Name: "Leon Stam",
			},
		},
	}

	// Run the app
	if err := app.Run(os.Args); err != nil {
		sentry.CaptureException(err)
	}

	if sentryIsInit {
		defer sentry.Flush(5 * time.Second)
	}
}

func initSentry(ctx *cli.Context) {
	if dsn := ctx.String(sentryFlagKey); dsn != "" {
		err := sentry.Init(sentry.ClientOptions{
			Dsn:              dsn,
			AttachStacktrace: true,
		})
		if err == nil {
			sentryIsInit = true
		} else {
			panic(err)
		}
	}
}
