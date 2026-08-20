package game

import (
	"errors"
	"fmt"
	"hash/fnv"
	"math"
	"time"
)

const (
	Width        = 8
	Height       = 5
	MaxCost      = 50
	MaxBaseHP    = 300
	TurnDuration = 90 * time.Second
)

var (
	ErrNotYourTurn   = errors.New("相手のターンです")
	ErrInvalidAction = errors.New("実行できない操作です")
	ErrStaleRevision = errors.New("古いゲーム状態からの操作です")
)

type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}
type AttackDefinition struct {
	Name         string     `json:"name"`
	Cost         int        `json:"cost"`
	Power        int        `json:"power"`
	Range        int        `json:"range"`
	Target       string     `json:"target"`
	Pattern      []Position `json:"pattern"`
	Effect       string     `json:"effect,omitempty"`
	EffectChance int        `json:"effectChance,omitempty"`
	Tile         string     `json:"tile,omitempty"`
	ClearDebuffs bool       `json:"clearDebuffs,omitempty"`
}
type CharacterDefinition struct {
	ID        string              `json:"id"`
	Name      string              `json:"name"`
	Image     string              `json:"image"`
	Portrait  string              `json:"portrait"`
	MaxHP     int                 `json:"maxHP"`
	MoveCost  int                 `json:"moveCost"`
	MoveRange int                 `json:"moveRange"`
	Attacks   [3]AttackDefinition `json:"attacks"`
}
type Character struct {
	ID           string   `json:"id"`
	DefinitionID string   `json:"definitionId"`
	OwnerID      string   `json:"ownerId"`
	Name         string   `json:"name"`
	HP           int      `json:"hp"`
	MaxHP        int      `json:"maxHP"`
	Position     Position `json:"position"`
	Effects      []string `json:"effects"`
}
type Player struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Cost int    `json:"cost"`
}
type Base struct {
	OwnerID  string   `json:"ownerId"`
	HP       int      `json:"hp"`
	MaxHP    int      `json:"maxHP"`
	Position Position `json:"position"`
}
type TileEffect struct {
	Position Position `json:"position"`
	Type     string   `json:"type"`
	OwnerID  string   `json:"ownerId"`
}
type Event struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
type State struct {
	MatchID        string       `json:"matchId"`
	Revision       uint64       `json:"revision"`
	Started        bool         `json:"started"`
	ReadyPlayerIDs []string     `json:"readyPlayerIds"`
	Players        [2]Player    `json:"players"`
	Bases          [2]Base      `json:"bases"`
	Characters     []Character  `json:"characters"`
	TileEffects    []TileEffect `json:"tileEffects"`
	BlockedCells   []Position   `json:"blockedCells"`
	TurnPlayerID   string       `json:"turnPlayerId"`
	Turn           int          `json:"turn"`
	TurnDeadline   time.Time    `json:"turnDeadline"`
	ServerTime     time.Time    `json:"serverTime,omitempty"`
	WinnerID       string       `json:"winnerId,omitempty"`
	Finished       bool         `json:"finished"`
	LastEvent      Event        `json:"lastEvent"`
	Events         []Event      `json:"events"`
}
type Command struct {
	ID               string   `json:"commandId"`
	ExpectedRevision uint64   `json:"expectedRevision"`
	CharacterID      string   `json:"characterId"`
	AttackIndex      int      `json:"attackIndex"`
	Target           Position `json:"target"`
	Direction        Position `json:"direction"`
}

func p(points ...Position) []Position { return points }
func atk(name string, cost, power int, target string, pattern []Position) AttackDefinition {
	r := 0
	for _, cell := range pattern {
		if d := abs(cell.X) + abs(cell.Y); d > r {
			r = d
		}
	}
	return AttackDefinition{Name: name, Cost: cost, Power: power, Range: r, Target: target, Pattern: pattern}
}

var adjacent = p(Position{1, 0})
var Definitions = []CharacterDefinition{
	{ID: "sophie", Name: "ソフィー", Image: "Sophie_mini.png", Portrait: "Sophie.png", MaxHP: 150, MoveCost: 10, MoveRange: 2, Attacks: [3]AttackDefinition{atk("突き sprout～芽生え～", 10, 10, "enemy", adjacent), atk("範囲狙撃 growth～成長～", 20, 50, "enemy", p(Position{3, -1}, Position{3, 0}, Position{3, 1})), atk("集中狙撃 bloom～開花～", 50, 250, "enemy", p(Position{3, 0}))}},
	{ID: "jude", Name: "ジュード", Image: "Jude_mini.png", Portrait: "Jude.png", MaxHP: 350, MoveCost: 10, MoveRange: 2, Attacks: [3]AttackDefinition{func() AttackDefinition {
		a := atk("急襲", 10, 20, "enemy", adjacent)
		a.Effect = "出血"
		a.EffectChance = 30
		return a
	}(), atk("切り裂き", 20, 50, "enemy", adjacent), func() AttackDefinition {
		a := atk("応急手当", 30, -30, "ally", p(Position{0, 0}))
		a.ClearDebuffs = true
		return a
	}()}},
	{ID: "nadia", Name: "ナディア", Image: "Nadia_mini.png", Portrait: "Nadia.png", MaxHP: 200, MoveCost: 7, MoveRange: 3, Attacks: [3]AttackDefinition{func() AttackDefinition {
		a := atk("対処番号05：対包囲戦術", 10, 20, "enemy", p(Position{1, 0}, Position{0, 1}, Position{0, -1}, Position{-1, 0}))
		a.Effect = "毒"
		a.EffectChance = 20
		return a
	}(), func() AttackDefinition {
		a := atk("対処番号03：前方範囲殲滅", 20, 40, "enemy", p(Position{1, -1}, Position{1, 0}, Position{1, 1}))
		a.Effect = "毒"
		a.EffectChance = 40
		return a
	}(), func() AttackDefinition {
		a := atk("対処番号02：前方殲滅・改", 30, 60, "enemy", adjacent)
		a.Effect = "毒"
		a.EffectChance = 60
		return a
	}()}},
	{ID: "tsukiha", Name: "月葉", Image: "Tsukiha_mini.png", Portrait: "Tsukiha.png", MaxHP: 100, MoveCost: 3, MoveRange: 4, Attacks: [3]AttackDefinition{func() AttackDefinition {
		a := atk("忍法：手裏剣投げの術・近", 5, 10, "enemy", p(Position{1, -1}, Position{2, 0}, Position{1, 1}))
		a.Effect = "出血"
		a.EffectChance = 10
		return a
	}(), func() AttackDefinition {
		a := atk("忍法：手裏剣投げの術・遠", 5, 10, "enemy", p(Position{2, -1}, Position{3, 0}, Position{2, 1}))
		a.Effect = "出血"
		a.EffectChance = 10
		return a
	}(), func() AttackDefinition {
		a := atk("忍法：まきびし投げの術", 10, 0, "cell", adjacent)
		a.Tile = "まきびし"
		return a
	}()}},
	{ID: "aoi", Name: "扇衣", Image: "Aoi_mini.png", Portrait: "Aoi.png", MaxHP: 250, MoveCost: 8, MoveRange: 2, Attacks: [3]AttackDefinition{atk("汐汲～しおくみ～", 10, 20, "enemy", p(Position{1, 0}, Position{0, 1}, Position{0, -1})), atk("女伊達～おんなだて～", 20, 40, "enemy", p(Position{1, -1}, Position{1, 0}, Position{1, 1})), atk("鷺娘～さぎむすめ～", 30, 60, "enemy", p(Position{-1, -1}, Position{0, -1}, Position{1, -1}, Position{-1, 1}, Position{0, 1}, Position{1, 1}))}},
	{ID: "sena", Name: "星凪", Image: "Sena_mini.png", Portrait: "Sena.png", MaxHP: 250, MoveCost: 10, MoveRange: 2, Attacks: [3]AttackDefinition{atk("一条流槍術：衝き", 20, 60, "enemy", p(Position{2, 0})), atk("一条流槍術：掃い", 25, 40, "enemy", p(Position{2, -1}, Position{2, 0}, Position{2, 1})), atk("一条流槍術：薙ぎ", 40, 90, "enemy", p(Position{1, 0}, Position{2, 0}))}},
	{ID: "berenice", Name: "ベレニス", Image: "berenice_mini.png", Portrait: "Berenice.png", MaxHP: 200, MoveCost: 6, MoveRange: 2, Attacks: [3]AttackDefinition{func() AttackDefinition {
		a := atk("地雷設置", 15, 0, "cell", adjacent)
		a.Tile = "地雷"
		return a
	}(), atk("爆破！", 25, 60, "enemy", p(Position{1, 0}, Position{2, -1}, Position{2, 0}, Position{2, 1}, Position{3, 0})), atk("小型爆弾", 20, 40, "enemy", adjacent)}},
	{ID: "chiyo", Name: "千代", Image: "Chiyo_mini.png", Portrait: "Chiyo.png", MaxHP: 150, MoveCost: 5, MoveRange: 3, Attacks: [3]AttackDefinition{atk("一文字斬り", 10, 30, "enemy", p(Position{1, -1}, Position{1, 0}, Position{1, 1})), func() AttackDefinition {
		a := atk("袈裟斬り", 20, 60, "enemy", adjacent)
		a.Effect = "出血"
		a.EffectChance = 50
		return a
	}(), atk("真向斬り", 50, 220, "enemy", adjacent)}},
	{ID: "shincho", Name: "新著", Image: "Shincho_mini.png", Portrait: "Shincho.png", MaxHP: 80, MoveCost: 15, MoveRange: 2, Attacks: [3]AttackDefinition{atk("進捗どうですか？", 20, 220, "any", p(Position{-2, 0}, Position{-1, -1}, Position{-1, 0}, Position{-1, 1}, Position{0, -2}, Position{0, -1}, Position{0, 1}, Position{0, 2}, Position{1, -1}, Position{1, 0}, Position{1, 1}, Position{2, 0})), atk(":oyoo:", 15, -40, "ally", p(Position{-1, 0}, Position{0, -1}, Position{0, 0}, Position{0, 1}, Position{1, 0})), func() AttackDefinition {
		a := atk(":iihanashi:", 15, 0, "ally", p(Position{-1, 0}, Position{0, -1}, Position{0, 0}, Position{0, 1}, Position{1, 0}))
		a.ClearDebuffs = true
		return a
	}()}},
	{ID: "zina", Name: "ジーナ", Image: "Zina_mini.png", Portrait: "Zina.png", MaxHP: 200, MoveCost: 5, MoveRange: 3, Attacks: [3]AttackDefinition{func() AttackDefinition {
		a := atk("遠距離制圧", 20, 20, "enemy", p(Position{3, 0}))
		a.Effect = "麻痺"
		a.EffectChance = 30
		return a
	}(), func() AttackDefinition {
		a := atk("中距離制圧", 20, 20, "enemy", p(Position{2, 0}))
		a.Effect = "麻痺"
		a.EffectChance = 60
		return a
	}(), atk("軍隊式近接格闘術", 30, 60, "enemy", adjacent)}},
	{ID: "dana", Name: "ダーナ", Image: "Dana_mini.png", Portrait: "Dana.png", MaxHP: 200, MoveCost: 10, MoveRange: 2, Attacks: [3]AttackDefinition{func() AttackDefinition {
		a := atk("残留型毒ガス", 10, 0, "cell", adjacent)
		a.Tile = "毒ガス"
		return a
	}(), func() AttackDefinition {
		a := atk("拡散型毒ガス", 20, 20, "enemy", p(Position{1, 0}, Position{2, -1}, Position{2, 0}, Position{2, 1}, Position{3, 0}))
		a.Effect = "毒"
		a.EffectChance = 50
		return a
	}(), atk("活性化ガス", 20, -30, "ally", p(Position{0, 0}))}},
}

func NewState(id string, players [2]Player, selections [2][]string) *State {
	return newState(id, players, selections, true)
}

func NewPendingState(id string, players [2]Player, selections [2][]string) *State {
	return newState(id, players, selections, false)
}

func newState(id string, players [2]Player, selections [2][]string, started bool) *State {
	lastEvent := Event{Type: "MATCH_FOUND", Text: "対戦相手が見つかりました"}
	var deadline time.Time
	if started {
		lastEvent = Event{Type: "MATCH_STARTED", Text: "対戦開始"}
		deadline = time.Now().Add(TurnDuration)
	}
	s := &State{MatchID: id, Revision: 1, Started: started, Players: players, TurnPlayerID: players[0].ID, Turn: 1, TurnDeadline: deadline, LastEvent: lastEvent}
	s.Players[0].Cost, s.Players[1].Cost = MaxCost, MaxCost
	s.EnsureBases()
	starts := [2][]Position{{{0, 0}, {1, 2}, {0, 4}}, {{6, 2}, {7, 0}, {7, 4}}}
	for side := range 2 {
		for i, defID := range selections[side] {
			if i >= 3 {
				break
			}
			if def, ok := Definition(defID); ok {
				s.Characters = append(s.Characters, Character{ID: fmt.Sprintf("p%d-c%d", side+1, i+1), DefinitionID: def.ID, OwnerID: players[side].ID, Name: def.Name, HP: def.MaxHP, MaxHP: def.MaxHP, Position: starts[side][i], Effects: []string{}})
			}
		}
	}
	s.record(s.LastEvent)
	return s
}

func (s *State) Ready(playerID string) error {
	if s.Started {
		return nil
	}
	if playerID != s.Players[0].ID && playerID != s.Players[1].ID {
		return ErrInvalidAction
	}
	for _, id := range s.ReadyPlayerIDs {
		if id == playerID {
			return nil
		}
	}
	s.ReadyPlayerIDs = append(s.ReadyPlayerIDs, playerID)
	s.Revision++
	s.LastEvent = Event{Type: "PLAYER_READY", Text: "相手を待っています"}
	if len(s.ReadyPlayerIDs) == 2 {
		s.Started = true
		s.TurnDeadline = time.Now().Add(TurnDuration)
		s.LastEvent = Event{Type: "MATCH_STARTED", Text: "対戦開始"}
		s.Events = append(s.Events, s.LastEvent)
	}
	return nil
}
func (s *State) EnsureBases() {
	positions := [2]Position{{0, 2}, {7, 2}}
	for i := range s.Bases {
		if s.Bases[i].MaxHP == 0 {
			s.Bases[i] = Base{OwnerID: s.Players[i].ID, HP: MaxBaseHP, MaxHP: MaxBaseHP, Position: positions[i]}
		} else if s.Bases[i].MaxHP != MaxBaseHP {
			s.Bases[i].HP = clamp(s.Bases[i].HP+(MaxBaseHP-s.Bases[i].MaxHP), 0, MaxBaseHP)
			s.Bases[i].MaxHP = MaxBaseHP
		}
	}
	if len(s.BlockedCells) == 0 {
		s.BlockedCells = []Position{{1, 1}, {2, 3}, {5, 1}, {6, 3}}
	}
}
func Definition(id string) (CharacterDefinition, bool) {
	for _, d := range Definitions {
		if d.ID == id {
			return d, true
		}
	}
	return CharacterDefinition{}, false
}
func (s *State) ApplyMove(playerID string, c Command) error {
	if err := s.validate(playerID, c); err != nil {
		return err
	}
	i, d, err := s.actor(playerID, c.CharacterID)
	if err != nil {
		return err
	}
	if s.hasEffect(i, "麻痺") {
		return ErrInvalidAction
	}
	moveCost := d.MoveCost
	if s.hasEffect(i, "俊足") {
		moveCost -= 2
	}
	if s.hasEffect(i, "鈍足") {
		moveCost += 2
	}
	if tile := s.tileAt(s.Characters[i].Position); tile >= 0 && s.TileEffects[tile].Type == "まきびし" {
		moveCost += 2
	}
	moveCost = max(0, moveCost)
	if !onBoard(c.Target) || s.blocked(c.Target) || distance(s.Characters[i].Position, c.Target) > d.MoveRange || s.occupied(c.Target, c.CharacterID) || s.enemyBaseAt(playerID, c.Target) || s.cost(playerID) < moveCost {
		return ErrInvalidAction
	}
	s.Characters[i].Position = c.Target
	s.spend(playerID, moveCost)
	s.triggerTile(i)
	s.commit("MOVED", fmt.Sprintf("%sが移動（コスト%d）", s.Characters[i].Name, moveCost))
	s.checkWinner()
	return nil
}
func (s *State) ApplyAttack(playerID string, c Command) error {
	if err := s.validate(playerID, c); err != nil {
		return err
	}
	i, d, err := s.actor(playerID, c.CharacterID)
	if err != nil || c.AttackIndex < 0 || c.AttackIndex >= len(d.Attacks) {
		return ErrInvalidAction
	}
	a := d.Attacks[c.AttackIndex]
	if s.hasEffect(i, "麻痺") {
		return ErrInvalidAction
	}
	attackCost := a.Cost
	if s.hasEffect(i, "俊敏化") {
		attackCost -= 2
	}
	if s.hasEffect(i, "鈍化") {
		attackCost += 2
	}
	attackCost = max(0, attackCost)
	cells := s.attackCells(i, a, c.Direction)
	if s.cost(playerID) < attackCost || !containsPosition(cells, c.Target) {
		return ErrInvalidAction
	}
	if a.Tile != "" {
		if s.baseAt(c.Target) >= 0 || s.blocked(c.Target) || s.occupied(c.Target, "") {
			return ErrInvalidAction
		}
		s.setTile(c.Target, a.Tile, playerID)
		s.spend(playerID, attackCost)
		s.commit("TILE_PLACED", fmt.Sprintf("%sが%sマスを設置", s.Characters[i].Name, a.Tile))
		return nil
	}
	affected := 0
	for j := range s.Characters {
		if s.Characters[j].HP <= 0 || !containsPosition(cells, s.Characters[j].Position) {
			continue
		}
		same := s.Characters[j].OwnerID == playerID
		if !(a.Target == "any" || a.Target == "ally" && same || a.Target == "enemy" && !same) {
			continue
		}
		power := a.Power
		if power > 0 {
			power = s.attackPower(i, power)
			if d.ID != "sena" {
				power = max(0, power-s.damageReduction(j))
			}
		}
		s.Characters[j].HP = clamp(s.Characters[j].HP-power, 0, s.Characters[j].MaxHP)
		if a.ClearDebuffs {
			s.clearDebuffs(j)
		}
		if a.Effect != "" && !same && s.roll(i, j, a.Effect, a.EffectChance) {
			s.addEffect(j, a.Effect)
		}
		if d.ID == "nadia" && !same && s.roll(i, j, "過量使用", 20) {
			s.addEffect(j, "毒")
		}
		affected++
	}
	if a.Power >= 0 && (a.Target == "enemy" || a.Target == "any") {
		for j := range s.Bases {
			if s.Bases[j].OwnerID != playerID && containsPosition(cells, s.Bases[j].Position) {
				s.Bases[j].HP = clamp(s.Bases[j].HP-s.attackPower(i, a.Power), 0, s.Bases[j].MaxHP)
				affected++
			}
		}
	}
	if affected == 0 {
		return ErrInvalidAction
	}
	s.spend(playerID, attackCost)
	s.commit("ATTACKED", fmt.Sprintf("%sの%s：%d対象に効果", s.Characters[i].Name, a.Name, affected))
	s.checkWinner()
	return nil
}
func (s *State) EndTurn(playerID string, expected uint64) error {
	if !s.Started || s.Finished {
		return ErrInvalidAction
	}
	if s.Revision != expected {
		return ErrStaleRevision
	}
	if s.TurnPlayerID != playerID {
		return ErrNotYourTurn
	}
	s.advanceTurn("ターン終了")
	return nil
}
func (s *State) Surrender(playerID string, expected uint64) error {
	if !s.Started || s.Finished {
		return ErrInvalidAction
	}
	if s.Revision != expected {
		return ErrStaleRevision
	}
	found := false
	for _, p := range s.Players {
		if p.ID == playerID {
			found = true
		} else {
			s.WinnerID = p.ID
		}
	}
	if !found {
		return ErrInvalidAction
	}
	s.Finished = true
	s.commit("MATCH_FINISHED", "投降により勝敗が決定")
	return nil
}
func (s *State) ExpireTurn(now time.Time) {
	if s.Started && !s.Finished && now.After(s.TurnDeadline) {
		s.advanceTurn("90秒経過によりターン交代")
	}
}
func (s *State) advanceTurn(reason string) {
	s.processTurnEnd(s.TurnPlayerID)
	s.checkWinner()
	if s.Finished {
		return
	}
	if s.Players[0].ID == s.TurnPlayerID {
		s.TurnPlayerID = s.Players[1].ID
	} else {
		s.TurnPlayerID = s.Players[0].ID
	}
	s.Turn++
	for i := range s.Players {
		s.Players[i].Cost = MaxCost
	}
	s.TurnDeadline = time.Now().Add(TurnDuration)
	s.commit("TURN_ENDED", reason)
}
func (s *State) validate(playerID string, c Command) error {
	if !s.Started || s.Finished {
		return ErrInvalidAction
	}
	if s.Revision != c.ExpectedRevision {
		return ErrStaleRevision
	}
	if s.TurnPlayerID != playerID {
		return ErrNotYourTurn
	}
	return nil
}
func (s *State) actor(playerID, id string) (int, CharacterDefinition, error) {
	for i := range s.Characters {
		if s.Characters[i].ID == id && s.Characters[i].OwnerID == playerID && s.Characters[i].HP > 0 {
			d, _ := Definition(s.Characters[i].DefinitionID)
			return i, d, nil
		}
	}
	return 0, CharacterDefinition{}, ErrInvalidAction
}
func (s *State) occupied(p Position, except string) bool {
	for _, c := range s.Characters {
		if c.ID != except && c.HP > 0 && c.Position == p {
			return true
		}
	}
	return false
}
func (s *State) attackCells(actor int, a AttackDefinition, facing Position) []Position {
	if abs(facing.X)+abs(facing.Y) != 1 {
		facing = Position{1, 0}
		if s.Characters[actor].OwnerID == s.Players[1].ID {
			facing = Position{-1, 0}
		}
	}
	result := make([]Position, 0, len(a.Pattern))
	for _, offset := range a.Pattern {
		rotated := Position{offset.X*facing.X - offset.Y*facing.Y, offset.X*facing.Y + offset.Y*facing.X}
		cell := Position{s.Characters[actor].Position.X + rotated.X, s.Characters[actor].Position.Y + rotated.Y}
		if onBoard(cell) {
			result = append(result, cell)
		}
	}
	return result
}
func (s *State) blocked(position Position) bool { return containsPosition(s.BlockedCells, position) }
func containsPosition(cells []Position, p Position) bool {
	for _, cell := range cells {
		if cell == p {
			return true
		}
	}
	return false
}
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
		if c.HP > 0 && c.OwnerID == s.Characters[character].OwnerID && c.DefinitionID == "zina" && distance(c.Position, s.Characters[character].Position) <= 1 && i != character {
			return 20
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
	if s.Characters[actor].DefinitionID == "chiyo" && s.Characters[actor].HP == s.Characters[actor].MaxHP {
		power += 50 + boost
	}
	for i, c := range s.Characters {
		if c.HP <= 0 || c.OwnerID != s.Characters[actor].OwnerID || distance(c.Position, s.Characters[actor].Position) > 1 {
			continue
		}
		if c.DefinitionID == "sophie" {
			power += 20 + s.passiveBoost(i)
		}
		if c.DefinitionID == "shincho" {
			power += 10 + s.passiveBoost(i)
		}
	}
	return power
}
func (s *State) damageReduction(character int) int {
	if s.Characters[character].DefinitionID == "jude" {
		return 20 + s.passiveBoost(character)
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
	if id == "tsukiha" || id == "dana" || id == "berenice" && tile.Type == "地雷" {
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
func (s *State) processTurnEnd(playerID string) {
	for i := range s.Characters {
		c := &s.Characters[i]
		if c.OwnerID != playerID || c.HP <= 0 {
			continue
		}
		if s.hasEffect(i, "毒") {
			c.HP = clamp(c.HP-40, 0, c.MaxHP)
		}
		if tile := s.tileAt(c.Position); tile >= 0 && s.TileEffects[tile].Type == "毒ガス" {
			s.addEffect(i, "毒")
		}
		effects := c.Effects[:0]
		for _, effect := range c.Effects {
			if effect != "麻痺" {
				effects = append(effects, effect)
			}
		}
		c.Effects = effects
	}
	for i, c := range s.Characters {
		if c.HP <= 0 || c.OwnerID != playerID {
			continue
		}
		boost := s.passiveBoost(i)
		if c.DefinitionID == "aoi" {
			s.healNearby(i, 30+boost)
		}
		if c.DefinitionID == "shincho" {
			s.healNearby(i, 10+boost)
		}
	}
}
func (s *State) healNearby(source, amount int) {
	for i := range s.Characters {
		if s.Characters[i].HP > 0 && s.Characters[i].OwnerID == s.Characters[source].OwnerID && distance(s.Characters[i].Position, s.Characters[source].Position) <= 1 {
			s.Characters[i].HP = clamp(s.Characters[i].HP+amount, 0, s.Characters[i].MaxHP)
		}
	}
}
func (s *State) cost(id string) int {
	for _, p := range s.Players {
		if p.ID == id {
			return p.Cost
		}
	}
	return 0
}
func (s *State) spend(id string, n int) {
	for i := range s.Players {
		if s.Players[i].ID == id {
			s.Players[i].Cost -= n
		}
	}
}
func (s *State) record(e Event) {
	s.Events = append(s.Events, e)
	if len(s.Events) > 30 {
		s.Events = s.Events[len(s.Events)-30:]
	}
}
func (s *State) commit(t, text string) {
	s.Revision++
	s.LastEvent = Event{Type: t, Text: text}
	s.record(s.LastEvent)
}
func (s *State) checkWinner() {
	alive := [2]int{}
	for _, c := range s.Characters {
		for i, p := range s.Players {
			if c.OwnerID == p.ID && c.HP > 0 {
				alive[i]++
			}
		}
	}
	winner := -1
	reason := ""
	if alive[0] == 0 && alive[1] == 0 {
		if s.Bases[0].HP > s.Bases[1].HP {
			winner = 0
		} else {
			winner = 1
		}
		reason = "両軍全滅のため拠点耐久力で決着"
	} else if alive[0] == 0 {
		winner = 1
		reason = "相手キャラクター3体を戦闘不能にして勝利"
	} else if alive[1] == 0 {
		winner = 0
		reason = "相手キャラクター3体を戦闘不能にして勝利"
	} else if s.Bases[0].HP == 0 && s.Bases[1].HP == 0 {
		if alive[0] > alive[1] {
			winner = 0
		} else {
			winner = 1
		}
		reason = "両拠点陥落のため生存キャラクター数で決着"
	} else if s.Bases[0].HP == 0 {
		winner = 1
		reason = "相手拠点の耐久力を0にして勝利"
	} else if s.Bases[1].HP == 0 {
		winner = 0
		reason = "相手拠点の耐久力を0にして勝利"
	}
	if winner >= 0 && !s.Finished {
		s.Finished = true
		s.WinnerID = s.Players[winner].ID
		s.commit("MATCH_FINISHED", reason)
	}
}
func (s *State) enemyBaseAt(playerID string, p Position) bool { return s.enemyBase(playerID, p) >= 0 }
func (s *State) enemyBase(playerID string, p Position) int {
	for i, b := range s.Bases {
		if b.OwnerID != playerID && b.Position == p && b.HP > 0 {
			return i
		}
	}
	return -1
}
func clamp(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}
func onBoard(p Position) bool    { return p.X >= 0 && p.X < Width && p.Y >= 0 && p.Y < Height }
func distance(a, b Position) int { return int(math.Abs(float64(a.X-b.X)) + math.Abs(float64(a.Y-b.Y))) }
func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
