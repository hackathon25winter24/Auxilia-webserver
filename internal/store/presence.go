package store

import (
	"gorm.io/gorm/clause"
	"time"
)

type Presence struct {
	PlayerID   string    `gorm:"primaryKey;size:64"`
	LastSeenAt time.Time `gorm:"index;not null"`
}

func (Presence) TableName() string { return "web_presence" }
func (s *Store) Heartbeat(playerID string) error {
	return s.db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "player_id"}}, DoUpdates: clause.AssignmentColumns([]string{"last_seen_at"})}).Create(&Presence{PlayerID: playerID, LastSeenAt: time.Now().UTC()}).Error
}
func (s *Store) ActiveCount() (int64, error) {
	var count int64
	err := s.db.Model(&Presence{}).Where("last_seen_at > ?", time.Now().UTC().Add(-60*time.Second)).Count(&count).Error
	return count, err
}
func (s *Store) LeavePresence(playerID string) error {
	return s.db.Where("player_id = ?", playerID).Delete(&Presence{}).Error
}
