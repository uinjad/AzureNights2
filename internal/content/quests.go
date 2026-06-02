package content

import (
	"fmt"

	"github.com/uinjad/AzureNights2/internal/domain/quest"
)

type objectiveDTO struct {
	Kind   string `json:"kind"`
	Target string `json:"target"`
	Count  int    `json:"count"`
	Desc   string `json:"desc"`
}

type questDTO struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Summary    string         `json:"summary"`
	Objectives []objectiveDTO `json:"objectives"`
	Reward     struct {
		XP   int `json:"xp"`
		Gold int `json:"gold"`
	} `json:"reward"`
}

func parseObjectiveKind(s string) (quest.ObjectiveKind, error) {
	switch s {
	case "defeat":
		return quest.DefeatEnemy, nil
	case "reach":
		return quest.ReachMap, nil
	default:
		return 0, fmt.Errorf("unknown objective kind %q", s)
	}
}

// loadQuests builds the quest set, validating every objective target against the
// enemies and maps that exist.
func loadQuests(enemies map[string]EnemyDef, maps map[string]MapDef) (*quest.Set, error) {
	list, err := readJSON[[]questDTO]("data/quests.json")
	if err != nil {
		return nil, err
	}
	out := make([]quest.Quest, 0, len(list))
	for _, q := range list {
		objs := make([]quest.Objective, 0, len(q.Objectives))
		for _, o := range q.Objectives {
			kind, err := parseObjectiveKind(o.Kind)
			if err != nil {
				return nil, fmt.Errorf("content: quest %q: %w", q.ID, err)
			}
			switch kind {
			case quest.DefeatEnemy:
				if _, ok := enemies[o.Target]; !ok {
					return nil, fmt.Errorf("content: quest %q targets unknown enemy %q", q.ID, o.Target)
				}
			case quest.ReachMap:
				if _, ok := maps[o.Target]; !ok {
					return nil, fmt.Errorf("content: quest %q targets unknown map %q", q.ID, o.Target)
				}
			}
			objs = append(objs, quest.Objective{Kind: kind, Target: o.Target, Count: o.Count, Desc: o.Desc})
		}
		out = append(out, quest.Quest{
			ID: q.ID, Name: q.Name, Summary: q.Summary, Objectives: objs,
			Reward: quest.Reward{XP: q.Reward.XP, Gold: q.Reward.Gold},
		})
	}
	return quest.NewSet(out...)
}
