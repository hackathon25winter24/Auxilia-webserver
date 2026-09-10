package store

import (
	"auxilia-webserver/internal/game"
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Store) CreateTestMatch(guestID, matchID string) (*Guest, error) {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var g Guest
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&g, "id = ?", guestID).Error; err != nil {
			return err
		}
		if g.Queued || g.MatchID != "" {
			return errors.New("マッチング中・対戦中はテストモードを開始できません")
		}
		selection := g.Selection()
		if len(selection) != 3 {
			return errors.New("キャラクターを3体選択してください")
		}
		seen := map[string]bool{}
		for _, id := range selection {
			if _, ok := game.Definition(id); !ok || seen[id] {
				return errors.New("異なるキャラクターを3体選択してください")
			}
			seen[id] = true
		}
		state := game.NewTestState(matchID, g.ID, g.Name, selection)
		data, err := json.Marshal(state)
		if err != nil {
			return err
		}
		// Test matches never call recordCharacterUsage, including when reloaded.
		if err := tx.Create(&Match{ID: matchID, Revision: state.Revision, Status: "active", StateJSON: string(data)}).Error; err != nil {
			return err
		}
		return tx.Model(&g).Updates(map[string]any{"match_id": matchID, "queued": false, "queued_at": nil}).Error
	})
	if err != nil {
		return nil, normalize(err)
	}
	return s.GuestByID(guestID)
}
