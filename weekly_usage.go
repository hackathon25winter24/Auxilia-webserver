package main

import (
	"bytes"
	"encoding/csv"
	"net/http"
	"sort"
	"strconv"

	"auxilia-webserver/internal/game"
	"auxilia-webserver/internal/store"
)

func (s *service) currentCharacterUsage(w http.ResponseWriter, r *http.Request) {
	week, err := s.store.CurrentUsageWeek()
	if err != nil {
		serverError(w, err)
		return
	}
	write(w, 200, week)
}
func (s *service) characterUsageHistory(w http.ResponseWriter, r *http.Request) {
	weeks, err := s.store.UsageHistory()
	if err != nil {
		serverError(w, err)
		return
	}
	write(w, 200, weeks)
}
func usageCSV(weeks []store.UsageWeek, rates bool) ([]byte, error) {
	ids := map[string]bool{}
	for _, d := range game.Definitions {
		ids[d.ID] = true
	}
	for _, week := range weeks {
		for id := range week.Counts {
			ids[id] = true
		}
	}
	columns := make([]string, 0, len(ids))
	for id := range ids {
		columns = append(columns, id)
	}
	sort.Strings(columns)
	var output bytes.Buffer
	writer := csv.NewWriter(&output)
	if err := writer.Write(append([]string{"week_start"}, columns...)); err != nil {
		return nil, err
	}
	for _, week := range weeks {
		row := []string{week.WeekStart}
		for _, id := range columns {
			cell := ""
			if count, ok := week.Counts[id]; ok {
				if rates {
					if rate := week.Rates[id]; rate != nil {
						cell = strconv.FormatFloat(*rate, 'f', 4, 64)
					}
				} else {
					cell = strconv.FormatUint(count, 10)
				}
			}
			row = append(row, cell)
		}
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	return output.Bytes(), writer.Error()
}
func (s *service) characterUsageCSV(rates bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		weeks, err := s.store.UsageHistory()
		if err != nil {
			serverError(w, err)
			return
		}
		data, err := usageCSV(weeks, rates)
		if err != nil {
			serverError(w, err)
			return
		}
		name := "counts"
		if rates {
			name = "rates"
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="character-usage-`+name+`.csv"`)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}
}
