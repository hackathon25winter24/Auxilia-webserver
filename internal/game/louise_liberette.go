package game

import (
	"fmt"
	"hash/fnv"
)

func described(a AttackDefinition, text string) AttackDefinition { a.Description = text; return a }
func init() {
	Definitions = append(Definitions,
		CharacterDefinition{ID: "louise", Name: "ルイース", Image: "Louise_mini.png", Portrait: "Louise.png", MaxHP: 100, MoveCost: 5, MoveRange: 1,
			PassiveName: "援護の舞踏", PassiveDescription: "周囲1マス以内の自身以外の味方のパッシブ効果値を20上昇させる。",
			Attacks: [3]AttackDefinition{
				described(atk("退魔の舞", 30, 0, "ally", p(Position{0, 0})), "味方全体のデバフを解除し、免疫を付与する。"),
				described(atk("守護の舞", 30, 0, "ally", p(Position{0, 0})), "自身以外の味方全体を50回復し、自身を含む味方全体に結界を付与する。"),
				described(atk("降納", 25, 0, "ally", p(Position{0, 0})), "自身を臨戦状態にする。"),
			}, AlternateAttacks: &[3]AttackDefinition{
				atk("突撃の踏", 20, 70, "enemy", p(Position{1, 0}, Position{2, 0}, Position{3, 0})),
				atk("薙ぎの踏", 20, 70, "enemy", p(Position{1, -1}, Position{1, 0}, Position{1, 1})),
				described(atk("掲揚", 25, 0, "ally", p(Position{0, 0})), "自身を支援状態にする。"),
			}},
		CharacterDefinition{ID: "liberette", Name: "リベレット", Image: "Liberette_mini.png", Portrait: "Liberette.png", MaxHP: 150, MoveCost: 8, MoveRange: 1,
			PassiveName: "スペードのエース", PassiveDescription: "自身のターン開始時、生存中の敵味方全員（自身を含む）から1人を選び、ランダムなバフまたはデバフを1つ付与する。",
			Attacks: [3]AttackDefinition{
				func() AttackDefinition {
					a := atk("ハートの４", 25, -30, "ally", p(Position{0, 0}))
					a.ClearDebuffs = true
					return a
				}(),
				described(atk("クラブの８", 10, 0, "ally", p(Position{0, 0})), "自身にランダムなバフを1つ付与する。"),
				described(atk("ダイヤの10", 20, 30, "enemy", p(Position{2, -1}, Position{1, 0}, Position{2, 0}, Position{3, 0}, Position{2, 1})), "自身にバフがあればダメージ+60。攻撃後、自身の全バフを解除する。"),
			}})
	passiveValues["louise"] = PassiveValues{PassiveValueBoost: 20}
}

var randomBuffs = []string{"威力上昇", "俊足", "俊敏化", "免疫", "結界"}
var randomDebuffs = []string{"毒", "麻痺", "鈍足", "鈍化", "出血"}

func isBuff(effect string) bool {
	for _, e := range randomBuffs {
		if e == effect {
			return true
		}
	}
	return false
}
func (s *State) hasBuff(i int) bool {
	for _, e := range s.Characters[i].Effects {
		if isBuff(e) {
			return true
		}
	}
	return false
}
func (s *State) removeEffect(i int, effect string) {
	kept := s.Characters[i].Effects[:0]
	for _, e := range s.Characters[i].Effects {
		if e != effect {
			kept = append(kept, e)
		}
	}
	s.Characters[i].Effects = kept
}
func (s *State) randomIndex(actor int, salt string, n int) int {
	h := fnv.New64a()
	fmt.Fprintf(h, "%s:%d:%d:%s:%s", s.MatchID, s.Revision, s.Turn, s.Characters[actor].ID, salt)
	return int(h.Sum64() % uint64(n))
}
func (s *State) randomEffect(actor int, includeDebuffs bool) string {
	effects := append([]string{}, randomBuffs...)
	if includeDebuffs {
		effects = append(effects, randomDebuffs...)
	}
	return effects[s.randomIndex(actor, "effect", len(effects))]
}
func (s *State) applyTurnStartPassives() {
	for i, c := range s.Characters {
		if c.BarrierTurn < s.Turn {
			s.removeEffect(i, "結界")
		}
	}
	for i, c := range s.Characters {
		if c.HP <= 0 || c.OwnerID != s.TurnPlayerID || c.DefinitionID != "liberette" {
			continue
		}
		var targets []int
		for j, t := range s.Characters {
			if t.HP > 0 {
				targets = append(targets, j)
			}
		}
		if len(targets) == 0 {
			continue
		}
		j := targets[s.randomIndex(i, "target", len(targets))]
		effect := s.randomEffect(i, true)
		s.addEffect(j, effect)
	}
}
func (s *State) applyLouiseSkill(actor, attack int) {
	if attack == 2 {
		s.Characters[actor].CombatStance = !s.Characters[actor].CombatStance
		return
	}
	for j, c := range s.Characters {
		if c.HP <= 0 || c.OwnerID != s.Characters[actor].OwnerID {
			continue
		}
		if attack == 0 {
			s.clearDebuffs(j)
			s.addEffect(j, "免疫")
		} else {
			if j != actor {
				s.Characters[j].HP = min(c.MaxHP, c.HP+50)
			}
			s.addEffect(j, "結界")
		}
	}
}
func (s *State) consumeBarrier(i int) bool {
	if s.hasEffect(i, "結界") && s.Characters[i].BarrierTurn == s.Turn {
		s.removeEffect(i, "結界")
		return true
	}
	return false
}
