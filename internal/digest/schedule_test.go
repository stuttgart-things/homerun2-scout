package digest

import (
	"strings"
	"testing"
	"time"
)

func berlin(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func TestScheduleLastEnd(t *testing.T) {
	loc := berlin(t)
	daily, err := Daily("07:00")
	if err != nil {
		t.Fatal(err)
	}
	at := func(s string) time.Time {
		v, err := time.ParseInLocation("2006-01-02 15:04", s, loc)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}

	cases := []struct {
		name     string
		schedule Schedule
		now      time.Time
		want     time.Time
	}{
		{"hourly mid-hour", Hourly(), at("2026-09-12 13:37"), at("2026-09-12 13:00")},
		{"hourly on the hour", Hourly(), at("2026-09-12 14:00"), at("2026-09-12 14:00")},
		{"daily after the time", daily, at("2026-09-12 07:05"), at("2026-09-12 07:00")},
		{"daily at the time", daily, at("2026-09-12 07:00"), at("2026-09-12 07:00")},
		{"daily before the time", daily, at("2026-09-12 06:59"), at("2026-09-11 07:00")},
		// A UTC clock does not decide the day: 23:30 UTC on the 11th is the 12th in Berlin.
		{"daily in the zone, not UTC", daily, time.Date(2026, 9, 12, 5, 30, 0, 0, time.UTC), at("2026-09-12 07:00")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.schedule.LastEnd(tc.now, loc); !got.Equal(tc.want) {
				t.Errorf("LastEnd = %v, want %v", got, tc.want)
			}
		})
	}
}

// On the last Sunday of October (2026-10-25) Berlin falls back from 03:00 to
// 02:00: the day that ends that Sunday at 07:00 is 25 hours long.
func TestDailyWindowFollowsTheCalendarAcrossDST(t *testing.T) {
	loc := berlin(t)
	daily, _ := Daily("07:00")
	end := time.Date(2026, 10, 25, 7, 0, 0, 0, loc)
	if got := end.Sub(daily.Start(end)); got != 25*time.Hour {
		t.Errorf("window length = %v, want 25h", got)
	}
	if got := daily.Start(end.AddDate(0, 0, 1)); end.AddDate(0, 0, 1).Sub(got) != 24*time.Hour {
		t.Errorf("the day after is 24h again, got %v", end.AddDate(0, 0, 1).Sub(got))
	}
	if got := Hourly().Start(end); got != end.Add(-time.Hour) {
		t.Errorf("hourly start = %v", got)
	}
}

func TestDaily(t *testing.T) {
	for _, bad := range []string{"7", "24:00", "07:60", "seven", ""} {
		if _, err := Daily(bad); err == nil || !strings.Contains(err.Error(), "HH:MM") {
			t.Errorf("Daily(%q) error = %v, want an HH:MM error", bad, err)
		}
	}
	s, err := Daily("23:45")
	if err != nil || s.Name != "daily" || s.Label != "24h" || s.hour != 23 || s.minute != 45 {
		t.Errorf("Daily(23:45) = %+v, %v", s, err)
	}
	if Hourly().MaxLateness() != 30*time.Minute || s.MaxLateness() != time.Hour {
		t.Error("unexpected lateness")
	}
}
