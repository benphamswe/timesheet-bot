package kernel

import (
	"fmt"
	"time"
)

// WeekID identifies an ISO-8601 week, e.g. 2026-W41.
type WeekID struct {
	Year int
	Week int
}

// ParseWeekID parses the "YYYY-Www" form.
func ParseWeekID(s string) (WeekID, error) {
	var y, w int
	if n, err := fmt.Sscanf(s, "%4d-W%2d", &y, &w); err != nil || n != 2 || len(s) != 8 {
		return WeekID{}, fmt.Errorf("invalid week id %q, want YYYY-Www", s)
	}
	id := WeekID{Year: y, Week: w}
	if w < 1 || WeekOf(id.monday()) != id {
		return WeekID{}, fmt.Errorf("week %q does not exist", s)
	}
	return id, nil
}

// WeekOf returns the ISO week containing t (in t's location).
func WeekOf(t time.Time) WeekID {
	y, w := t.ISOWeek()
	return WeekID{Year: y, Week: w}
}

func (w WeekID) String() string { return fmt.Sprintf("%04d-W%02d", w.Year, w.Week) }

// monday returns the Monday of the week at 00:00 UTC.
func (w WeekID) monday() time.Time {
	// Jan 4th is always in ISO week 1.
	jan4 := time.Date(w.Year, time.January, 4, 0, 0, 0, 0, time.UTC)
	offset := (int(jan4.Weekday()) + 6) % 7 // days since Monday
	return jan4.AddDate(0, 0, -offset+(w.Week-1)*7)
}

// Days returns Monday to Friday of the week.
func (w WeekID) Days() []WorkDay {
	m := w.monday()
	days := make([]WorkDay, 5)
	for i := range days {
		days[i] = WorkDay{t: m.AddDate(0, 0, i)}
	}
	return days
}

// Contains reports whether d falls on Monday-Friday of w.
func (w WeekID) Contains(d WorkDay) bool { return WeekOf(d.t) == w }

// Prev returns the previous ISO week.
func (w WeekID) Prev() WeekID { return WeekOf(w.monday().AddDate(0, 0, -7)) }
