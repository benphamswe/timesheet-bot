// Package cli is the command-line interface. It only parses flags, calls
// application handlers and renders results; wiring happens in cmd/timesheet.
package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"timesheet-bot/internal/shared/kernel"
	"timesheet-bot/internal/timesheet/application"
)

// App holds the use cases a command may need. Nil fields were not requested.
type App struct {
	Propose    *application.ProposeHandler
	Transition *application.TransitionHandler
	Submit     *application.SubmitHandler
	Location   *time.Location
	Close      func()
}

// Needs tells the factory which external dependencies to build.
type Needs struct {
	DB     bool
	Submit bool // the submit targets in config: Tempo, Mavenlink or both
}

// Factory builds an App from the config file. It is supplied by the composition root.
type Factory func(ctx context.Context, configPath string, needs Needs) (*App, error)

// Migrator runs schema migrations; it needs only DATABASE_URL.
type Migrator func(ctx context.Context, direction string) error

// NewRoot returns the timesheet root command with its subcommands. Commands
// that need use cases call factory with the --config path; the migrate
// command calls migrate.
func NewRoot(factory Factory, migrate Migrator) *cobra.Command {
	var configPath string
	root := &cobra.Command{
		Use:           "timesheet",
		Short:         "Fill Jira Tempo and Mavenlink timesheets from Slack activity",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVar(&configPath, "config", "config/config.yaml", "path to config YAML")

	load := func(cmd *cobra.Command, needs Needs) (*App, error) {
		return factory(cmd.Context(), configPath, needs)
	}
	root.AddCommand(
		planCmd(load),
		approveCmd(load),
		rejectCmd(load),
		submitCmd(load),
		migrateCmd(migrate),
	)
	return root
}

type loader func(cmd *cobra.Command, needs Needs) (*App, error)

// addWeekFlag adds --week to cmd. Commands read it with cmd.Flag("week");
// empty means the current week in the configured timezone.
func addWeekFlag(cmd *cobra.Command) {
	cmd.Flags().String("week", "", "ISO week, e.g. 2026-W41 (default: current week)")
}

func resolveWeek(s string, loc *time.Location) (kernel.WeekID, error) {
	if s == "" {
		return kernel.WeekOf(time.Now().In(loc)), nil
	}
	w, err := kernel.ParseWeekID(s)
	if err != nil {
		return kernel.WeekID{}, fmt.Errorf("--week: %w", err)
	}
	return w, nil
}
