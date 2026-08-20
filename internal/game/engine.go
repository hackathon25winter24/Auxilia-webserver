package game

import (
	"errors"
	"fmt"
	"math"
	"time"
)

const (
	Width        = 8
	Height       = 5
	MaxCost      = 50
	MaxBaseHP    = 200
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
	Name   string `json:"name"`
	Cost   int    `json:"cost"`
	Power  int    `json:"power"`
	Range  int    `json:"range"`
	Target string `json:"target"`
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
type Event struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
type State struct {
	MatchID        string      `json:"matchId"`
	Revision       uint64      `json:"revision"`
	Started        bool        `json:"started"`
	ReadyPlayerIDs []string    `json:"readyPlayerIds"`
	Players        [2]Player   `json:"players"`
	Bases          [2]Base     `json:"bases"`
	Characters     []Character `json:"characters"`
	TurnPlayerID   string      `json:"turnPlayerId"`
	Turn           int         `json:"turn"`
	TurnDeadline   time.Time   `json:"turnDeadline"`
	ServerTime     time.Time   `json:"serverTime,omitempty"`
	WinnerID       string      `json:"winnerId,omitempty"`
	Finished       bool        `json:"finished"`
	LastEvent      Event       `json:"lastEvent"`
	Events         []Event     `json:"events"`
}
type Command struct {
	ID               string   `json:"commandId"`
	ExpectedRevision uint64   `json:"expectedRevision"`
	CharacterID      string   `json:"characterId"`
	AttackIndex      int      `json:"attackIndex"`
	Target           Position `json:"target"`
}

func attacks(a1 string, c1, p1, r1 int, a2 string, c2, p2, r2 int, a3 string, c3, p3, r3 int) [3]AttackDefinition {
	return [3]AttackDefinition{{a1, c1, p1, r1, "enemy"}, {a2, c2, p2, r2, "enemy"}, {a3, c3, p3, r3, "enemy"}}
}

var Definitions = []CharacterDefinition{
	{ID: "sophie", Name: "ソフィー", Image: "Sophie_mini.png", Portrait: "Sophie.png", MaxHP: 150, MoveCost: 10, MoveRange: 2, Attacks: attacks("突き sprout", 10, 10, 1, "範囲狙撃 growth", 30, 40, 3, "集中狙撃 bloom", 50, 200, 3)},
	{ID: "jude", Name: "ジュード", Image: "Jude_mini.png", Portrait: "Jude.png", MaxHP: 300, MoveCost: 10, MoveRange: 2, Attacks: [3]AttackDefinition{{"急襲", 10, 20, 1, "enemy"}, {"切り裂き", 20, 40, 1, "enemy"}, {"応急手当", 30, -20, 0, "ally"}}},
	{ID: "nadia", Name: "ナディア", Image: "Nadia_mini.png", Portrait: "Nadia.png", MaxHP: 150, MoveCost: 7, MoveRange: 3, Attacks: attacks("対処番号05", 10, 20, 1, "対処番号03", 20, 40, 1, "対処番号02", 30, 60, 1)},
	{ID: "tsukiha", Name: "月葉", Image: "Tsukiha_mini.png", Portrait: "Tsukiha.png", MaxHP: 100, MoveCost: 3, MoveRange: 4, Attacks: attacks("手裏剣・近", 5, 10, 2, "手裏剣・遠", 5, 10, 3, "まきびし投げ", 10, 0, 1)},
	{ID: "aoi", Name: "扇衣", Image: "Aoi_mini.png", Portrait: "Aoi.png", MaxHP: 250, MoveCost: 10, MoveRange: 2, Attacks: attacks("汐汲", 10, 20, 1, "女伊達", 20, 40, 1, "鷺娘", 30, 60, 1)},
	{ID: "sena", Name: "星凪", Image: "Sena_mini.png", Portrait: "Sena.png", MaxHP: 250, MoveCost: 10, MoveRange: 2, Attacks: attacks("一条流槍術・衝き", 20, 60, 2, "一条流槍術・掃い", 25, 40, 3, "一条流槍術・薙ぎ", 40, 90, 2)},
	{ID: "berenice", Name: "ベレニス", Image: "berenice_mini.png", Portrait: "Berenice.png", MaxHP: 200, MoveCost: 10, MoveRange: 2, Attacks: attacks("地雷設置", 20, 0, 1, "爆破！", 30, 40, 3, "小型爆弾", 20, 40, 1)},
	{ID: "chiyo", Name: "千代", Image: "Chiyo_mini.png", Portrait: "Chiyo.png", MaxHP: 300, MoveCost: 5, MoveRange: 3, Attacks: attacks("一文字斬り", 10, 20, 1, "袈裟斬り", 20, 40, 1, "真向斬り", 50, 200, 1)},
	{ID: "shincho", Name: "新著", Image: "Shincho_mini.png", Portrait: "Shincho.png", MaxHP: 100, MoveCost: 10, MoveRange: 2, Attacks: [3]AttackDefinition{{"進捗どうですか？", 10, 50, 2, "any"}, {":oyoo:", 40, -50, 1, "ally"}, {":iikanashi:", 20, 0, 1, "ally"}}},
	{ID: "zina", Name: "ジーナ", Image: "Zina_mini.png", Portrait: "Zina.png", MaxHP: 200, MoveCost: 5, MoveRange: 3, Attacks: attacks("遠距離制圧", 20, 20, 3, "中距離制圧", 20, 20, 2, "軍隊式近接格闘術", 30, 60, 1)},
	{ID: "dana", Name: "ダーナ", Image: "Dana_mini.png", Portrait: "Dana_select.png", MaxHP: 200, MoveCost: 10, MoveRange: 2, Attacks: [3]AttackDefinition{{"残留型ガス", 10, 0, 3, "enemy"}, {"拡散型毒ガス", 20, 20, 3, "enemy"}, {"活性化ガス", 20, 0, 0, "ally"}}},
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
		}
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
	if !onBoard(c.Target) || distance(s.Characters[i].Position, c.Target) > d.MoveRange || s.occupied(c.Target, c.CharacterID) || s.enemyBaseAt(playerID, c.Target) || s.cost(playerID) < d.MoveCost {
		return ErrInvalidAction
	}
	s.Characters[i].Position = c.Target
	s.spend(playerID, d.MoveCost)
	s.commit("MOVED", fmt.Sprintf("%sが移動（コスト%d）", s.Characters[i].Name, d.MoveCost))
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
	if s.cost(playerID) < a.Cost || distance(s.Characters[i].Position, c.Target) > a.Range {
		return ErrInvalidAction
	}
	target := -1
	for j := range s.Characters {
		same := s.Characters[j].OwnerID == playerID
		allowed := a.Target == "any" || (a.Target == "ally" && same) || (a.Target == "enemy" && !same)
		if allowed && s.Characters[j].HP > 0 && s.Characters[j].Position == c.Target {
			target = j
			break
		}
	}
	text := ""
	if target >= 0 {
		before := s.Characters[target].HP
		s.Characters[target].HP = clamp(s.Characters[target].HP-a.Power, 0, s.Characters[target].MaxHP)
		amount := before - s.Characters[target].HP
		text = fmt.Sprintf("%sの%s：%sに%dダメージ", s.Characters[i].Name, a.Name, s.Characters[target].Name, amount)
		if amount < 0 {
			text = fmt.Sprintf("%sの%s：%sを%d回復", s.Characters[i].Name, a.Name, s.Characters[target].Name, -amount)
		}
	} else {
		base := s.enemyBase(playerID, c.Target)
		if base < 0 || (a.Target != "enemy" && a.Target != "any") || a.Power < 0 {
			return ErrInvalidAction
		}
		before := s.Bases[base].HP
		s.Bases[base].HP = clamp(s.Bases[base].HP-a.Power, 0, s.Bases[base].MaxHP)
		text = fmt.Sprintf("%sの%s：相手拠点に%dダメージ", s.Characters[i].Name, a.Name, before-s.Bases[base].HP)
	}
	s.spend(playerID, a.Cost)
	s.commit("ATTACKED", text)
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
	for _, b := range s.Bases {
		if b.HP == 0 {
			s.Finished = true
			for _, p := range s.Players {
				if p.ID != b.OwnerID {
					s.WinnerID = p.ID
				}
			}
			s.commit("MATCH_FINISHED", "相手拠点の耐久力を0にして勝利")
			return
		}
	}
	for _, p := range s.Players {
		alive := false
		for _, c := range s.Characters {
			if c.OwnerID == p.ID && c.HP > 0 {
				alive = true
			}
		}
		if !alive {
			s.Finished = true
			if s.Players[0].ID == p.ID {
				s.WinnerID = s.Players[1].ID
			} else {
				s.WinnerID = s.Players[0].ID
			}
			s.commit("MATCH_FINISHED", "相手キャラクター3体を戦闘不能にして勝利")
		}
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
