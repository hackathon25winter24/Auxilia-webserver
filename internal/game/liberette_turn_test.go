package game

import "testing"

func TestLiberetteBothSidesActivateOnEitherTurn(t *testing.T) {
	for _, owner := range []string{"a", "b"} {
		t.Run(owner, func(t *testing.T) {
			s := NewPendingState("both-liberettes", [2]Player{{ID: "a"}, {ID: "b"}}, [2][]string{{"liberette", "jude", "tsukiha"}, {"liberette", "jude", "tsukiha"}})
			s.Turn = 1
			s.TurnPlayerID = owner
			// Use distinct recipients so stacking and immunity cannot hide an activation.
			for s.randomIndex(0, "target", 6) == s.randomIndex(3, "target", 6) {
				s.Revision++
				if s.Revision > 1000 {
					t.Fatal("could not find distinct recipients")
				}
			}
			targets := [2]int{s.randomIndex(0, "target", 6), s.randomIndex(3, "target", 6)}
			effects := [2]string{s.randomEffect(0, true), s.randomEffect(3, true)}
			s.applyTurnStartPassives()
			for i, target := range targets {
				if !s.hasEffect(target, effects[i]) {
					t.Fatalf("source %d did not activate on %s turn", i, owner)
				}
			}
			total := 0
			for _, c := range s.Characters {
				total += len(c.Effects)
			}
			if total != 2 {
				t.Fatalf("got %d effects, want 2", total)
			}
		})
	}
}

func TestLiberetteKnockedOutSourceDoesNotActivate(t *testing.T) {
	s := NewPendingState("one-liberette", [2]Player{{ID: "a"}, {ID: "b"}}, [2][]string{{"liberette", "jude", "tsukiha"}, {"liberette", "jude", "tsukiha"}})
	s.Turn = 1
	s.TurnPlayerID = "a"
	s.Characters[0].HP = 0
	s.applyTurnStartPassives()
	total := 0
	for _, c := range s.Characters {
		total += len(c.Effects)
	}
	if total != 1 {
		t.Fatalf("got %d effects, want living enemy's one activation", total)
	}
	if len(s.Characters[0].Effects) != 0 {
		t.Fatal("knocked-out recipient selected")
	}
}
