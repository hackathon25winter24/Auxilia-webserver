package store

import (
	"testing"
	"time"
)

func TestLegacyCharacterData(t *testing.T) {
	g := Guest{SelectionJSON: `["wellbulus","shincho","zina"]`}
	if got := g.Selection(); got[0] != "verbulus" || got[1] != "shicho" {
		t.Fatal(got)
	}
	s, err := decodeState(`{"characters":[{"definitionId":"shincho"},{"definitionId":"wellbulus"}]}`)
	if err != nil || s.Characters[0].DefinitionID != "shicho" || s.Characters[1].DefinitionID != "verbulus" {
		t.Fatal(s, err)
	}
	w := usageWeek(time.Now(), map[string]uint64{"wellbulus": 2, "verbulus": 3, "shincho": 1}, 10)
	if w.Counts["verbulus"] != 5 || w.Counts["shicho"] != 1 || len(w.Counts) != 2 {
		t.Fatal(w.Counts)
	}
}
