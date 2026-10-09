package kernel

import "fmt"

// Granularity is the smallest unit of logged time.
const Granularity = 15

// Duration is logged time in minutes, a positive multiple of Granularity.
type Duration struct{ min int }

// NewDuration returns a Duration of the given number of minutes. It returns an
// error unless minutes is a positive multiple of [Granularity].
func NewDuration(minutes int) (Duration, error) {
	if minutes <= 0 || minutes%Granularity != 0 {
		return Duration{}, fmt.Errorf("duration %dm must be a positive multiple of %dm", minutes, Granularity)
	}
	return Duration{min: minutes}, nil
}

// MustDuration is for constants and tests.
func MustDuration(minutes int) Duration {
	d, err := NewDuration(minutes)
	if err != nil {
		panic(err)
	}
	return d
}

// Minutes returns d as a whole number of minutes.
func (d Duration) Minutes() int { return d.min }

// Seconds returns d as a whole number of seconds.
func (d Duration) Seconds() int { return d.min * 60 }

func (d Duration) String() string {
	h, m := d.min/60, d.min%60
	switch {
	case h == 0:
		return fmt.Sprintf("%dm", m)
	case m == 0:
		return fmt.Sprintf("%dh", h)
	default:
		return fmt.Sprintf("%dh%dm", h, m)
	}
}
