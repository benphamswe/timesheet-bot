package kernel

import (
	"errors"
	"time"
)

// ErrWeekend is returned by [NewWorkDay] and [ParseWorkDay] for a Saturday or
// a Sunday.
var ErrWeekend = errors.New("day is a weekend")

// WorkDay is a Monday-Friday calendar date, independent of time zone.
type WorkDay struct{ t time.Time }

// NewWorkDay takes the calendar date of t in t's location.
func NewWorkDay(t time.Time) (WorkDay, error) {
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	if wd := d.Weekday(); wd == time.Saturday || wd == time.Sunday {
		return WorkDay{}, ErrWeekend
	}
	return WorkDay{t: d}, nil
}

// ParseWorkDay parses YYYY-MM-DD.
func ParseWorkDay(s string) (WorkDay, error) {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return WorkDay{}, err
	}
	return NewWorkDay(t)
}

func (d WorkDay) String() string { return d.t.Format(time.DateOnly) }

// Time returns midnight UTC at the start of d.
func (d WorkDay) Time() time.Time { return d.t }

// Weekday returns the day of the week of d.
func (d WorkDay) Weekday() time.Weekday { return d.t.Weekday() }

// Before reports whether d is an earlier date than o.
func (d WorkDay) Before(o WorkDay) bool { return d.t.Before(o.t) }

// IsZero reports whether d is the zero WorkDay, which no constructor returns
// without an error.
func (d WorkDay) IsZero() bool { return d.t.IsZero() }
