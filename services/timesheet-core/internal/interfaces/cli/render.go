package cli

import (
	"fmt"
	"io"
	"text/tabwriter"

	"timesheet-bot/internal/shared/kernel"
	"timesheet-bot/internal/timesheet/domain"
)

func renderTimesheet(w io.Writer, ts *domain.Timesheet, reasoning map[string]string) error {
	fmt.Fprintf(w, "Week %s   status %s   policy %s\n\n", ts.Week(), ts.Status(), ts.Policy())
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "DAY\tISSUE\tTIME\tDESCRIPTION")
	entries := ts.Entries()
	weekTotal := 0
	for _, day := range ts.Week().Days() {
		label := day.Time().Format("Mon 01-02")
		total := 0
		for _, e := range entries {
			if e.Day != day {
				continue
			}
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", label, e.IssueKey, e.Duration, e.Description)
			label = ""
			total += e.Duration.Minutes()
		}
		if total == 0 {
			fmt.Fprintf(tw, "%s\t-\t-\tno activity\n", label)
			continue
		}
		fmt.Fprintf(tw, "\ttotal\t%s\t%s\n", minutes(total), reasoning[day.String()])
		weekTotal += total
	}
	fmt.Fprintf(tw, "\tWEEK\t%s\t\n", minutes(weekTotal))
	return tw.Flush()
}

func minutes(m int) string {
	if m == 0 {
		return "0m"
	}
	return kernel.MustDuration(m).String()
}
