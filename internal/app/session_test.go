package app

import (
	"errors"
	"testing"

	"github.com/uinjad/AzureNights2/internal/content"
	"github.com/uinjad/AzureNights2/internal/domain/stats"
	"github.com/uinjad/AzureNights2/internal/domain/world"
)

type fakeRepo struct{ snap *Snapshot }

func (r *fakeRepo) Save(s *Snapshot) error { r.snap = s; return nil }
func (r *fakeRepo) Load() (*Snapshot, error) {
	if r.snap == nil {
		return nil, errors.New("no save")
	}
	return r.snap, nil
}
func (r *fakeRepo) Exists() bool { return r.snap != nil }

func newTestSession(t *testing.T) *Session {
	t.Helper()
	reg, err := content.Load()
	if err != nil {
		t.Fatalf("content.Load: %v", err)
	}
	s := New(reg, &fakeRepo{}, WithRoll(func() float64 { return 1.0 })) // weather never shifts
	if err := s.NewGame("Hero"); err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	return s
}

// approach stands the hero on a walkable neighbor of target and steps onto it.
func approach(t *testing.T, s *Session, target world.Point) {
	t.Helper()
	tries := []struct {
		dir  world.Direction
		from world.Point
	}{
		{world.North, world.Point{X: target.X, Y: target.Y + 1}},
		{world.South, world.Point{X: target.X, Y: target.Y - 1}},
		{world.East, world.Point{X: target.X - 1, Y: target.Y}},
		{world.West, world.Point{X: target.X + 1, Y: target.Y}},
	}
	for _, tr := range tries {
		if s.currentMap().Walkable(tr.from) && s.spawnAt(tr.from) < 0 {
			s.PlayerPos = tr.from
			_ = s.Move(tr.dir)
			return
		}
	}
	t.Fatal("no approach tile found")
}

func TestNewGameSeedsWorld(t *testing.T) {
	s := newTestSession(t)
	if s.Hero == nil || s.Hero.Name != "Hero" {
		t.Fatal("hero not created")
	}
	if len(s.Spawns) == 0 {
		t.Error("expected enemy spawns from the map")
	}
	if !s.currentMap().Walkable(s.PlayerPos) {
		t.Error("hero should spawn on a walkable tile")
	}
}

func TestMoveBlockedByWall(t *testing.T) {
	s := newTestSession(t)
	y := s.PlayerPos.Y
	for i := 0; i < 10; i++ {
		_ = s.Move(world.West)
	}
	if s.PlayerPos.X != 1 || s.PlayerPos.Y != y {
		t.Errorf("expected to stop at the western wall (x=1), got %+v", s.PlayerPos)
	}
}

func TestSteppingOntoEnemyStartsBattle(t *testing.T) {
	s := newTestSession(t)
	approach(t, s, s.Spawns[0].Pos)
	if !s.InBattle() {
		t.Fatal("stepping onto a spawn should start a battle")
	}
}

func TestWinningBattleRewardsAndClearsSpawn(t *testing.T) {
	s := newTestSession(t)
	beforeGold, beforeSpawns := s.Hero.Gold, len(s.Spawns)
	goblinPos := s.Spawns[0].Pos

	approach(t, s, goblinPos)
	for s.InBattle() {
		if err := s.Attack(0); err != nil {
			t.Fatalf("Attack: %v", err)
		}
	}
	if s.Hero.Gold <= beforeGold {
		t.Errorf("winning should grant gold: %d -> %d", beforeGold, s.Hero.Gold)
	}
	if len(s.Spawns) != beforeSpawns-1 {
		t.Errorf("defeated spawn should be removed")
	}
	if s.PlayerPos != goblinPos {
		t.Errorf("hero should move onto the cleared tile, got %+v", s.PlayerPos)
	}
}

func TestNightStrengthensEncounter(t *testing.T) {
	s := newTestSession(t)
	s.Clock.TimeOfDay = world.Night

	approach(t, s, s.Spawns[0].Pos)
	if !s.InBattle() {
		t.Fatal("battle should start")
	}
	base := s.reg.Enemies["goblin"].Stats.PAtk
	if got := s.Battle.Enemies()[0].Stats.PAtk; got <= base {
		t.Errorf("night should boost enemy PAtk: base %d, got %d", base, got)
	}
}

func TestTickPausesDuringBattle(t *testing.T) {
	s := newTestSession(t)
	before := s.Clock.Tick
	s.Tick()
	if s.Clock.Tick != before+1 {
		t.Errorf("tick should advance the clock: %d -> %d", before, s.Clock.Tick)
	}
	approach(t, s, s.Spawns[0].Pos)
	frozen := s.Clock.Tick
	s.Tick()
	if s.Clock.Tick != frozen {
		t.Error("clock must not advance during battle")
	}
}

func TestEquipFromRegistry(t *testing.T) {
	s := newTestSession(t)
	before, _ := s.Hero.EffectiveStats(s.reg.Classes)
	if err := s.Equip("iron_sword"); err != nil {
		t.Fatalf("Equip: %v", err)
	}
	after, _ := s.Hero.EffectiveStats(s.reg.Classes)
	if after.PAtk <= before.PAtk {
		t.Errorf("a sword should raise PAtk: %d -> %d", before.PAtk, after.PAtk)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	reg, _ := content.Load()
	repo := &fakeRepo{}
	s := New(reg, repo, WithRoll(func() float64 { return 1.0 }))
	_ = s.NewGame("Hero")
	s.PlayerPos = world.Point{X: 1, Y: 1}
	s.Hero.Gold = 99
	if err := s.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	s2 := New(reg, repo, WithRoll(func() float64 { return 1.0 }))
	if err := s2.LoadGame(); err != nil {
		t.Fatalf("LoadGame: %v", err)
	}
	if s2.PlayerPos != (world.Point{X: 1, Y: 1}) || s2.Hero.Gold != 99 {
		t.Errorf("state not restored: pos %+v gold %d", s2.PlayerPos, s2.Hero.Gold)
	}
}

func TestDefeatedEnemyRespawnsAfterDelay(t *testing.T) {
	s := newTestSession(t)
	goblin := s.Spawns[0].Pos

	approach(t, s, goblin)
	for s.InBattle() {
		if err := s.Attack(0); err != nil {
			t.Fatalf("Attack: %v", err)
		}
	}
	if s.spawnAt(goblin) >= 0 {
		t.Fatal("spawn should be cleared right after the win")
	}

	_ = s.Move(world.West) // step off the cleared tile so the respawn has room
	for i := 0; i < respawnDelay+1; i++ {
		s.Tick()
	}
	if s.spawnAt(goblin) < 0 {
		t.Errorf("enemy should respawn at %+v after %d ticks", goblin, respawnDelay)
	}
}

func TestPortalTravelsBetweenMaps(t *testing.T) {
	s := newTestSession(t)
	s.PlayerPos = world.Point{X: 7, Y: 2} // just west of the forest portal at (8,2)
	if err := s.Move(world.East); err != nil {
		t.Fatalf("Move: %v", err)
	}
	if s.MapID != "cavern" {
		t.Fatalf("portal should lead to the cavern, got %q", s.MapID)
	}
	if s.PlayerPos != (world.Point{X: 2, Y: 1}) {
		t.Errorf("arrived at %+v, want (2,1)", s.PlayerPos)
	}
}

func TestCampfireRestoresPools(t *testing.T) {
	s := newTestSession(t)
	s.PlayerPos = world.Point{X: 3, Y: 3} // west of the campfire at (4,3)
	s.Hero.HP, s.Hero.MP = 1, 0

	if err := s.Move(world.East); err != nil {
		t.Fatalf("Move: %v", err)
	}
	d, _ := s.Hero.EffectiveStats(s.reg.Classes)
	if s.Hero.HP != d.MaxHP || s.Hero.MP != d.MaxMP {
		t.Errorf("campfire should refill pools: HP %d/%d MP %d/%d", s.Hero.HP, d.MaxHP, s.Hero.MP, d.MaxMP)
	}
}

func TestScaleForLevelGrowsWithLevel(t *testing.T) {
	base := stats.Derived{MaxHP: 100, PAtk: 20, PDef: 10, Init: 8}
	if got := scaleForLevel(base, 1); got != base {
		t.Errorf("level 1 must be the baseline, got %+v", got)
	}
	hi := scaleForLevel(base, 9)
	if hi.MaxHP <= base.MaxHP || hi.PAtk <= base.PAtk {
		t.Errorf("higher level should scale enemies up: %+v", hi)
	}
}
