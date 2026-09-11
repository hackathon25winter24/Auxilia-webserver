package store

import (
	"testing"
	"time"
)

func TestWeekStartsAtMondayMidnightJST(t *testing.T) {
	for _, tt := range []struct{ instant, want string }{
		{"2026-09-13T14:59:59Z", "2026-09-07"},
		{"2026-09-13T15:00:00Z", "2026-09-14"},
		{"2026-09-20T14:59:59Z", "2026-09-14"},
		{"2026-09-20T15:00:00Z", "2026-09-21"},
		{"2027-01-01T00:00:00Z", "2026-12-28"},
	} {
		instant, err := time.Parse(time.RFC3339, tt.instant)
		if err != nil {
			t.Fatal(err)
		}
		got := WeekStart(instant)
		if got.Format("2006-01-02") != tt.want || got.Hour() != 0 {
			t.Fatalf("%s => %s, want %s", tt.instant, got, tt.want)
		}
	}
}
func TestIncompleteInitialWeekIsNotFinalized(t *testing.T) {
	monday := time.Date(2026, 9, 14, 0, 0, 0, 0, japan)
	if !firstFullWeek(monday).Equal(monday) {
		t.Fatal("full week skipped")
	}
	if !firstFullWeek(monday.Add(time.Second)).Equal(monday.AddDate(0, 0, 7)) {
		t.Fatal("partial week treated as full")
	}
}
func TestWeeklyRatesUsePlayerPicksNotTotalCharacterPicks(t *testing.T) {
	start := time.Date(2026, 9, 14, 0, 0, 0, 0, japan)
	week := usageWeek(start, map[string]uint64{"aoi": 2, "jude": 1, "dana": 0}, 2)
	if *week.Rates["aoi"] != 100 || *week.Rates["jude"] != 50 || *week.Rates["dana"] != 0 {
		t.Fatal("incorrect denominator")
	}
	empty := usageWeek(start, map[string]uint64{"aoi": 0}, 0)
	if empty.Rates["aoi"] != nil {
		t.Fatal("zero-match week must have null rate")
	}
	if week.WeekEnd != "2026-09-21" {
		t.Fatal("wrong exclusive week end")
	}
}
