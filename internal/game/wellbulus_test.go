package game

import (
	"encoding/json"
	"reflect"
	"testing"
)

func wellbulusFixture() *State {
	s := NewState("wellbulus", [2]Player{{ID: "a"}, {ID: "b"}}, [2][]string{{"wellbulus", "sophie", "jude"}, {"dana", "chiyo", "zina"}})
	s.TurnPlayerID = "a"
	positions := []Position{{3, 2}, {3, 3}, {1, 0}, {4, 2}, {4, 3}, {7, 4}}
	for i := range s.Characters {
		s.Characters[i].Position = positions[i]
	}
	return s
}

func wellbulusAct(s *State, attack int, target Position) error {
	return s.ApplyAttack("a", Command{CharacterID: s.Characters[0].ID, ExpectedRevision: s.Revision, AttackIndex: attack, Target: target, Direction: Position{1, 0}})
}

func TestWellbulusDispelAndHeal(t *testing.T) {
	s := wellbulusFixture()
	for i := range s.Characters {
		s.Characters[i].Effects = []string{"威力上昇", "俊足", "俊敏化", "毒"}
	}
	before := s.Characters[3].HP
	if err := wellbulusAct(s, 0, Position{4, 2}); err != nil {
		t.Fatal(err)
	}
	if s.Characters[3].HP != before || !reflect.DeepEqual(s.Characters[3].Effects, []string{"毒"}) {
		t.Fatal("dispel damaged or failed to remove buffs")
	}
	for _, i := range []int{0, 1, 4} {
		if !s.hasEffect(i, "威力上昇") {
			t.Fatalf("non-target %d lost buff", i)
		}
	}
	if s.LastEvent.Type != "BUFFS_CLEARED" || s.cost("a") != 32 {
		t.Fatal("wrong dispel event/cost")
	}

	s = wellbulusFixture()
	for i := range s.Characters {
		s.Characters[i].HP = 50
	}
	if err := wellbulusAct(s, 2, Position{3, 2}); err != nil {
		t.Fatal(err)
	}
	for i, want := range []int{110, 100, 50, 50, 50, 50} {
		if s.Characters[i].HP != want {
			t.Errorf("heal target %d hp=%d want=%d", i, s.Characters[i].HP, want)
		}
	}
	if s.cost("a") != 25 || s.LastEvent.Type != "RECOVERED" {
		t.Fatal("wrong heal event/cost")
	}
}

func TestImmutablePlacementAndPersistence(t *testing.T) {
	s := wellbulusFixture()
	if err := wellbulusAct(s, 1, Position{4, 2}); err != nil {
		t.Fatal(err)
	}
	if len(s.TileEffects) != 1 || s.TileEffects[0].HP != 120 || s.cost("a") != 20 {
		t.Fatal("placement failed")
	}
	raw, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var restored State
	if err = json.Unmarshal(raw, &restored); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s.TileEffects, restored.TileEffects) {
		t.Fatal("tile HP not persisted")
	}
	s.TurnPlayerID = "b"
	if err := s.ApplyMove("b", Command{CharacterID: s.Characters[3].ID, ExpectedRevision: s.Revision, Target: Position{4, 1}}); err == nil {
		t.Fatal("trapped character escaped")
	}
	s.Characters[3].DefinitionID = "tsukiha"
	if err := s.ApplyMove("b", Command{CharacterID: s.Characters[3].ID, ExpectedRevision: s.Revision, Target: Position{4, 1}}); err != nil {
		t.Fatalf("tile immune character cannot escape: %v", err)
	}
	if err := s.ApplyMove("b", Command{CharacterID: s.Characters[3].ID, ExpectedRevision: s.Revision, Target: Position{4, 2}}); err == nil {
		t.Fatal("entered impassable tile")
	}
}

func TestImmutableCannotBeOverwrittenOrPlacedOnForbiddenCells(t *testing.T) {
	for _, kind := range []string{"base", "blocked", "immutable"} {
		s := wellbulusFixture()
		switch kind {
		case "base":
			s.Bases[1].Position = Position{4, 2}
		case "blocked":
			s.BlockedCells = append(s.BlockedCells, Position{4, 2})
		case "immutable":
			s.setTile(Position{4, 2}, "不変", "b")
		}
		if err := wellbulusAct(s, 1, Position{4, 2}); err == nil || s.cost("a") != 50 {
			t.Fatalf("invalid placement accepted: %s", kind)
		}
	}
	for _, id := range []string{"dana", "tsukiha", "berenice"} {
		s := wellbulusFixture()
		s.Characters[0].DefinitionID = id
		s.Characters[3].Position = Position{6, 2}
		s.setTile(Position{4, 2}, "不変", "b")
		attack := 0
		if id == "tsukiha" {
			attack = 2
		}
		if err := wellbulusAct(s, attack, Position{4, 2}); err == nil {
			t.Fatalf("%s overwrote immutable tile", id)
		}
	}
	s := wellbulusFixture()
	s.setTile(Position{4, 2}, "毒ガス", "b")
	if err := wellbulusAct(s, 1, Position{4, 2}); err != nil || s.TileEffects[0].Type != "不変" {
		t.Fatal("cannot replace ordinary debuff tile")
	}
}

func TestImmutableDecayAndDestruction(t *testing.T) {
	s := wellbulusFixture()
	s.setTile(Position{4, 2}, "不変", "a")
	before := s.Characters[3].HP
	for i, player := range []string{"a", "b", "a"} {
		s.processTurnEnd(player)
		if i < 2 && s.TileEffects[0].HP != 120-50*(i+1) {
			t.Fatal("wrong decay")
		}
	}
	if len(s.TileEffects) != 0 || s.Characters[3].HP != before {
		t.Fatal("decay did not destroy only tile")
	}
	s.TurnPlayerID = "b"
	if err := s.ApplyMove("b", Command{CharacterID: s.Characters[3].ID, ExpectedRevision: s.Revision, Target: Position{4, 1}}); err != nil {
		t.Fatal("still trapped after decay")
	}
}

func TestImmutableCanBeAttackedByEitherSide(t *testing.T) {
	for _, owner := range []string{"a", "b"} {
		s := wellbulusFixture()
		s.Characters[0].DefinitionID = "chiyo"
		s.Characters[0].HP = 100                  // no full HP bonus
		s.Characters[1].Position = Position{0, 4} // no Sophie aura
		s.Characters[3].Position = Position{6, 2} // empty tile can be targeted
		s.setTile(Position{4, 2}, "不変", owner)
		if err := wellbulusAct(s, 1, Position{4, 2}); err != nil {
			t.Fatal(err)
		}
		if s.TileEffects[0].HP != 60 {
			t.Fatal("wrong attack damage")
		}
		if err := wellbulusAct(s, 1, Position{4, 2}); err != nil {
			t.Fatal(err)
		}
		if len(s.TileEffects) != 0 {
			t.Fatal("tile survived lethal damage")
		}
	}
}

func TestClearBuffsPreservesDebuffs(t *testing.T) {
	s := fixture()
	s.Characters[0].Effects = []string{"威力上昇", "毒", "俊足", "麻痺", "俊敏化", "出血", "鈍足", "鈍化"}
	s.clearBuffs(0)
	for _, effect := range []string{"威力上昇", "俊足", "俊敏化"} {
		if s.hasEffect(0, effect) {
			t.Errorf("buff retained: %s", effect)
		}
	}
	for _, effect := range []string{"毒", "麻痺", "出血", "鈍足", "鈍化"} {
		if !s.hasEffect(0, effect) {
			t.Errorf("debuff removed: %s", effect)
		}
	}
}

func TestImmutableDoesNotPreventActionsAndReceivesAreaDamage(t *testing.T) {
	s := wellbulusFixture()
	s.setTile(s.Characters[0].Position, "不変", "b")
	s.Characters[0].HP = 50
	if err := wellbulusAct(s, 2, s.Characters[0].Position); err != nil || s.Characters[0].HP != 110 {
		t.Fatal("trapped character cannot heal")
	}
	s = wellbulusFixture()
	s.Characters[0].DefinitionID = "chiyo"
	s.Characters[0].HP = 100
	s.Characters[1].Position = Position{0, 4}
	s.setTile(Position{4, 2}, "不変", "b")
	before := s.Characters[3].HP
	if err := wellbulusAct(s, 1, Position{4, 2}); err != nil {
		t.Fatal(err)
	}
	if s.Characters[3].HP != before-60 || s.TileEffects[0].HP != 60 {
		t.Fatal("attack must damage both enemy and tile")
	}
}
