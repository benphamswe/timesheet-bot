// Package kernel is the shared kernel: value objects, the domain event
// interface and the [Clock] port that every bounded context's domain may
// import. Value objects are validated by their constructors ([NewDuration],
// [NewWorkDay], [NewIssueKey], [ParseWeekID]); the package has no I/O and
// imports only the standard library.
package kernel

// WorkActivity is one piece of evidence of work, e.g. a Slack message.
// It is shared between the Activity and Allocation contexts.
type WorkActivity struct {
	Day         WorkDay
	ChannelID   string
	ChannelName string
	Text        string
	Permalink   string
	ThreadTS    string
	IssueKeys   []IssueKey
}

// ReportLine is one line of a timesheet summary the owner wrote in Slack:
// time spent on a project on a day. It is shared between the Activity
// context, whose collector may deliver lines an LLM already parsed (activity
// file v2), and the Allocation context, which turns lines into entries.
type ReportLine struct {
	Date            string // YYYY-MM-DD
	Project         string // as written in the report; matched to Mavenlink case-insensitively
	Minutes         int
	Notes           string
	SourceTS        string // ts of the Slack message the line came from, when known
	SourceChannelID string // channel of that message, when known
}
