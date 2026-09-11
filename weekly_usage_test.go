package main

import (
	"auxilia-webserver/internal/store"
	"encoding/csv"
	"strings"
	"testing"
)

func TestWeeklyCSVColumnsMissingCharactersAndEmptyWeeks(t *testing.T) {
	rate := 50.0
	weeks := []store.UsageWeek{
		{WeekStart: "2026-09-14", Counts: map[string]uint64{"aoi": 1, "jude": 0}, Rates: map[string]*float64{"aoi": &rate, "jude": nil}},
		{WeekStart: "2026-09-21", Counts: map[string]uint64{"aoi": 0, "jude": 0, "new-character": 0}, Rates: map[string]*float64{}},
	}
	for _, rates := range []bool{false, true} {
		data, err := usageCSV(weeks, rates)
		if err != nil {
			t.Fatal(err)
		}
		rows, err := csv.NewReader(strings.NewReader(string(data))).ReadAll()
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 3 || rows[0][0] != "week_start" || rows[1][0] != "2026-09-14" {
			t.Fatal("wrong orientation")
		}
		indexes := map[string]int{}
		for i, id := range rows[0] {
			indexes[id] = i
			if i > 1 && rows[0][i-1] > id {
				t.Fatal("IDs not alphabetically sorted")
			}
		}
		want := "1"
		if rates {
			want = "50.0000"
		}
		if rows[1][indexes["aoi"]] != want {
			t.Fatal("wrong value")
		}
		if rows[1][indexes["new-character"]] != "" {
			t.Fatal("character before introduction must be blank")
		}
		want = "0"
		if rates {
			want = ""
		}
		if rows[2][indexes["aoi"]] != want {
			t.Fatal("incorrect zero-match week")
		}
	}
}
