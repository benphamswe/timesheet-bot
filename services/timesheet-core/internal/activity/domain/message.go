package domain

import (
	"strings"
	"time"

	"timesheet-bot/internal/shared/kernel"
)

// Message is one of the user's chat messages as an adapter read it, before it
// becomes work evidence. Every Source that reads Slack messages, however it
// gets them, turns them into activities through [ToActivity], so allocation
// sees the same evidence from each.
type Message struct {
	PostedAt    time.Time
	ChannelID   string
	ChannelName string
	Text        string
	Permalink   string
	// ThreadTS is the parent's timestamp when the message is a thread reply,
	// if the adapter knows it; otherwise it is taken from the permalink.
	ThreadTS string
}

// ToActivity returns m as work evidence for week, on the calendar day it was
// posted in loc. ok is false when that day is a weekend or outside week.
func ToActivity(m Message, week kernel.WeekID, loc *time.Location) (a kernel.WorkActivity, ok bool) {
	day, err := kernel.NewWorkDay(m.PostedAt.In(loc))
	if err != nil || !week.Contains(day) {
		return kernel.WorkActivity{}, false // weekend or timezone edge outside the week
	}
	thread := m.ThreadTS
	if thread == "" {
		thread = threadTSFromPermalink(m.Permalink)
	}
	return kernel.WorkActivity{
		Day:         day,
		ChannelID:   m.ChannelID,
		ChannelName: m.ChannelName,
		Text:        m.Text,
		Permalink:   m.Permalink,
		ThreadTS:    thread,
		IssueKeys:   kernel.ExtractIssueKeys(m.Text),
	}, true
}

// threadTSFromPermalink extracts thread_ts from a permalink of a thread reply,
// if present.
func threadTSFromPermalink(permalink string) string {
	_, q, ok := strings.Cut(permalink, "thread_ts=")
	if !ok {
		return ""
	}
	v, _, _ := strings.Cut(q, "&")
	return v
}
