package game

import "testing"

func fixture() *State {
	return NewState("m1", [2]Player{{ID: "a", Name: "A"}, {ID: "b", Name: "B"}}, [2][]string{{"zina", "jude", "dana"}, {"sophie", "chiyo", "aoi"}})
}
func TestRejectsOpponentTurn(t *testing.T) {
	s := fixture()
	err := s.ApplyMove("b", Command{ExpectedRevision: s.Revision, CharacterID: "p2-c1", Target: Position{6, 1}})
	if err != ErrNotYourTurn {
		t.Fatalf("got %v", err)
	}
}
func TestRejectsOutOfRangeMove(t *testing.T) {
	s := fixture()
	err := s.ApplyMove("a", Command{ExpectedRevision: s.Revision, CharacterID: "p1-c1", Target: Position{6, 4}})
	if err != ErrInvalidAction {
		t.Fatalf("got %v", err)
	}
}
func TestServerCalculatesDamage(t *testing.T) {
	s := fixture()
	s.Characters[3].Position = Position{3, 1}
	err := s.ApplyAttack("a", Command{ExpectedRevision: s.Revision, CharacterID: "p1-c1", AttackIndex: 0, Target: Position{3, 1}})
	if err != nil {
		t.Fatal(err)
	}
	if s.Characters[3].HP != 130 {
		t.Fatalf("hp=%d", s.Characters[3].HP)
	}
}
func TestRejectsStaleRevision(t *testing.T) {
	s := fixture()
	old := s.Revision
	_ = s.ApplyMove("a", Command{ExpectedRevision: old, CharacterID: "p1-c1", Target: Position{1, 1}})
	if err := s.ApplyMove("a", Command{ExpectedRevision: old, CharacterID: "p1-c1", Target: Position{2, 1}}); err != ErrStaleRevision {
		t.Fatalf("got %v", err)
	}
}
