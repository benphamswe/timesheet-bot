package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"timesheet-bot/internal/timesheet/application"
)

func planCmd(load loader) *cobra.Command {
	var dryRun, force bool
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Collect Slack activity (or your Slack reports) and propose the week's timesheet",
		RunE: func(cmd *cobra.Command, _ []string) error {
			app, err := load(cmd, Needs{DB: !dryRun})
			if err != nil {
				return err
			}
			defer app.Close()
			week, err := resolveWeek(cmd.Flag("week").Value.String(), app.Location)
			if err != nil {
				return err
			}
			res, err := app.Propose.Handle(cmd.Context(), application.ProposeCommand{Week: week, Force: force, DryRun: dryRun})
			for _, f := range res.Failures {
				fmt.Fprintf(cmd.ErrOrStderr(), "policy %s rejected: %v\n", f.Policy, f.Err)
			}
			if err != nil {
				return err
			}
			if err := renderTimesheet(cmd.OutOrStdout(), res.Timesheet, res.Proposal.Reasoning, res.Placements); err != nil {
				return fmt.Errorf("render timesheet: %w", err)
			}
			if len(res.Proposal.Warnings) > 0 {
				fmt.Fprintln(cmd.OutOrStdout())
			}
			for _, w := range res.Proposal.Warnings {
				fmt.Fprintf(cmd.OutOrStdout(), "warning: %s\n", w)
			}
			if dryRun {
				fmt.Fprintln(cmd.OutOrStdout(), "\ndry run: nothing saved")
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "\nsaved. approve with: timesheet approve --week %s\n", week)
			}
			return nil
		},
	}
	addWeekFlag(cmd)
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the proposal without saving (no database needed)")
	cmd.Flags().BoolVar(&force, "force", false, "replace an existing, not yet submitted timesheet")
	return cmd
}

func approveCmd(load loader) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "approve",
		Short: "Approve the proposed timesheet",
		RunE: func(cmd *cobra.Command, _ []string) error {
			app, err := load(cmd, Needs{DB: true})
			if err != nil {
				return err
			}
			defer app.Close()
			week, err := resolveWeek(cmd.Flag("week").Value.String(), app.Location)
			if err != nil {
				return err
			}
			if err := app.Transition.Approve(cmd.Context(), week); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s approved. send it with: timesheet submit --week %s\n", week, week)
			return nil
		},
	}
	addWeekFlag(cmd)
	return cmd
}

func rejectCmd(load loader) *cobra.Command {
	var reason string
	cmd := &cobra.Command{
		Use:   "reject",
		Short: "Reject the proposed timesheet",
		RunE: func(cmd *cobra.Command, _ []string) error {
			app, err := load(cmd, Needs{DB: true})
			if err != nil {
				return err
			}
			defer app.Close()
			week, err := resolveWeek(cmd.Flag("week").Value.String(), app.Location)
			if err != nil {
				return err
			}
			if err := app.Transition.Reject(cmd.Context(), week, reason); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s rejected. re-plan with: timesheet plan --week %s --force\n", week, week)
			return nil
		},
	}
	addWeekFlag(cmd)
	cmd.Flags().StringVar(&reason, "reason", "", "why it was rejected (used for learning later)")
	return cmd
}

func submitCmd(load loader) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit",
		Short: "Write the approved timesheet to the submit targets (Tempo, Mavenlink)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			app, err := load(cmd, Needs{DB: true, Submit: true})
			if err != nil {
				return err
			}
			defer app.Close()
			week, err := resolveWeek(cmd.Flag("week").Value.String(), app.Location)
			if err != nil {
				return err
			}
			res, err := app.Submit.Handle(cmd.Context(), week)
			fmt.Fprintf(cmd.OutOrStdout(), "%s: %d submitted, %d failed, status %s\n", week, res.Submitted, res.Failed, res.Status)
			if res.Submissions > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "mavenlink: %d timesheet submissions sent for approval\n", res.Submissions)
			}
			if err == nil && res.Failed > 0 {
				err = fmt.Errorf("%d entries failed; run submit again to retry only those", res.Failed)
			}
			return err
		},
	}
	addWeekFlag(cmd)
	return cmd
}

func migrateCmd(migrate Migrator) *cobra.Command {
	return &cobra.Command{
		Use:       "migrate up|down",
		Short:     "Apply or roll back database migrations",
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs: []string{"up", "down"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := migrate(cmd.Context(), args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "migrate %s: ok\n", args[0])
			return nil
		},
	}
}
