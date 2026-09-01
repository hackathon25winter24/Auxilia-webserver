package game

import "testing"

func TestCharacterDefinitionsMatchCharacterSpec(t *testing.T) {
	type attackNumbers struct {
		cost, power, effectChance int
	}
	type characterNumbers struct {
		hp, moveCost int
		attacks      [3]attackNumbers
	}

	// 仕様書/キャラ.md に記載された数値。回復量は内部表現に合わせて負数で表す。
	want := map[string]characterNumbers{
		"sophie":   {100, 10, [3]attackNumbers{{10, 10, 0}, {20, 50, 0}, {50, 250, 0}}},
		"jude":     {350, 10, [3]attackNumbers{{10, 20, 30}, {20, 50, 0}, {30, -30, 0}}},
		"nadia":    {200, 7, [3]attackNumbers{{10, 20, 20}, {20, 40, 40}, {30, 60, 60}}},
		"tsukiha":  {100, 3, [3]attackNumbers{{5, 15, 10}, {5, 10, 10}, {10, 0, 0}}},
		"aoi":      {250, 8, [3]attackNumbers{{10, 20, 0}, {20, 40, 0}, {30, 60, 0}}},
		"sena":     {250, 10, [3]attackNumbers{{20, 60, 0}, {25, 40, 0}, {40, 90, 0}}},
		"berenice": {200, 6, [3]attackNumbers{{15, 0, 0}, {25, 60, 0}, {20, 40, 0}}},
		"chiyo":    {100, 5, [3]attackNumbers{{10, 30, 0}, {20, 60, 50}, {50, 220, 0}}},
		"shincho":  {50, 15, [3]attackNumbers{{20, 250, 0}, {15, -40, 0}, {15, 0, 0}}},
		"zina":     {150, 5, [3]attackNumbers{{20, 20, 30}, {20, 20, 60}, {30, 60, 0}}},
		"dana":     {250, 10, [3]attackNumbers{{10, 0, 0}, {20, 20, 50}, {20, -30, 0}}},
	}

	if len(Definitions) != len(want) {
		t.Fatalf("character count = %d, want %d", len(Definitions), len(want))
	}
	for id, expected := range want {
		got, ok := Definition(id)
		if !ok {
			t.Errorf("definition %q is missing", id)
			continue
		}
		if got.MaxHP != expected.hp || got.MoveCost != expected.moveCost {
			t.Errorf("%s base stats = HP %d / move cost %d, want HP %d / move cost %d", id, got.MaxHP, got.MoveCost, expected.hp, expected.moveCost)
		}
		for i, attack := range got.Attacks {
			attackWant := expected.attacks[i]
			if attack.Cost != attackWant.cost || attack.Power != attackWant.power || attack.EffectChance != attackWant.effectChance {
				t.Errorf("%s attack %d = cost %d / power %d / chance %d, want %d / %d / %d", id, i+1, attack.Cost, attack.Power, attack.EffectChance, attackWant.cost, attackWant.power, attackWant.effectChance)
			}
		}
	}
}

func TestPassiveNumericValuesMatchCharacterSpec(t *testing.T) {
	tests := map[string]PassiveValues{
		"sophie":  {AttackBoost: 20, ExcludeSelf: true},
		"jude":    {DamageReduction: 20},
		"nadia":   {ExtraEffectChance: 20},
		"aoi":     {TurnHeal: 30},
		"sena":    {IgnorePassiveReduce: true},
		"chiyo":   {FullHPAttackBoost: 50},
		"shincho": {AttackBoost: 10, TurnHeal: 10},
		"zina":    {PassiveValueBoost: 20},
	}
	for id, want := range tests {
		if got := passiveFor(id); got != want {
			t.Errorf("%s passive values = %+v, want %+v", id, got, want)
		}
	}
}
