// Package reportparser reads the timesheet summaries the owner writes in
// Slack (report mode, ADR-0012) into report lines.
package reportparser

import (
	"errors"
	"time"

	alloc "timesheet-bot/internal/allocation/domain"
	"timesheet-bot/internal/shared/kernel"
)

// ErrNotImplemented is returned by [Parser.Parse] until the owner writes it.
var ErrNotImplemented = errors.New("report parsing is not implemented yet (TODO(owner) in reportparser.Parser.Parse)")

// Parser is the [alloc.ReportParser] for the owner's summary format.
type Parser struct{}

// New returns a Parser.
func New() *Parser { return &Parser{} }

var _ alloc.ReportParser = (*Parser)(nil)

// Parse returns the report lines in msg, the text of one of the owner's
// summary messages (from the report channel or the self-DM). msg.Day is the
// working day the message was posted, in tz; msg.Text is the raw text.
//
// Return one [alloc.ReportLine] per project block, with Date as YYYY-MM-DD,
// Project as written, Minutes, and Notes. Return no lines and no error for a
// message that is not a summary. An error fails the whole `plan`, naming the
// message, so the owner can fix the report; ReportPolicy then drops exact
// duplicates (the same summary posted in both places) and warns about days
// over the cap.
//
// Decisions this function owns:
//   - Which date a line belongs to: day headings in the text ("Mon 06/10",
//     "2026-10-06", "Today") or the message date (msg.Day). A heading needs
//     the week to resolve a weekday name; tz is passed for "today"/"yesterday"
//     style headings.
//   - Hour formats to accept: "4h", "2.5h", "30m", "1h30", "1:30", maybe
//     "1h 30m". Minutes must end up a positive multiple of 15 (round, or
//     return an error naming the line).
//   - Separators between project, time and notes: for example
//     "ProjectX - 2h - API review", "ProjectX: 2h API review" or
//     "[ProjectX] 2h API review", and bullets ("-", "*", "•") before a line.
//   - Lines without hours: return an error (nothing is lost silently) or skip
//     them (free text around the summary is ignored).
//   - Weekly vs daily messages: one message for the whole week with a heading
//     per day, or one message per day dated by msg.Day, or both.
func (p *Parser) Parse(msg kernel.WorkActivity, tz *time.Location) ([]alloc.ReportLine, error) {
	// TODO(owner): implement report parsing.
	_, _ = msg, tz
	return nil, ErrNotImplemented
}
