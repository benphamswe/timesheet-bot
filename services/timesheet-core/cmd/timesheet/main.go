// Command timesheet fills Jira Tempo timesheets from Slack activity.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	_ "time/tzdata" // distroless images have no zoneinfo

	"timesheet-bot/internal/interfaces/cli"
)

func main() {
	os.Exit(run())
}

// run executes the root command and returns the process exit code. It is
// separate from main so that its deferred calls run before os.Exit.
func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := cli.NewRoot(buildApp, migrate).ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	return 0
}
