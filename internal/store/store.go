package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"auxilia-webserver/internal/game"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	ID        string `gorm:"primaryKey;size:64"`
	Revision  uint64 `gorm:"not null"`
	Status    string `gorm:"size:16;index;not null"`
	StateJSON string `gorm:"type:longtext;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Match) TableName() string { return "web_matches" }

type ProcessedCommand struct {
	ID        string    `gorm:"primaryKey;size:180"`
	MatchID   string    `gorm:"size:64;index;not null"`
	CreatedAt time.Time `gorm:"index"`
}

func (ProcessedCommand) TableName() string { return "web_processed_commands" }

type Store struct{ db *gorm.DB }

func New(db *gorm.DB) (*Store, error) {
	if err := db.AutoMigrate(&Guest{}, &Match{}, &ProcessedCommand{}); err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *Store) CreateGuest(id, name, token string) (*Guest, error) {
	g := &Guest{ID: id, Name: name, TokenHash: HashToken(token), SelectionJSON: "[]", ExpiresAt: time.Now().Add(24 * time.Hour)}
	return g, s.db.Create(g).Error
}
func (s *Store) GuestByToken(token string) (*Guest, error) {
	var g Guest
	if err := s.db.Where("token_hash = ? AND expires_at > ?", HashToken(token), time.Now()).First(&g).Error; err != nil {
		return nil, normalize(err)
	}
	return &g, nil
}
func (s *Store) GuestByID(id string) (*Guest, error) {
	var g Guest
	if err := s.db.First(&g, "id = ?", id).Error; err != nil {
		return nil, normalize(err)
	}
	return &g, nil
}

func (s *Store) SetSelection(id string, selection []string) (*Guest, error) {
	data, _ := json.Marshal(selection)
	result := s.db.Model(&Guest{}).Where("id = ? AND queued = ? AND match_id = ''", id, false).Update("selection_json", string(data))
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected != 1 {
		return nil, errors.New("マッチング中は変更できません")
	}
	return s.GuestByID(id)
}

func (s *Store) EnqueueAndMatch(id, newMatchID string) (*Guest, error) {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var self Guest
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&self, "id = ?", id).Error; err != nil {
			return err
		}
		if self.MatchID != "" {
			return nil
		}
		if len(self.Selection()) != 3 {
			return errors.New("キャラクターを3体選択してください")
		}
		if !self.Queued {
			if err := tx.Model(&self).Updates(map[string]any{"queued": true, "queued_at": time.Now()}).Error; err != nil {
				return err
			}
		}
		var waiting []Guest
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("queued = ? AND match_id = ''", true).Order("queued_at ASC").Limit(2).Find(&waiting).Error; err != nil {
			return err
		}
		if len(waiting) < 2 {
			return nil
		}
		players := [2]game.Player{{ID: waiting[0].ID, Name: waiting[0].Name}, {ID: waiting[1].ID, Name: waiting[1].Name}}
		selections := [2][]string{waiting[0].Selection(), waiting[1].Selection()}
		state := game.NewState(newMatchID, players, selections)
		data, err := json.Marshal(state)
		if err != nil {
			return err
		}
		if err := tx.Create(&Match{ID: newMatchID, Revision: state.Revision, Status: "active", StateJSON: string(data)}).Error; err != nil {
			return err
		}
		ids := []string{waiting[0].ID, waiting[1].ID}
		return tx.Model(&Guest{}).Where("id IN ?", ids).Updates(map[string]any{"queued": false, "match_id": newMatchID}).Error
	})
	if err != nil {
		return nil, err
	}
	return s.GuestByID(id)
}

func (s *Store) CancelQueue(id string) (*Guest, error) {
	result := s.db.Model(&Guest{}).Where("id = ? AND match_id = ''", id).Update("queued", false)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected != 1 {
		return nil, errors.New("開始済みの試合は解除できません")
	}
	return s.GuestByID(id)
}

func (s *Store) LoadState(matchID, guestID string) (*game.State, error) {
	var state *game.State
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var g Guest
		if err := tx.First(&g, "id = ? AND match_id = ?", guestID, matchID).Error; err != nil {
			return err
		}
		var m Match
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&m, "id = ?", matchID).Error; err != nil {
			return err
		}
		parsed, err := decodeState(m.StateJSON)
		if err != nil {
			return err
		}
		before := parsed.Revision
		parsed.ExpireTurn(time.Now())
		if parsed.Revision != before {
			if err := saveState(tx, &m, parsed); err != nil {
				return err
			}
		}
		state = parsed
		return nil
	})
	return state, normalize(err)
}

func (s *Store) Apply(matchID, guestID, commandID string, apply func(*game.State) error) (*game.State, error) {
	var state *game.State
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var g Guest
		if err := tx.First(&g, "id = ? AND match_id = ?", guestID, matchID).Error; err != nil {
			return err
		}
		var m Match
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&m, "id = ?", matchID).Error; err != nil {
			return err
		}
		parsed, err := decodeState(m.StateJSON)
		if err != nil {
			return err
		}
		parsed.ExpireTurn(time.Now())
		key := matchID + ":" + guestID + ":" + commandID
		var count int64
		if err := tx.Model(&ProcessedCommand{}).Where("id = ?", key).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			state = parsed
			return nil
		}
		if err := apply(parsed); err != nil {
			return err
		}
		if err := saveState(tx, &m, parsed); err != nil {
			return err
		}
		if err := tx.Create(&ProcessedCommand{ID: key, MatchID: matchID}).Error; err != nil {
			return err
		}
		state = parsed
		return nil
	})
	return state, normalize(err)
}

func (s *Store) Cleanup(now time.Time) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("created_at < ?", now.Add(-24*time.Hour)).Delete(&ProcessedCommand{}).Error; err != nil {
			return err
		}
		if err := tx.Where("expires_at < ?", now).Delete(&Guest{}).Error; err != nil {
			return err
		}
		return tx.Where("status = ? AND updated_at < ?", "finished", now.Add(-7*24*time.Hour)).Delete(&Match{}).Error
	})
}
func decodeState(raw string) (*game.State, error) {
	var state game.State
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return nil, err
	}
	return &state, nil
}
func saveState(tx *gorm.DB, m *Match, state *game.State) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	status := "active"
	if state.Finished {
		status = "finished"
	}
	return tx.Model(m).Updates(map[string]any{"state_json": string(data), "revision": state.Revision, "status": status}).Error
}
func normalize(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}
