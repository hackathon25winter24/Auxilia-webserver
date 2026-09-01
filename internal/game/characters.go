package game

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
	ID                 string              `json:"id"`
	Name               string              `json:"name"`
	Image              string              `json:"image"`
	Portrait           string              `json:"portrait"`
	MaxHP              int                 `json:"maxHP"`
	MoveCost           int                 `json:"moveCost"`
	MoveRange          int                 `json:"moveRange"`
	PassiveName        string              `json:"passiveName"`
	PassiveDescription string              `json:"passiveDescription"`
	Attacks            [3]AttackDefinition `json:"attacks"`
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
	{ID: "sophie", Name: "ソフィー", Image: "Sophie_mini.png", Portrait: "Sophie.png", MaxHP: 100, MoveCost: 10, MoveRange: 2, Attacks: [3]AttackDefinition{atk("突き sprout～芽生え～", 10, 10, "enemy", adjacent), atk("範囲狙撃 growth～成長～", 20, 50, "enemy", p(Position{3, -1}, Position{3, 0}, Position{3, 1})), atk("集中狙撃 bloom～開花～", 50, 250, "enemy", p(Position{3, 0}))}},
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
		a := atk("忍法：手裏剣投げの術・近", 5, 15, "enemy", p(Position{1, -1}, Position{2, 0}, Position{1, 1}))
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
	{ID: "chiyo", Name: "千代", Image: "Chiyo_mini.png", Portrait: "Chiyo.png", MaxHP: 100, MoveCost: 5, MoveRange: 3, Attacks: [3]AttackDefinition{atk("一文字斬り", 10, 30, "enemy", p(Position{1, -1}, Position{1, 0}, Position{1, 1})), func() AttackDefinition {
		a := atk("袈裟斬り", 20, 60, "enemy", adjacent)
		a.Effect = "出血"
		a.EffectChance = 50
		return a
	}(), atk("真向斬り", 50, 220, "enemy", adjacent)}},
	{ID: "shincho", Name: "新著", Image: "Shincho_mini.png", Portrait: "Shincho.png", MaxHP: 50, MoveCost: 15, MoveRange: 2, Attacks: [3]AttackDefinition{atk("進捗どうですか？", 20, 250, "any", p(Position{-2, 0}, Position{-1, -1}, Position{-1, 0}, Position{-1, 1}, Position{0, -2}, Position{0, -1}, Position{0, 0}, Position{0, 1}, Position{0, 2}, Position{1, -1}, Position{1, 0}, Position{1, 1}, Position{2, 0})), atk(":oyoo:", 15, -40, "ally", p(Position{-1, 0}, Position{0, -1}, Position{0, 0}, Position{0, 1}, Position{1, 0})), func() AttackDefinition {
		a := atk(":iihanashi:", 15, 0, "ally", p(Position{-1, 0}, Position{0, -1}, Position{0, 0}, Position{0, 1}, Position{1, 0}))
		a.ClearDebuffs = true
		return a
	}()}},
	{ID: "zina", Name: "ジーナ", Image: "Zina_mini.png", Portrait: "Zina.png", MaxHP: 150, MoveCost: 5, MoveRange: 3, Attacks: [3]AttackDefinition{func() AttackDefinition {
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
	{ID: "dana", Name: "ダーナ", Image: "Dana_mini.png", Portrait: "Dana.png", MaxHP: 250, MoveCost: 10, MoveRange: 2, Attacks: [3]AttackDefinition{func() AttackDefinition {
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

var passiveDefinitions = map[string][2]string{
	"sophie":   {"範囲支援 fruit～結実～", "周囲1マス以内にいる自身以外の味方の攻撃ダメージを20上昇させる。"},
	"jude":     {"受け身", "自身が受けるダメージを20軽減する。"},
	"nadia":    {"対処番号04：過量使用", "攻撃が当たった敵に追加判定を行い、20%の確率で毒を与える。"},
	"tsukiha":  {"忍法：隠れ身の術", "デバフマスの影響を受けない。"},
	"aoi":      {"藤娘～ふじむすめ～", "自身のターン終了時、周囲1マス以内にいる味方のHPを30回復する。"},
	"sena":     {"一条流槍術：翻弄", "敵のパッシブによるダメージ軽減を無視して攻撃する。"},
	"berenice": {"爆弾処理", "地雷マスに乗ってもダメージを受けない。"},
	"chiyo":    {"刀剣拝見", "HPが最大のとき、攻撃ダメージを50上昇させる。"},
	"shincho":  {":ganbare-:", "周囲1マス以内の味方の攻撃ダメージを10上昇させ、自身のターン終了時にHPを10回復する。"},
	"zina":     {"補給拠点", "周囲1マス以内にいる味方のパッシブ効果値を20上昇させる。"},
	"dana":     {"毒物耐性", "デバフの影響を受けない。デバフマスによるダメージや移動コスト増加は受ける。"},
}

// PassiveValues contains every numeric value used by a character passive.
// Keeping these beside the base stats and attacks makes character balancing a
// data-only change instead of requiring edits to the battle engine.
type PassiveValues struct {
	AttackBoost         int
	DamageReduction     int
	ExtraEffectChance   int
	FullHPAttackBoost   int
	TurnHeal            int
	PassiveValueBoost   int
	IgnorePassiveReduce bool
	ExcludeSelf         bool
}

var passiveValues = map[string]PassiveValues{
	"sophie":  {AttackBoost: 20, ExcludeSelf: true},
	"jude":    {DamageReduction: 20},
	"nadia":   {ExtraEffectChance: 20},
	"aoi":     {TurnHeal: 30},
	"sena":    {IgnorePassiveReduce: true},
	"chiyo":   {FullHPAttackBoost: 50},
	"shincho": {AttackBoost: 10, TurnHeal: 10},
	"zina":    {PassiveValueBoost: 20},
}

func passiveFor(id string) PassiveValues { return passiveValues[id] }

func init() {
	for i := range Definitions {
		passive := passiveDefinitions[Definitions[i].ID]
		Definitions[i].PassiveName = passive[0]
		Definitions[i].PassiveDescription = passive[1]
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
