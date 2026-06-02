package quest

import "testing"

func sampleSet(t *testing.T) *Set {
	t.Helper()
	s, err := NewSet(
		Quest{ID: "hunt", Name: "Hunt", Objectives: []Objective{
			{Kind: DefeatEnemy, Target: "goblin", Count: 2, Desc: "Defeat 2 goblins"},
		}, Reward: Reward{XP: 50, Gold: 10}},
		Quest{ID: "explore", Name: "Explore", Objectives: []Objective{
			{Kind: ReachMap, Target: "cavern", Desc: "Enter the cavern"},
			{Kind: DefeatEnemy, Target: "skeleton", Count: 1, Desc: "Defeat a skeleton"},
		}},
	)
	if err != nil {
		t.Fatalf("NewSet: %v", err)
	}
	return s
}

func TestApplyAdvancesMatchingObjectiveAndCaps(t *testing.T) {
	q, _ := sampleSet(t).Get("hunt")
	counts := make([]int, len(q.Objectives))
	if !q.Apply(counts, Event{Kind: DefeatEnemy, Target: "goblin"}) || counts[0] != 1 {
		t.Fatalf("a goblin kill should advance the hunt: %v", counts)
	}
	q.Apply(counts, Event{Kind: DefeatEnemy, Target: "goblin"})
	if changed := q.Apply(counts, Event{Kind: DefeatEnemy, Target: "goblin"}); changed || counts[0] != 2 {
		t.Errorf("progress should cap at 2, got %v changed=%v", counts, changed)
	}
	if !q.Complete(counts) {
		t.Error("two kills should complete the hunt")
	}
}

func TestApplyIgnoresUnrelatedEvents(t *testing.T) {
	q, _ := sampleSet(t).Get("hunt")
	counts := make([]int, len(q.Objectives))
	if q.Apply(counts, Event{Kind: DefeatEnemy, Target: "wolf"}) {
		t.Error("a wolf kill must not advance a goblin hunt")
	}
}

func TestMultiObjectiveNeedsAll(t *testing.T) {
	q, _ := sampleSet(t).Get("explore")
	counts := make([]int, len(q.Objectives))
	q.Apply(counts, Event{Kind: ReachMap, Target: "cavern"})
	if q.Complete(counts) {
		t.Error("not done until the skeleton falls too")
	}
	q.Apply(counts, Event{Kind: DefeatEnemy, Target: "skeleton"})
	if !q.Complete(counts) {
		t.Error("both objectives met should complete")
	}
}

func TestNewSetRejectsBadData(t *testing.T) {
	if _, err := NewSet(Quest{ID: "x"}); err == nil {
		t.Error("a quest with no objectives should be rejected")
	}
	if _, err := NewSet(
		Quest{ID: "d", Objectives: []Objective{{Kind: ReachMap, Target: "a"}}},
		Quest{ID: "d", Objectives: []Objective{{Kind: ReachMap, Target: "b"}}},
	); err == nil {
		t.Error("duplicate ids should be rejected")
	}
}
