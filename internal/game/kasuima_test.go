package game

import (
	"encoding/json"
	"testing"
	"time"
)

func kasuimaState() *State {
	s := NewState("kasuima", [2]Player{{ID: "a"}, {ID: "b"}}, [2][]string{{"kasuima"}, {"dana"}})
	s.TurnPlayerID = "a"
	s.Characters[0].Position = Position{2, 2}
	s.Characters[1].Position = Position{3, 2}
	return s
}
func kasuimaAttack(s *State, attack int) error {
	target := s.Characters[1].Position
	if attack == 0 {
		target = s.Characters[0].Position
	}
	return s.ApplyAttack("a", Command{ExpectedRevision: s.Revision, CharacterID: s.Characters[0].ID, AttackIndex: attack, Target: target, Direction: Position{1, 0}})
}
func nextKasuimaTurn(t *testing.T, s *State) {
	t.Helper()
	if err := s.EndTurn(s.TurnPlayerID, s.Revision); err != nil {
		t.Fatal(err)
	}
	s.ExpireTurn(s.PhaseDeadline.Add(time.Millisecond))
}
func TestKasuimaDrinkLifecycleAndImmunity(t *testing.T) {
	s := kasuimaState()
	for _, buff := range []string{"威力上昇", "俊足", "俊敏化"} {
		s.addEffect(0, buff)
		if s.hasEffect(0, buff) {
			t.Fatal("external buff accepted")
		}
	}
	if err := kasuimaAttack(s, 0); err != nil {
		t.Fatal(err)
	}
	if !s.hasEffect(0, "威力上昇") || s.attackPower(0, 100) != 125 || s.cost("a") != 40 {
		t.Fatal("drink failed")
	}
	nextKasuimaTurn(t, s)
	if s.hasEffect(0, "威力上昇") || s.hasEffect(0, "二日酔い") {
		t.Fatal("incorrect opponent-turn effects")
	}
	raw, _ := json.Marshal(s)
	if err := json.Unmarshal(raw, s); err != nil {
		t.Fatal(err)
	}
	nextKasuimaTurn(t, s)
	if !s.hasEffect(0, "二日酔い") || s.attackPower(0, 100) != 80 {
		t.Fatal("hangover missing")
	}
	if err := s.ApplyMove("a", Command{ExpectedRevision: s.Revision, CharacterID: s.Characters[0].ID, Target: Position{2, 1}}); err != nil {
		t.Fatal(err)
	}
	if s.cost("a") != 30 {
		t.Fatal("hangover movement cost")
	}
	if err := kasuimaAttack(s, 0); err != nil {
		t.Fatal(err)
	}
	if s.cost("a") != 15 {
		t.Fatal("hangover attack cost")
	}
	nextKasuimaTurn(t, s)
	if s.hasEffect(0, "二日酔い") || s.hasEffect(0, "威力上昇") {
		t.Fatal("temporary effects did not expire")
	}
	nextKasuimaTurn(t, s)
	if !s.hasEffect(0, "二日酔い") {
		t.Fatal("repeat drink lost next hangover")
	}
	s.clearDebuffs(0)
	if s.hasEffect(0, "二日酔い") {
		t.Fatal("hangover cannot be cleansed")
	}
}
func TestKasuimaReverseFallbackAndLanding(t *testing.T) {
	for _, tt := range []struct {
		name    string
		blocked []Position
		want    Position
	}{
		{"two", nil, Position{5, 2}},
		{"one", []Position{{5, 2}}, Position{4, 2}},
		{"none", []Position{{5, 2}, {4, 2}}, Position{3, 2}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := kasuimaState()
			s.BlockedCells = tt.blocked
			if err := kasuimaAttack(s, 2); err != nil {
				t.Fatal(err)
			}
			if s.Characters[1].Position != tt.want || s.Characters[1].HP != 195 {
				t.Fatalf("wrong push %+v", s.Characters[1])
			}
		})
	}
	s := kasuimaState()
	s.setTile(Position{5, 2}, "地雷", "a")
	if err := kasuimaAttack(s, 2); err != nil {
		t.Fatal(err)
	}
	if s.Characters[1].HP != 95 || len(s.TileEffects) != 0 {
		t.Fatal("landing did not trigger mine")
	}
}
