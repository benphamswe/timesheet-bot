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
