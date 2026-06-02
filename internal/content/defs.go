package content

import (
	"fmt"

	"github.com/uinjad/AzureNights2/internal/domain/class"
	"github.com/uinjad/AzureNights2/internal/domain/combat"
	"github.com/uinjad/AzureNights2/internal/domain/faction"
	"github.com/uinjad/AzureNights2/internal/domain/item"
)

// --- classes ---

type advanceDTO struct {
	To       string `json:"to"`
	MinLevel int    `json:"min_level"`
}

type classDTO struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	Faction  string       `json:"faction"`
	Bonus    primaryDTO   `json:"bonus"`
	Skills   []string     `json:"skills"`
	Advances []advanceDTO `json:"advances"`
}

type classFileDTO struct {
	Root    string     `json:"root"`
	Classes []classDTO `json:"classes"`
}

func loadClasses(skills map[string]combat.Skill) (*class.Tree, error) {
	file, err := readJSON[classFileDTO]("data/classes.json")
	if err != nil {
		return nil, err
	}
	out := make([]class.Class, 0, len(file.Classes))
	for _, c := range file.Classes {
		for _, sid := range c.Skills {
			if _, ok := skills[sid]; !ok {
				return nil, fmt.Errorf("content: class %q references unknown skill %q", c.ID, sid)
			}
		}
		adv := make([]class.Advance, 0, len(c.Advances))
		for _, a := range c.Advances {
			adv = append(adv, class.Advance{To: class.ID(a.To), MinLevel: a.MinLevel})
		}
		out = append(out, class.Class{
			ID:       class.ID(c.ID),
			Name:     c.Name,
			Faction:  faction.ID(c.Faction),
			Bonus:    c.Bonus.toDomain(),
			Skills:   c.Skills,
			Advances: adv,
		})
	}
	return class.NewTree(class.ID(file.Root), out...)
}

// --- skills ---

type skillDTO struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Emoji    string `json:"emoji"`
	Kind     string `json:"kind"`
	MPCost   int    `json:"mp_cost"`
	Cooldown int    `json:"cooldown"`
	Power    int    `json:"power"`
}

func loadSkills() (map[string]combat.Skill, error) {
	list, err := readJSON[[]skillDTO]("data/skills.json")
	if err != nil {
		return nil, err
	}
	out := make(map[string]combat.Skill, len(list))
	for _, s := range list {
		kind, err := parseDamageKind(s.Kind)
		if err != nil {
			return nil, fmt.Errorf("content: skill %q: %w", s.ID, err)
		}
		out[s.ID] = combat.Skill{
			ID: s.ID, Name: s.Name, Emoji: s.Emoji, Kind: kind,
			MPCost: s.MPCost, Cooldown: s.Cooldown, Power: s.Power,
		}
	}
	return out, nil
}

// --- items ---

type itemDTO struct {
	ID    string     `json:"id"`
	Name  string     `json:"name"`
	Emoji string     `json:"emoji"`
	Kind  string     `json:"kind"`
	Slot  string     `json:"slot"`
	Bonus derivedDTO `json:"bonus"`
	Heal  int        `json:"heal"`
	Mana  int        `json:"mana"`
}

func loadItems() (map[string]item.Item, error) {
	list, err := readJSON[[]itemDTO]("data/items.json")
	if err != nil {
		return nil, err
	}
	out := make(map[string]item.Item, len(list))
	for _, it := range list {
		kind := item.Gear
		if it.Kind == "potion" {
			kind = item.Potion
		}
		var slot item.Slot
		if kind == item.Gear {
			s, err := parseSlot(it.Slot)
			if err != nil {
				return nil, fmt.Errorf("content: item %q: %w", it.ID, err)
			}
			slot = s
		}
		out[it.ID] = item.Item{
			ID: it.ID, Name: it.Name, Emoji: it.Emoji, Kind: kind,
			Slot: slot, Bonus: it.Bonus.toDomain(), Heal: it.Heal, Mana: it.Mana,
		}
	}
	return out, nil
}

// --- enemies ---

type enemyDTO struct {
	ID    string     `json:"id"`
	Name  string     `json:"name"`
	Emoji string     `json:"emoji"`
	Fact  string     `json:"faction"`
	Stats derivedDTO `json:"stats"`
	XP    int        `json:"xp"`
	Gold  int        `json:"gold"`
	Drop  string     `json:"drop"`
}

func loadEnemies() (map[string]EnemyDef, error) {
	list, err := readJSON[[]enemyDTO]("data/enemies.json")
	if err != nil {
		return nil, err
	}
	out := make(map[string]EnemyDef, len(list))
	for _, e := range list {
		out[e.ID] = EnemyDef{
			ID: e.ID, Name: e.Name, Emoji: e.Emoji, Faction: faction.ID(e.Fact),
			Stats: e.Stats.toDomain(), XPReward: e.XP, GoldReward: e.Gold, Drop: e.Drop,
		}
	}
	return out, nil
}
