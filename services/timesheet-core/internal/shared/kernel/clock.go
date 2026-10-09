package kernel

import "time"

// Clock is the port for reading the current time. Domain code takes a Clock
// or a now argument instead of calling time.Now, so tests can fix the time.
type Clock interface{ Now() time.Time }

// SystemClock returns the current time in Location.
type SystemClock struct{ Location *time.Location }

// Now returns the wall-clock time converted to c.Location. It panics if
// c.Location is nil.
func (c SystemClock) Now() time.Time { return time.Now().In(c.Location) }
