package app

import (
	"fmt"
	"math/rand"

	"github.com/uinjad/AzureNights2/internal/content"
	"github.com/uinjad/AzureNights2/internal/domain/character"
	"github.com/uinjad/AzureNights2/internal/domain/class"
	"github.com/uinjad/AzureNights2/internal/domain/combat"
	"github.com/uinjad/AzureNights2/internal/domain/stats"
	"github.com/uinjad/AzureNights2/internal/domain/world"
)

const respawnDelay = 20

// Session is the live game. The UI drives it through these use-cases and never
// touches the domain packages directly.
type Session struct {
	reg  *content.Registry
	repo Repository
	roll func() float64

	Hero      *character.Character
	MapID     string
	PlayerPos world.Point
	Clock     world.Clock
	Spawns    []Spawn

	Battle   *combat.Battle
	curSpawn int
	Log      []string

	pending []PendingRespawn
}

// Option configures a Session at construction.
type Option func(*Session)

// WithRoll injects the randomness source for the world clock, so tests can make
// the weather deterministic.
func WithRoll(roll func() float64) Option {
	return func(s *Session) { s.roll = roll }
}

// New builds a session bound to loaded content and a persistence adapter.
func New(reg *content.Registry, repo Repository, opts ...Option) *Session {
	s := &Session{reg: reg, repo: repo, roll: rand.Float64, curSpawn: -1}
	for _, o := range opts {
		o(s)
	}
	return s
}

func (s *Session) NewGame(heroName string) error {
	hero, err := character.New(heroName, s.reg.Classes)
	if err != nil {
		return err
	}
	md, ok := s.reg.Maps["forest"]
	if !ok {
		return fmt.Errorf("app: starting map %q not found", "forest")
	}
	for _, id := range []string{"iron_sword", "padded_robe"} { // starter loadout
		if it, ok := s.reg.Items[id]; ok {
			hero.AddItem(it)
		}
	}
	s.Hero = hero
	s.Clock = world.Clock{}
	s.Battle, s.curSpawn = nil, -1
	s.enterMap("forest", md.Spawn)
	s.logf("%s sets out into %s.", hero.Name, md.Name)
	return nil
}

// enterMap places the hero on a map and resets that map's enemies from its
// definition. Spawn state is local to the current map, so leaving and returning
// gives a clean board — simple, and enough for the MVP.
func (s *Session) enterMap(mapID string, at world.Point) {
	md := s.reg.Maps[mapID]
	s.MapID, s.PlayerPos = mapID, at
	s.Spawns = s.Spawns[:0]
	for _, e := range md.Enemies {
		s.Spawns = append(s.Spawns, Spawn{Pos: e.Pos, DefID: e.DefID})
	}
	s.pending = s.pending[:0]
}

// InBattle reports whether a fight is in progress.
func (s *Session) InBattle() bool { return s.Battle != nil }

// GameOver reports whether the hero has fallen and the run is over.
func (s *Session) GameOver() bool {
	return s.Hero != nil && s.Hero.HP <= 0 && !s.InBattle()
}

// Map returns the tiles of the current map for rendering.
func (s *Session) Map() *world.TileMap { return s.currentMap() }

func (s *Session) currentMap() *world.TileMap { return s.reg.Maps[s.MapID].Map }

func (s *Session) Move(dir world.Direction) error {
	if s.InBattle() {
		return ErrBusy
	}
	next, ok := s.currentMap().Step(s.PlayerPos, dir)
	if !ok {
		return nil // walked into a wall; harmless no-op
	}
	if p, ok := s.portalAt(next); ok {
		s.enterMap(p.ToMap, p.ToPos)
		s.logf("You travel to %s.", s.reg.Maps[p.ToMap].Name)
		return nil
	}
	if idx := s.spawnAt(next); idx >= 0 {
		s.startBattle(idx)
		return nil
	}
	s.PlayerPos = next
	if s.restAt(next) {
		s.restHero()
	}
	return nil
}

func (s *Session) portalAt(p world.Point) (content.Portal, bool) {
	for _, pt := range s.reg.Maps[s.MapID].Portals {
		if pt.At == p {
			return pt, true
		}
	}
	return content.Portal{}, false
}

func (s *Session) restAt(p world.Point) bool {
	for _, r := range s.reg.Maps[s.MapID].Rests {
		if r == p {
			return true
		}
	}
	return false
}

func (s *Session) restHero() {
	d, _ := s.Hero.EffectiveStats(s.reg.Classes)
	s.Hero.HP, s.Hero.MP = d.MaxHP, d.MaxMP
	s.logf("You rest at the campfire — HP and MP restored.")
}

func (s *Session) Tick() {
	if s.InBattle() {
		return
	}
	for _, note := range s.Clock.Advance(s.roll) {
		s.logf("%s", note)
	}
	s.processRespawns()
}

// processRespawns returns queued enemies to the map once their timer is up and
// their tile is clear (and the hero isn't standing on it).
func (s *Session) processRespawns() {
	keep := s.pending[:0]
	for _, pr := range s.pending {
		ready := s.Clock.Tick >= pr.AtTick && s.spawnAt(pr.Pos) < 0 && s.PlayerPos != pr.Pos
		if ready {
			s.Spawns = append(s.Spawns, Spawn{Pos: pr.Pos, DefID: pr.DefID})
			s.logf("A %s prowls back into view.", s.reg.Enemies[pr.DefID].Name)
		} else {
			keep = append(keep, pr)
		}
	}
	s.pending = keep
}

// Attack performs the hero's basic attack on the chosen enemy.
func (s *Session) Attack(targetIdx int) error {
	if !s.InBattle() {
		return ErrNotInBattle
	}
	if err := s.Battle.PlayerAttack(targetIdx); err != nil {
		return err
	}
	s.resolveBattleProgress()
	return nil
}

// UseSkill casts a skill by ID on the chosen enemy.
func (s *Session) UseSkill(skillID string, targetIdx int) error {
	if !s.InBattle() {
		return ErrNotInBattle
	}
	sk, ok := s.reg.Skills[skillID]
	if !ok {
		return fmt.Errorf("app: unknown skill %q", skillID)
	}
	if err := s.Battle.PlayerUseSkill(sk, targetIdx); err != nil {
		return err
	}
	s.resolveBattleProgress()
	return nil
}

// AdvancementOptions lists the class branches the hero currently qualifies for.
func (s *Session) AdvancementOptions() []class.Class {
	return s.reg.Classes.Options(s.Hero.ClassID, s.Hero.Level)
}

// AdvanceClass advances the hero onto a new class branch.
func (s *Session) AdvanceClass(to class.ID) error {
	if s.InBattle() {
		return ErrBusy
	}
	if err := s.Hero.Advance(s.reg.Classes, to); err != nil {
		return err
	}
	c, _ := s.reg.Classes.Get(to)
	s.logf("%s becomes a %s!", s.Hero.Name, c.Name)
	return nil
}

// Equip puts an item from the content registry onto the hero.
func (s *Session) Equip(itemID string) error {
	it, ok := s.reg.Items[itemID]
	if !ok {
		return fmt.Errorf("app: unknown item %q", itemID)
	}
	if err := s.Hero.Equip(s.reg.Classes, it); err != nil {
		return err
	}
	s.logf("Equipped %s.", it.Name)
	return nil
}

// Save persists the current game through the repository port.
func (s *Session) Save() error { return s.repo.Save(s.snapshot()) }

func (s *Session) resolveBattleProgress() {
	for s.Battle.Phase == combat.Ongoing && !s.Battle.IsPlayerTurn() {
		_ = s.Battle.Step()
	}
	if s.Battle.Phase != combat.Ongoing {
		s.settleBattle()
	}
}

func (s *Session) settleBattle() {
	pc := s.Battle.Player()
	s.Hero.HP, s.Hero.MP = pc.HP, pc.MP // sync exploration pools back

	if s.Battle.Phase == combat.PlayerWon {
		sp := s.Spawns[s.curSpawn]
		def := s.reg.Enemies[sp.DefID]
		s.Hero.Gold += def.GoldReward
		levels, _ := s.Hero.AddXP(s.reg.Classes, def.XPReward)
		s.PlayerPos = sp.Pos
		s.removeSpawn(s.curSpawn)
		s.pending = append(s.pending, PendingRespawn{Pos: sp.Pos, DefID: sp.DefID, AtTick: s.Clock.Tick + respawnDelay})
		s.logf("You defeat %s! +%d XP, +%d gold.", def.Name, def.XPReward, def.GoldReward)
		if levels > 0 {
			s.logf("You reach level %d!", s.Hero.Level)
		}
	} else {
		s.logf("%s has fallen…", s.Hero.Name)
	}
	s.curSpawn, s.Battle = -1, nil
}

func (s *Session) spawnAt(p world.Point) int {
	for i, sp := range s.Spawns {
		if sp.Pos == p {
			return i
		}
	}
	return -1
}

func (s *Session) removeSpawn(i int) {
	s.Spawns = append(s.Spawns[:i], s.Spawns[i+1:]...)
}

func (s *Session) snapshot() *Snapshot {
	return &Snapshot{
		Hero: s.Hero, MapID: s.MapID, PlayerPos: s.PlayerPos,
		Clock: s.Clock, Spawns: s.Spawns, Pending: s.pending,
	}
}

func (s *Session) LoadGame() error {
	snap, err := s.repo.Load()
	if err != nil {
		return err
	}
	s.Hero, s.MapID, s.PlayerPos = snap.Hero, snap.MapID, snap.PlayerPos
	s.Clock, s.Spawns, s.pending = snap.Clock, snap.Spawns, snap.Pending
	s.Battle, s.curSpawn = nil, -1
	s.logf("Game loaded.")
	return nil
}

func (s *Session) logf(format string, a ...any) {
	s.Log = append(s.Log, fmt.Sprintf(format, a...))
}

// AdvanceTo advances the hero onto the named class branch.
func (s *Session) AdvanceTo(id string) error { return s.AdvanceClass(class.ID(id)) }

// EquipFromInventory equips the bag item at idx, returning whatever occupied
// that slot back to the bag.
func (s *Session) EquipFromInventory(idx int) error {
	inv := s.Hero.Inventory
	if idx < 0 || idx >= len(inv) {
		return ErrInvalidItem
	}
	it := inv[idx]
	s.Hero.Inventory = append(inv[:idx], inv[idx+1:]...) // take it out of the bag
	if old, ok := s.Hero.Equipment[it.Slot]; ok {
		s.Hero.Inventory = append(s.Hero.Inventory, old) // old gear goes back
	}
	if err := s.Hero.Equip(s.reg.Classes, it); err != nil {
		return err
	}
	s.logf("Equipped %s.", it.Name)
	return nil
}

func (s *Session) startBattle(idx int) {
	def := s.reg.Enemies[s.Spawns[idx].DefID]
	s.Battle = combat.NewBattle(
		s.buildHeroCombatant(),
		[]*combat.Combatant{s.buildEnemyCombatant(def)},
		combat.WithRNG(s.roll),
		combat.WithFactions(s.reg.Factions),
	)
	s.curSpawn = idx
	s.logf("A %s blocks your path!", def.Name)
	s.resolveBattleProgress()
}

func (s *Session) buildHeroCombatant() *combat.Combatant {
	d, _ := s.Hero.EffectiveStats(s.reg.Classes)
	cls, _ := s.reg.Classes.Get(s.Hero.ClassID)
	c := combat.NewCombatant(s.Hero.Name, "🧝", combat.SidePlayer, d)
	c.Faction = cls.Faction
	c.HP, c.MP = s.Hero.HP, s.Hero.MP
	return c
}

func (s *Session) buildEnemyCombatant(def content.EnemyDef) *combat.Combatant {
	st := scaleForLevel(def.Stats, s.Hero.Level)
	if bonus := s.Clock.EnemyPowerBonus(); bonus > 0 { // living-world hook
		st.PAtk += bonus
		st.MAtk += bonus
	}
	c := combat.NewCombatant(def.Name, def.Emoji, combat.SideEnemy, st)
	c.Faction = def.Faction
	return c
}

// scaleForLevel grows an enemy template with the hero's level so encounters stay
// meaningful as the hero advances. Level 1 is the authored baseline.
func scaleForLevel(d stats.Derived, level int) stats.Derived {
	steps := level - 1
	if steps <= 0 {
		return d
	}
	d.MaxHP += d.MaxHP * steps / 8
	d.PAtk += d.PAtk * steps / 10
	d.MAtk += d.MAtk * steps / 10
	d.PDef += d.PDef * steps / 12
	d.MDef += d.MDef * steps / 12
	d.Init += d.Init * steps / 20
	return d
}
