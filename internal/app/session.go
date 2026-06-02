package app

import (
	"fmt"
	"math/rand"

	"github.com/uinjad/AzureNights2/internal/content"
	"github.com/uinjad/AzureNights2/internal/domain/character"
	"github.com/uinjad/AzureNights2/internal/domain/class"
	"github.com/uinjad/AzureNights2/internal/domain/combat"
	"github.com/uinjad/AzureNights2/internal/domain/item"
	"github.com/uinjad/AzureNights2/internal/domain/quest"
	"github.com/uinjad/AzureNights2/internal/domain/world"
)

const respawnDelay = 20

// Session is the live game. The UI drives it through these use-cases.
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
	quests  []QuestProgress
	cleared map[string]bool
	won     bool
}

type Option func(*Session)

func WithRoll(roll func() float64) Option { return func(s *Session) { s.roll = roll } }

func New(reg *content.Registry, repo Repository, opts ...Option) *Session {
	s := &Session{reg: reg, repo: repo, roll: rand.Float64, curSpawn: -1, cleared: map[string]bool{}}
	for _, o := range opts {
		o(s)
	}
	return s
}

func (s *Session) Started() bool { return s.Hero != nil }
func (s *Session) HasSave() bool { return s.repo.Exists() }
func (s *Session) Won() bool     { return s.won }

func (s *Session) NewGame(heroName string) error {
	hero, err := character.New(heroName, s.reg.Classes)
	if err != nil {
		return err
	}
	md, ok := s.reg.Maps["forest"]
	if !ok {
		return fmt.Errorf("app: starting map %q not found", "forest")
	}
	for _, id := range []string{"iron_sword", "padded_robe"} {
		if it, ok := s.reg.Items[id]; ok {
			hero.AddItem(it)
		}
	}
	s.Hero = hero
	s.Clock = world.Clock{}
	s.Battle, s.curSpawn = nil, -1
	s.cleared = map[string]bool{}
	s.won = false

	s.quests = s.quests[:0]
	for _, q := range s.reg.Quests.All() {
		s.quests = append(s.quests, QuestProgress{ID: q.ID, Counts: make([]int, len(q.Objectives))})
	}

	s.enterMap("forest", md.Spawn)
	s.logf("%s sets out into %s.", hero.Name, md.Name)
	s.logf("Press 'c' to equip your starting gear.")
	return nil
}

func (s *Session) InBattle() bool { return s.Battle != nil }

func (s *Session) GameOver() bool {
	return s.Hero != nil && s.Hero.HP <= 0 && !s.InBattle()
}

func (s *Session) Map() *world.TileMap { return s.currentMap() }

func (s *Session) currentMap() *world.TileMap { return s.reg.Maps[s.MapID].Map }

func (s *Session) Move(dir world.Direction) error {
	if s.InBattle() {
		return ErrBusy
	}
	next, ok := s.currentMap().Step(s.PlayerPos, dir)
	if !ok {
		return nil
	}
	if p, ok := s.portalAt(next); ok {
		if p.Locked && !s.cleared[s.MapID] {
			s.logf("The way onward is sealed. Defeat %s first.", s.bossName())
			return nil
		}
		s.enterMap(p.ToMap, p.ToPos)
		s.logf("You travel to %s.", s.reg.Maps[p.ToMap].Name)
		return nil
	}
	if idx := s.spawnAt(next); idx >= 0 {
		s.startBattle(idx)
		return nil
	}
	s.PlayerPos = next
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

func (s *Session) bossName() string {
	md := s.reg.Maps[s.MapID]
	if md.Boss == "" {
		return "the guardian"
	}
	return s.reg.Enemies[md.Boss].Name
}

// Tick advances the living world by one step: time, weather, slow regeneration,
// and enemy respawns. Frozen during battle.
func (s *Session) Tick() {
	if s.InBattle() {
		return
	}
	for _, note := range s.Clock.Advance(s.roll) {
		s.logf("%s", note)
	}
	s.regen()
	s.processRespawns()
}

// regen trickles HP and MP back while exploring — the replacement for campfires.
// It is deliberately slow so potions and careful play still matter.
func (s *Session) regen() {
	if s.Hero == nil || s.Hero.HP <= 0 {
		return
	}
	d, _ := s.Hero.EffectiveStats(s.reg.Classes)
	if s.Hero.HP < d.MaxHP {
		s.Hero.HP = min(s.Hero.HP+max(1, d.MaxHP/40), d.MaxHP)
	}
	if s.Hero.MP < d.MaxMP {
		s.Hero.MP = min(s.Hero.MP+max(1, d.MaxMP/60), d.MaxMP)
	}
}

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

func (s *Session) AdvancementOptions() []class.Class {
	return s.reg.Classes.Options(s.Hero.ClassID, s.Hero.Level)
}

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

func (s *Session) AdvanceTo(id string) error { return s.AdvanceClass(class.ID(id)) }

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

func (s *Session) EquipFromInventory(idx int) error {
	inv := s.Hero.Inventory
	if idx < 0 || idx >= len(inv) {
		return ErrInvalidItem
	}
	it := inv[idx]
	if it.Kind != item.Gear {
		return ErrInvalidItem
	}
	s.Hero.Inventory = append(inv[:idx], inv[idx+1:]...)
	if old, ok := s.Hero.Equipment[it.Slot]; ok {
		s.Hero.Inventory = append(s.Hero.Inventory, old)
	}
	if err := s.Hero.Equip(s.reg.Classes, it); err != nil {
		return err
	}
	s.logf("Equipped %s.", it.Name)
	return nil
}

func (s *Session) UsePotion(idx int) error {
	inv := s.Hero.Inventory
	if idx < 0 || idx >= len(inv) || inv[idx].Kind != item.Potion {
		return ErrInvalidItem
	}
	p := inv[idx]
	d, _ := s.Hero.EffectiveStats(s.reg.Classes)
	s.Hero.HP = min(s.Hero.HP+p.Heal, d.MaxHP)
	s.Hero.MP = min(s.Hero.MP+p.Mana, d.MaxMP)
	s.Hero.Inventory = append(inv[:idx], inv[idx+1:]...)
	s.logf("Used %s.", p.Name)
	return nil
}

func (s *Session) Save() error {
	if err := s.repo.Save(s.snapshot()); err != nil {
		s.logf("Save failed: %v", err)
		return err
	}
	s.logf("Game saved.")
	return nil
}

func (s *Session) LoadGame() error {
	snap, err := s.repo.Load()
	if err != nil {
		return err
	}
	s.Hero, s.MapID, s.PlayerPos = snap.Hero, snap.MapID, snap.PlayerPos
	s.Clock, s.Spawns, s.pending = snap.Clock, snap.Spawns, snap.Pending
	s.quests, s.won = snap.Quests, snap.Won
	s.cleared = snap.Cleared
	if s.cleared == nil {
		s.cleared = map[string]bool{}
	}
	s.Battle, s.curSpawn = nil, -1
	s.logf("Game loaded.")
	return nil
}

// --- internals ---

func (s *Session) enterMap(mapID string, at world.Point) {
	md := s.reg.Maps[mapID]
	s.MapID, s.PlayerPos = mapID, at
	s.Spawns = s.Spawns[:0]
	for _, e := range md.Enemies {
		s.Spawns = append(s.Spawns, Spawn{Pos: e.Pos, DefID: e.DefID})
	}
	s.pending = s.pending[:0]
	s.fireQuestEvent(quest.Event{Kind: quest.ReachMap, Target: mapID})
	if md.Boss != "" && !s.cleared[mapID] {
		s.logf("A %s guards the way onward.", s.reg.Enemies[md.Boss].Name)
	}
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
	st := def.Stats
	if bonus := s.Clock.EnemyPowerBonus(); bonus > 0 {
		st.PAtk += bonus
		st.MAtk += bonus
	}
	c := combat.NewCombatant(def.Name, def.Emoji, combat.SideEnemy, st)
	c.Faction = def.Faction
	return c
}

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
	s.Hero.HP, s.Hero.MP = pc.HP, pc.MP

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
		s.rollLoot(def)
		s.checkBoss(sp.DefID)
		s.fireQuestEvent(quest.Event{Kind: quest.DefeatEnemy, Target: sp.DefID})
	} else {
		s.logf("%s has fallen…", s.Hero.Name)
	}
	s.curSpawn, s.Battle = -1, nil
}

// checkBoss marks a zone cleared (and the game won) when its boss falls.
func (s *Session) checkBoss(defID string) {
	md := s.reg.Maps[s.MapID]
	if md.Boss == "" || defID != md.Boss || s.cleared[s.MapID] {
		return
	}
	s.cleared[s.MapID] = true
	s.logf("You have cleared %s!", md.Name)
	if !hasLockedPortal(md) {
		s.won = true
		s.logf("%s falls. The realm is saved!", s.reg.Enemies[defID].Name)
	} else {
		s.logf("The way onward opens.")
	}
}

func hasLockedPortal(md content.MapDef) bool {
	for _, p := range md.Portals {
		if p.Locked {
			return true
		}
	}
	return false
}

func (s *Session) rollLoot(def content.EnemyDef) {
	if def.Drop != "" && s.roll() < 0.05 {
		if it, ok := s.reg.Items[def.Drop]; ok {
			s.Hero.AddItem(it)
			s.logf("%s dropped %s! ('c' to equip)", def.Name, it.Name)
		}
	}
	if s.roll() < 0.20 {
		id := "hp_potion"
		if s.roll() < 0.5 {
			id = "mp_potion"
		}
		if it, ok := s.reg.Items[id]; ok {
			s.Hero.AddItem(it)
			s.logf("Found a %s. ('c' to use)", it.Name)
		}
	}
}

func (s *Session) fireQuestEvent(e quest.Event) {
	for i := range s.quests {
		qp := &s.quests[i]
		if qp.Done {
			continue
		}
		def, ok := s.reg.Quests.Get(qp.ID)
		if !ok {
			continue
		}
		def.Apply(qp.Counts, e)
		if def.Complete(qp.Counts) {
			qp.Done = true
			s.Hero.Gold += def.Reward.Gold
			levels, _ := s.Hero.AddXP(s.reg.Classes, def.Reward.XP)
			s.logf("Quest complete: %s! +%d XP, +%d gold.", def.Name, def.Reward.XP, def.Reward.Gold)
			if levels > 0 {
				s.logf("You reach level %d!", s.Hero.Level)
			}
		}
	}
}

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
		Quests: s.quests, Cleared: s.cleared, Won: s.won,
	}
}

func (s *Session) logf(format string, a ...any) {
	s.Log = append(s.Log, fmt.Sprintf(format, a...))
}
