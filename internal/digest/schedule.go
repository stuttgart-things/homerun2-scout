// Package digest periodically summarizes the messages of a time window - how
// many, at which severity, from which systems, compared with the window before
// - and pitches the summary back through omni-pitcher, so it reaches the same
// catchers as the messages it describes (#45).
package digest

import (
	"fmt"
	"time"
)

// Schedule is one kind of digest: when its windows end and how long they are.
type Schedule struct {
	// Name identifies the schedule in the pitched message and in the key
	// that makes sure a window is pitched once: hourly or daily.
	Name string
	// Label is the short window length the title shows: 1h or 24h.
	Label string

	daily        bool
	hour, minute int
}

// Hourly is the schedule whose windows end on every full hour.
func Hourly() Schedule {
	return Schedule{Name: "hourly", Label: "1h"}
}

// Daily is the schedule whose windows end every day at at, a local time of
// day as HH:MM.
func Daily(at string) (Schedule, error) {
	t, err := time.Parse("15:04", at)
	if err != nil {
		return Schedule{}, fmt.Errorf("daily digest time %q is not HH:MM: %w", at, err)
	}
	return Schedule{Name: "daily", Label: "24h", daily: true, hour: t.Hour(), minute: t.Minute()}, nil
}

// LastEnd returns the end of the latest window that ended at or before now,
// in loc. Hourly windows end on the full local hour, daily ones at the
// configured local time of day.
func (s Schedule) LastEnd(now time.Time, loc *time.Location) time.Time {
	local := now.In(loc)
	if !s.daily {
		return time.Date(local.Year(), local.Month(), local.Day(), local.Hour(), 0, 0, 0, loc)
	}
	end := time.Date(local.Year(), local.Month(), local.Day(), s.hour, s.minute, 0, 0, loc)
	if end.After(local) {
		end = end.AddDate(0, 0, -1)
	}
	return end
}

// Start returns the start of the window ending at end. A daily window starts
// at the same local time the day before, so on a daylight saving change it is
// 23 or 25 hours long - the calendar day, not a fixed 24 hours.
func (s Schedule) Start(end time.Time) time.Time {
	if s.daily {
		return end.AddDate(0, 0, -1)
	}
	return end.Add(-time.Hour)
}

// MaxLateness is how long after a window ended its digest is still pitched.
// A scout that was down longer skips that window instead of pitching a
// yesterday's summary in the afternoon.
func (s Schedule) MaxLateness() time.Duration {
	if s.daily {
		return time.Hour
	}
	return 30 * time.Minute
}
