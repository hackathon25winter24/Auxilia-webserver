package game

import "testing"

func fixture() *State {
	return NewState("m1", [2]Player{{ID: "a", Name: "A"}, {ID: "b", Name: "B"}}, [2][]string{{"zina", "jude", "dana"}, {"sophie", "chiyo", "aoi"}})
}

func TestPendingMatchStartsAfterBothPlayersAreReady(t *testing.T) {
	s := NewPendingState("m1", [2]Player{{ID: "a", Name: "A"}, {ID: "b", Name: "B"}}, [2][]string{{"zina", "jude", "dana"}, {"sophie", "chiyo", "aoi"}})
	if s.Started {
		t.Fatal("pending match started before either player was ready")
	}
	if err := s.Ready("a"); err != nil {
		t.Fatal(err)
	}
	if s.Started || len(s.ReadyPlayerIDs) != 1 {
		t.Fatalf("started=%v ready=%v", s.Started, s.ReadyPlayerIDs)
	}
	if err := s.Ready("b"); err != nil {
		t.Fatal(err)
	}
	if !s.Started || s.TurnDeadline.IsZero() {
		t.Fatalf("started=%v deadline=%v", s.Started, s.TurnDeadline)
	}
}

func TestInitialPositionsUseBottomLeftOriginLayout(t *testing.T) {
	s := fixture()
	want := []Position{{0, 0}, {1, 2}, {0, 4}, {6, 2}, {7, 0}, {7, 4}}
	for i, position := range want {
		if s.Characters[i].Position != position {
			t.Fatalf("character %d position=%v want=%v", i, s.Characters[i].Position, position)
		}
	}
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
	s.Characters[3].Position = Position{3, 0}
	err := s.ApplyAttack("a", Command{ExpectedRevision: s.Revision, CharacterID: "p1-c1", AttackIndex: 0, Target: Position{3, 0}})
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

func TestServerDamagesBaseAndEndsMatch(t *testing.T) {
	s := fixture()
	s.Characters[0].Position = Position{4, 2}
	s.Characters[4].Position = Position{6, 3}
	s.Bases[1].HP = 20
	err := s.ApplyAttack("a", Command{ExpectedRevision: s.Revision, CharacterID: "p1-c1", AttackIndex: 0, Target: Position{7, 2}})
	if err != nil {
		t.Fatal(err)
	}
	if s.Bases[1].HP != 0 {
		t.Fatalf("base hp=%d", s.Bases[1].HP)
	}
	if !s.Finished || s.WinnerID != "a" {
		t.Fatalf("finished=%v winner=%q", s.Finished, s.WinnerID)
	}
}

func TestRejectsMovingOntoEnemyBase(t *testing.T) {
	s := fixture()
	s.Characters[0].Position = Position{6, 2}
	err := s.ApplyMove("a", Command{ExpectedRevision: s.Revision, CharacterID: "p1-c1", Target: Position{7, 2}})
	if err != ErrInvalidAction {
		t.Fatalf("got %v", err)
	}
}

func TestSurrenderAwardsWinToOpponent(t *testing.T) {
	s := fixture()
	if err := s.Surrender("a", s.Revision); err != nil {
		t.Fatal(err)
	}
	if !s.Finished || s.WinnerID != "b" {
		t.Fatalf("finished=%v winner=%q", s.Finished, s.WinnerID)
	}
}

func TestMineDealsDamageAndDisappears(t *testing.T) {
	s := NewState("m1", [2]Player{{ID: "a"}, {ID: "b"}}, [2][]string{{"berenice", "jude", "dana"}, {"sophie", "chiyo", "aoi"}})
	s.Characters[0].Position = Position{2, 2}
	if err := s.ApplyAttack("a", Command{ExpectedRevision: s.Revision, CharacterID: "p1-c1", AttackIndex: 0, Target: Position{3, 2}}); err != nil {
		t.Fatal(err)
	}
	s.advanceTurn("test")
	s.Characters[3].Position = Position{4, 2}
	before := s.Characters[3].HP
	if err := s.ApplyMove("b", Command{ExpectedRevision: s.Revision, CharacterID: "p2-c1", Target: Position{3, 2}}); err != nil {
		t.Fatal(err)
	}
	if s.Characters[3].HP != before-100 || len(s.TileEffects) != 0 {
		t.Fatalf("hp=%d tiles=%v", s.Characters[3].HP, s.TileEffects)
	}
}

func TestPoisonDealsFortyDamageAtTurnEnd(t *testing.T) {
	s := fixture()
	s.Characters[0].Effects = []string{"毒"}
	before := s.Characters[0].HP
	s.advanceTurn("test")
	if s.Characters[0].HP != before-40 {
		t.Fatalf("hp=%d", s.Characters[0].HP)
	}
}

func TestJudeReducesDamage(t *testing.T) {
	s := NewState("m1", [2]Player{{ID: "a"}, {ID: "b"}}, [2][]string{{"zina", "dana", "aoi"}, {"jude", "chiyo", "sophie"}})
	s.Characters[0].Position = Position{0, 0}
	s.Characters[3].Position = Position{3, 0}
	before := s.Characters[3].HP
	if err := s.ApplyAttack("a", Command{ExpectedRevision: s.Revision, CharacterID: "p1-c1", AttackIndex: 0, Target: Position{3, 0}}); err != nil {
		t.Fatal(err)
	}
	if s.Characters[3].HP != before {
		t.Fatalf("hp=%d", s.Characters[3].HP)
	}
}
