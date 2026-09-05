package store

import (
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("not found")

type Guest struct {
	ID            string     `gorm:"primaryKey;size:64"`
	Name          string     `gorm:"size:80;not null"`
	TokenHash     string     `gorm:"size:64;uniqueIndex;not null"`
	SelectionJSON string     `gorm:"type:text;not null"`
	MatchID       string     `gorm:"size:64;index"`
	Queued        bool       `gorm:"index:idx_queue,priority:1"`
	QueuedAt      *time.Time `gorm:"index:idx_queue,priority:2"`
	ExpiresAt     time.Time  `gorm:"index"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (Guest) TableName() string { return "web_guests" }
func (g Guest) Selection() []string {
	var result []string
	_ = json.Unmarshal([]byte(g.SelectionJSON), &result)
	return result
}

type Match struct {
	ID           string `gorm:"primaryKey;size:64"`
	Revision     uint64 `gorm:"not null"`
	Status       string `gorm:"size:16;index;not null"`
	StateJSON    string `gorm:"type:longtext;not null"`
	UsageCounted bool   `gorm:"not null;default:false"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (Match) TableName() string { return "web_matches" }

type ProcessedCommand struct {
	ID        string    `gorm:"primaryKey;size:180"`
	MatchID   string    `gorm:"size:64;index;not null"`
	CreatedAt time.Time `gorm:"index"`
}

func (ProcessedCommand) TableName() string { return "web_processed_commands" }

type CharacterUsage struct {
	CharacterID string `gorm:"primaryKey;size:64"`
	UseCount    uint64 `gorm:"not null;default:0"`
	UpdatedAt   time.Time
}

func (CharacterUsage) TableName() string { return "web_character_usages" }

type UsageSummary struct {
	ID              string `gorm:"primaryKey;size:32"`
	PlayerPickCount uint64 `gorm:"not null;default:0"`
	UpdatedAt       time.Time
}

func (UsageSummary) TableName() string { return "web_usage_summaries" }

type Store struct{ db *gorm.DB }

func New(db *gorm.DB) (*Store, error) {
	if err := db.AutoMigrate(&Guest{}, &Match{}, &ProcessedCommand{}, &CharacterUsage{}, &UsageSummary{}); err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}
