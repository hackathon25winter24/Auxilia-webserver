package store

import (
	"auxilia-webserver/internal/game"
	"encoding/json"
	"testing"
)

func TestTestModeUsageNeverTouchesDatabaseIncludingAfterReload(t *testing.T) {
	state := game.NewTestState("test", "owner", "tester", []string{"sophie", "jude", "dana"})
	raw, _ := json.Marshal(state)
	restored, err := decodeState(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	// A nil DB deliberately fails if the test match attempts any counter writes.
	for _, s := range []*game.State{state, restored} {
		if err := recordCharacterUsage(nil, s); err != nil {
			t.Fatal(err)
		}
	}
}
