package store

import (
	"errors"

	"auxilia-webserver/internal/game"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func recordCharacterUsage(tx *gorm.DB, characters []game.Character) error {
	for _, character := range characters {
		usage := CharacterUsage{CharacterID: character.DefinitionID, UseCount: 1}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "character_id"}},
			DoUpdates: clause.Assignments(map[string]any{"use_count": gorm.Expr("use_count + ?", 1)}),
		}).Create(&usage).Error; err != nil {
			return err
		}
	}
	summary := UsageSummary{ID: "global", PlayerPickCount: 2}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.Assignments(map[string]any{"player_pick_count": gorm.Expr("player_pick_count + ?", 2)}),
	}).Create(&summary).Error
}

func (s *Store) CharacterUsageCounts() (map[string]uint64, uint64, error) {
	var rows []CharacterUsage
	if err := s.db.Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	counts := make(map[string]uint64, len(rows))
	for _, row := range rows {
		counts[row.CharacterID] = row.UseCount
	}
	var summary UsageSummary
	err := s.db.First(&summary, "id = ?", "global").Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return counts, 0, nil
	}
	if err != nil {
		return nil, 0, err
	}
	return counts, summary.PlayerPickCount, nil
}
