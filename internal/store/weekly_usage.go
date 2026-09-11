package store

import (
	"encoding/json"
	"time"

	"auxilia-webserver/internal/game"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var japan = time.FixedZone("JST", 9*60*60)

// These records intentionally outlive the short-lived match history.
type UsageEvent struct {
	MatchID   string    `gorm:"primaryKey;size:64"`
	StartedAt time.Time `gorm:"index;not null"`
	PicksJSON string    `gorm:"type:text;not null"`
}

func (UsageEvent) TableName() string { return "web_usage_events" }

type UsageTracking struct {
	ID        string    `gorm:"primaryKey;size:32"`
	StartedAt time.Time `gorm:"not null"`
}

func (UsageTracking) TableName() string { return "web_usage_tracking" }

type UsageCharacter struct {
	ID          string    `gorm:"primaryKey;size:64"`
	AvailableAt time.Time `gorm:"not null"`
}

func (UsageCharacter) TableName() string { return "web_usage_characters" }

type WeeklyUsage struct {
	WeekStart       string    `gorm:"primaryKey;size:10" json:"weekStart"`
	PlayerPickCount uint64    `json:"playerPickCount"`
	CountsJSON      string    `gorm:"type:longtext;not null" json:"-"`
	SampledAt       time.Time `json:"sampledAt"`
}

func (WeeklyUsage) TableName() string { return "web_weekly_usage" }

type UsageWeek struct {
	WeekStart       string              `json:"weekStart"`
	WeekEnd         string              `json:"weekEnd"`
	PlayerPickCount uint64              `json:"playerPickCount"`
	Counts          map[string]uint64   `json:"counts"`
	Rates           map[string]*float64 `json:"rates"`
	Partial         bool                `json:"partial"`
	RecordingSince  *time.Time          `json:"recordingSince,omitempty"`
}

func WeekStart(t time.Time) time.Time {
	t = t.In(japan)
	day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, japan)
	return day.AddDate(0, 0, -(int(day.Weekday())+6)%7)
}
func firstFullWeek(start time.Time) time.Time {
	week := WeekStart(start)
	if start.After(week) {
		return week.AddDate(0, 0, 7)
	}
	return week
}
func (s *Store) initUsageTracking() error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&UsageTracking{ID: "global", StartedAt: now}).Error; err != nil {
			return err
		}
		for _, d := range game.Definitions {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&UsageCharacter{ID: d.ID, AvailableAt: now}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
func lockUsage(tx *gorm.DB) (UsageTracking, error) {
	var tracking UsageTracking
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&tracking, "id = ?", "global").Error
	return tracking, err
}
func recordUsageEvent(tx *gorm.DB, state *game.State) error {
	if state.TestOwnerID != "" {
		return nil
	}
	// Serialize the timestamp assignment with weekly finalization, avoiding late commits.
	if _, err := lockUsage(tx); err != nil {
		return err
	}
	picks := make([]string, 0, len(state.Characters))
	for _, c := range state.Characters {
		picks = append(picks, c.DefinitionID)
	}
	data, err := json.Marshal(picks)
	if err != nil {
		return err
	}
	return tx.Create(&UsageEvent{MatchID: state.MatchID, StartedAt: time.Now().UTC(), PicksJSON: string(data)}).Error
}
func usageWeek(start time.Time, counts map[string]uint64, total uint64) UsageWeek {
	rates := map[string]*float64{}
	for id, n := range counts {
		rates[id] = nil
		if total > 0 {
			rate := float64(n) * 100 / float64(total)
			rates[id] = &rate
		}
	}
	return UsageWeek{WeekStart: start.Format("2006-01-02"), WeekEnd: start.AddDate(0, 0, 7).Format("2006-01-02"), Counts: counts, Rates: rates, PlayerPickCount: total}
}
func collectWeek(tx *gorm.DB, start, end time.Time) (UsageWeek, error) {
	var catalog []UsageCharacter
	if err := tx.Where("available_at < ?", end.UTC()).Find(&catalog).Error; err != nil {
		return UsageWeek{}, err
	}
	counts := map[string]uint64{}
	for _, c := range catalog {
		counts[c.ID] = 0
	}
	var events []UsageEvent
	if err := tx.Where("started_at >= ? AND started_at < ?", start.UTC(), end.UTC()).Find(&events).Error; err != nil {
		return UsageWeek{}, err
	}
	for _, e := range events {
		var picks []string
		if err := json.Unmarshal([]byte(e.PicksJSON), &picks); err != nil {
			return UsageWeek{}, err
		}
		for _, id := range picks {
			counts[id]++
		}
	}
	return usageWeek(start, counts, uint64(len(events))*2), nil
}
func (s *Store) SampleWeeklyUsage() error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		tracking, err := lockUsage(tx)
		if err != nil {
			return err
		}
		end := WeekStart(time.Now())
		for start := firstFullWeek(tracking.StartedAt); start.Before(end); start = start.AddDate(0, 0, 7) {
			var count int64
			if err := tx.Model(&WeeklyUsage{}).Where("week_start = ?", start.Format("2006-01-02")).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				continue
			}
			week, err := collectWeek(tx, start, start.AddDate(0, 0, 7))
			if err != nil {
				return err
			}
			data, err := json.Marshal(week.Counts)
			if err != nil {
				return err
			}
			if err := tx.Create(&WeeklyUsage{WeekStart: week.WeekStart, PlayerPickCount: week.PlayerPickCount, CountsJSON: string(data), SampledAt: time.Now().UTC()}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
func (s *Store) CurrentUsageWeek() (UsageWeek, error) {
	var result UsageWeek
	err := s.db.Transaction(func(tx *gorm.DB) error {
		tracking, err := lockUsage(tx)
		if err != nil {
			return err
		}
		now := time.Now()
		start := WeekStart(now)
		result, err = collectWeek(tx, start, now)
		result.Partial = true
		result.RecordingSince = &tracking.StartedAt
		return err
	})
	return result, err
}
func (s *Store) UsageHistory() ([]UsageWeek, error) {
	if err := s.SampleWeeklyUsage(); err != nil {
		return nil, err
	}
	var rows []WeeklyUsage
	if err := s.db.Order("week_start ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]UsageWeek, 0, len(rows))
	for _, row := range rows {
		start, err := time.ParseInLocation("2006-01-02", row.WeekStart, japan)
		if err != nil {
			return nil, err
		}
		counts := map[string]uint64{}
		if err := json.Unmarshal([]byte(row.CountsJSON), &counts); err != nil {
			return nil, err
		}
		result = append(result, usageWeek(start, counts, row.PlayerPickCount))
	}
	return result, nil
}
