package cli

import (
	"fmt"
	"io"
	"text/tabwriter"

	"timesheet-bot/internal/shared/kernel"
	"timesheet-bot/internal/timesheet/application"
	"timesheet-bot/internal/timesheet/domain"
)

// renderTimesheet prints ts as a table. When placements is not nil (Mavenlink
// is a submit target), a MAVENLINK column shows each entry's project and task.
func renderTimesheet(w io.Writer, ts *domain.Timesheet, reasoning map[string]string, placements map[string]application.MavenlinkPlacement) error {
	fmt.Fprintf(w, "Week %s   status %s   policy %s\n\n", ts.Week(), ts.Status(), ts.Policy())
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	// col returns the MAVENLINK cell with its leading tab, or nothing without the column.
	col := func(string) string { return "" }
	head := "DAY\tISSUE/PROJECT\tTIME\tDESCRIPTION"
	if placements != nil {
		col = func(cell string) string { return "\t" + cell }
		head = "DAY\tISSUE/PROJECT\tTIME\tMAVENLINK\tDESCRIPTION"
	}
	fmt.Fprintln(tw, head)
	entries := ts.Entries()
	weekTotal := 0
	for _, day := range ts.Week().Days() {
		label := day.Time().Format("Mon 01-02")
		total := 0
		for _, e := range entries {
			if e.Day != day {
				continue
			}
			fmt.Fprintf(tw, "%s\t%s\t%s%s\t%s\n", label, workOn(e), e.Duration, col(placementText(placements[e.ID])), e.Description)
			label = ""
			total += e.Duration.Minutes()
		}
		if total == 0 {
			fmt.Fprintf(tw, "%s\t-\t-%s\tno activity\n", label, col(""))
			continue
		}
		fmt.Fprintf(tw, "\ttotal\t%s%s\t%s\n", minutes(total), col(""), reasoning[day.String()])
		weekTotal += total
	}
	fmt.Fprintf(tw, "\tWEEK\t%s%s\t\n", minutes(weekTotal), col(""))
	return tw.Flush()
}

// placementText is "project 1000003 task 70 (default)", omitting what is unset.
func placementText(p application.MavenlinkPlacement) string {
	s := fmt.Sprintf("project %d", p.WorkspaceID)
	if p.StoryID > 0 {
		s += fmt.Sprintf(" task %d", p.StoryID)
	}
	if p.Fallback {
		s += " (default)"
	}
	return s
}

// workOn names what an entry logs time on: its Jira issue, or its report project.
func workOn(e domain.Entry) string {
	if e.IssueKey != "" {
		return e.IssueKey.String()
	}
	return e.Project
}

func minutes(m int) string {
	if m == 0 {
		return "0m"
	}
	return kernel.MustDuration(m).String()
}
