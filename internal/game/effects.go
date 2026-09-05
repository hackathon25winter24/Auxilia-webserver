package game

import (
	"fmt"
	"hash/fnv"
)

func (s *State) hasEffect(character int, effect string) bool {
	for _, v := range s.Characters[character].Effects {
		if v == effect {
			return true
		}
	}
	return false
}
func (s *State) addEffect(character int, effect string) {
	if s.Characters[character].DefinitionID == "dana" {
		return
	}
	if !s.hasEffect(character, effect) {
		s.Characters[character].Effects = append(s.Characters[character].Effects, effect)
	}
}
func (s *State) clearDebuffs(character int) {
	debuffs := map[string]bool{"毒": true, "麻痺": true, "鈍足": true, "鈍化": true, "出血": true}
	kept := s.Characters[character].Effects[:0]
	for _, effect := range s.Characters[character].Effects {
		if !debuffs[effect] {
			kept = append(kept, effect)
		}
	}
	s.Characters[character].Effects = kept
}
func (s *State) roll(actor, target int, effect string, chance int) bool {
	h := fnv.New32a()
	fmt.Fprintf(h, "%s:%d:%s:%s:%s", s.MatchID, s.Revision, s.Characters[actor].ID, s.Characters[target].ID, effect)
	return int(h.Sum32()%100) < chance
}
func (s *State) passiveBoost(character int) int {
	for i, c := range s.Characters {
		if c.HP > 0 && c.OwnerID == s.Characters[character].OwnerID && passiveFor(c.DefinitionID).PassiveValueBoost > 0 && inSurroundingArea(c.Position, s.Characters[character].Position) && i != character {
			return passiveFor(c.DefinitionID).PassiveValueBoost
		}
	}
	return 0
}
func (s *State) attackPower(actor, power int) int {
	if s.hasEffect(actor, "威力上昇") {
		power = power * 125 / 100
	}
	if s.hasEffect(actor, "出血") {
		power = power * 75 / 100
	}
	boost := s.passiveBoost(actor)
	if fullHPBoost := passiveFor(s.Characters[actor].DefinitionID).FullHPAttackBoost; fullHPBoost > 0 && s.Characters[actor].HP == s.Characters[actor].MaxHP {
		power += fullHPBoost + boost
	}
	for i, c := range s.Characters {
		if c.HP <= 0 || c.OwnerID != s.Characters[actor].OwnerID || !inSurroundingArea(c.Position, s.Characters[actor].Position) {
			continue
		}
		passive := passiveFor(c.DefinitionID)
		if attackBoost := passive.AttackBoost; attackBoost > 0 && !(passive.ExcludeSelf && i == actor) {
			power += attackBoost + s.passiveBoost(i)
		}
	}
	return power
}
func (s *State) damageReduction(character int) int {
	if reduction := passiveFor(s.Characters[character].DefinitionID).DamageReduction; reduction > 0 {
		return reduction + s.passiveBoost(character)
	}
	return 0
}
func (s *State) tileAt(position Position) int {
	for i, tile := range s.TileEffects {
		if tile.Position == position {
			return i
		}
	}
	return -1
}
func (s *State) baseAt(position Position) int {
	for i, base := range s.Bases {
		if base.Position == position {
			return i
		}
	}
	return -1
}
func (s *State) setTile(position Position, tileType, ownerID string) {
	if i := s.tileAt(position); i >= 0 {
		s.TileEffects[i] = TileEffect{position, tileType, ownerID}
		return
	}
	s.TileEffects = append(s.TileEffects, TileEffect{position, tileType, ownerID})
}
func (s *State) triggerTile(character int) {
	index := s.tileAt(s.Characters[character].Position)
	if index < 0 {
		return
	}
	tile := s.TileEffects[index]
	id := s.Characters[character].DefinitionID
	if id == "tsukiha" || id == "berenice" && tile.Type == "地雷" {
		return
	}
	switch tile.Type {
	case "地雷":
		s.Characters[character].HP = clamp(s.Characters[character].HP-100, 0, s.Characters[character].MaxHP)
		s.TileEffects = append(s.TileEffects[:index], s.TileEffects[index+1:]...)
	case "まきびし":
		s.Characters[character].HP = clamp(s.Characters[character].HP-10, 0, s.Characters[character].MaxHP)
	case "毒ガス":
		if s.roll(character, character, "毒ガスマス", 50) {
			s.addEffect(character, "毒")
		}
	}
}

func (s *State) ignoresDebuffTiles(character int) bool {
	return s.Characters[character].DefinitionID == "tsukiha"
}

func (s *State) processTurnEnd(playerID string) {
	// 同時に解決されるターン終了時効果は、回復をすべて適用してから
	// ダメージを適用する。途中のHPで勝敗判定は行わない。
	healedTargets := 0
	healedAmount := 0
	for i, c := range s.Characters {
		if c.HP <= 0 || c.OwnerID != playerID {
			continue
		}
		boost := s.passiveBoost(i)
		if turnHeal := passiveFor(c.DefinitionID).TurnHeal; turnHeal > 0 {
			targets, amount := s.healNearby(i, turnHeal+boost)
			healedTargets += targets
			healedAmount += amount
		}
	}
	if healedAmount > 0 {
		s.commit("TURN_END_RECOVERY", fmt.Sprintf("パッシブにより%d体を合計%d回復", healedTargets, healedAmount))
	}

	poisonedTargets := 0
	poisonDamage := 0
	gasPoisonedTargets := 0
	for i := range s.Characters {
		c := &s.Characters[i]
		if c.OwnerID != playerID || c.HP <= 0 {
			continue
		}
		if s.hasEffect(i, "毒") {
			before := c.HP
			c.HP = clamp(c.HP-40, 0, c.MaxHP)
			poisonedTargets++
			poisonDamage += before - c.HP
		}
		if tile := s.tileAt(c.Position); tile >= 0 && s.TileEffects[tile].Type == "毒ガス" && !s.ignoresDebuffTiles(i) {
			alreadyPoisoned := s.hasEffect(i, "毒")
			s.addEffect(i, "毒")
			if !alreadyPoisoned && s.hasEffect(i, "毒") {
				gasPoisonedTargets++
			}
		}
		effects := c.Effects[:0]
		for _, effect := range c.Effects {
			if effect != "麻痺" {
				effects = append(effects, effect)
			}
		}
		c.Effects = effects
	}
	if poisonedTargets > 0 {
		s.commit("TURN_END_DAMAGE", fmt.Sprintf("%d体が毒により合計%dダメージ", poisonedTargets, poisonDamage))
	}
	if gasPoisonedTargets > 0 {
		s.commit("TURN_END_EFFECT", fmt.Sprintf("毒ガスマスにより%d体に毒を付与", gasPoisonedTargets))
	}
}
func (s *State) healNearby(source, amount int) (int, int) {
	targets := 0
	total := 0
	for i := range s.Characters {
		if s.Characters[i].HP > 0 && s.Characters[i].OwnerID == s.Characters[source].OwnerID && inSurroundingArea(s.Characters[i].Position, s.Characters[source].Position) {
			before := s.Characters[i].HP
			s.Characters[i].HP = clamp(s.Characters[i].HP+amount, 0, s.Characters[i].MaxHP)
			if healed := s.Characters[i].HP - before; healed > 0 {
				targets++
				total += healed
			}
		}
	}
	return targets, total
}
