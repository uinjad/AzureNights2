package combat

import (
	"errors"
	"fmt"
	"math/rand"

	"github.com/uinjad/AzureNights2/internal/domain/faction"
)

type Phase int

const (
	Ongoing Phase = iota
	PlayerWon
	PlayerLost
)

var (
	ErrNotPlayerTurn = errors.New("combat: not the player's turn")
	ErrInvalidTarget = errors.New("combat: invalid or defeated target")
)

// Damage roll bounds and crit multiplier — tuned by the balance simulator.
const (
	varianceLo = 0.85
	varianceHi = 1.15
	critMult   = 2.0
)

// Battle is the turn-based state machine. Randomness and the faction table are
// injected, so the engine stays deterministic under test and lock-free at run.
type Battle struct {
	order    []*Combatant
	turn     int
	Round    int
	Phase    Phase
	Log      []string
	AI       func(b *Battle, enemy *Combatant)
	rng      func() float64
	factions *faction.Table
}

type Option func(*Battle)

func WithRNG(rng func() float64) Option    { return func(b *Battle) { b.rng = rng } }
func WithFactions(t *faction.Table) Option { return func(b *Battle) { b.factions = t } }

func NewBattle(player *Combatant, enemies []*Combatant, opts ...Option) *Battle {
	all := append([]*Combatant{player}, enemies...)
	b := &Battle{order: Order(all), Round: 1, Phase: Ongoing, AI: BasicAI, rng: rand.Float64}
	for _, o := range opts {
		o(b)
	}
	b.logf("⚔ Battle begins!")
	return b
}

func (b *Battle) Current() *Combatant { return b.order[b.turn] }

func (b *Battle) IsPlayerTurn() bool {
	return b.Phase == Ongoing && b.Current().Side == SidePlayer
}

func (b *Battle) Player() *Combatant {
	for _, c := range b.order {
		if c.Side == SidePlayer {
			return c
		}
	}
	return nil
}

func (b *Battle) Enemies() []*Combatant {
	var out []*Combatant
	for _, c := range b.order {
		if c.Side == SideEnemy {
			out = append(out, c)
		}
	}
	return out
}

func (b *Battle) PlayerAttack(targetIdx int) error {
	if !b.IsPlayerTurn() {
		return ErrNotPlayerTurn
	}
	target, err := b.enemyAt(targetIdx)
	if err != nil {
		return err
	}
	b.resolveAttack(b.Current(), target)
	return nil
}

func (b *Battle) PlayerUseSkill(s Skill, targetIdx int) error {
	if !b.IsPlayerTurn() {
		return ErrNotPlayerTurn
	}
	if !b.Current().CanUse(s) {
		return ErrSkillUnavailable
	}
	target, err := b.enemyAt(targetIdx)
	if err != nil {
		return err
	}
	b.resolveSkill(b.Current(), s, target)
	return nil
}

func (b *Battle) Step() error {
	if b.Phase != Ongoing {
		return nil
	}
	if b.Current().Side == SidePlayer {
		return ErrNotPlayerTurn
	}
	b.AI(b, b.Current())
	return nil
}

func BasicAI(b *Battle, enemy *Combatant) { b.resolveAttack(enemy, b.Player()) }

func (b *Battle) enemyAt(idx int) (*Combatant, error) {
	enemies := b.Enemies()
	if idx < 0 || idx >= len(enemies) || !enemies[idx].IsAlive() {
		return nil, ErrInvalidTarget
	}
	return enemies[idx], nil
}

// hit resolves one strike: subtractive base, then variance, then a crit roll
// against the attacker's DEX-derived crit chance, then the faction multiplier.
func (b *Battle) hit(attacker, target *Combatant, kind DamageKind, power int, label string) {
	var atk, def int
	if kind == Magical {
		atk, def = attacker.Stats.MAtk, target.Stats.MDef
	} else {
		atk, def = attacker.Stats.PAtk, target.Stats.PDef
	}
	base := PhysicalDamage(atk+power, def)
	dmg := float64(base) * (varianceLo + b.rng()*(varianceHi-varianceLo))
	crit := b.rng() < float64(attacker.Stats.Crit)/100.0
	if crit {
		dmg *= critMult
	}
	if b.factions != nil {
		dmg *= b.factions.DamageMultiplier(attacker.Faction, target.Faction)
	}
	final := int(dmg)
	if final < 1 {
		final = 1
	}
	target.HP -= final
	if target.HP < 0 {
		target.HP = 0
	}

	if label == "" {
		b.logf("%s hits %s for %d%s", attacker.Name, target.Name, final, critTag(crit))
	} else {
		b.logf("%s casts %s on %s for %d%s", attacker.Name, label, target.Name, final, critTag(crit))
	}
	if !target.IsAlive() {
		b.logf("%s is defeated!", target.Name)
	}
}

func critTag(c bool) string {
	if c {
		return " — CRIT!"
	}
	return ""
}

func (b *Battle) resolveAttack(attacker, target *Combatant) {
	b.hit(attacker, target, Physical, 0, "")
	b.afterAction()
}

func (b *Battle) resolveSkill(caster *Combatant, s Skill, target *Combatant) {
	caster.MP -= s.MPCost
	if s.Cooldown > 0 {
		caster.cooldowns[s.ID] = s.Cooldown
	}
	b.hit(caster, target, s.Kind, s.Power, s.Name)
	b.afterAction()
}

func (b *Battle) afterAction() {
	b.checkEnd()
	if b.Phase == Ongoing {
		b.advanceTurn()
	}
}

func (b *Battle) checkEnd() {
	if !b.Player().IsAlive() {
		b.Phase = PlayerLost
		b.logf("You have fallen…")
		return
	}
	for _, e := range b.Enemies() {
		if e.IsAlive() {
			return
		}
	}
	b.Phase = PlayerWon
	b.logf("Victory!")
}

func (b *Battle) advanceTurn() {
	for {
		b.turn++
		if b.turn >= len(b.order) {
			b.turn = 0
			b.Round++
		}
		if b.order[b.turn].IsAlive() {
			break
		}
	}
	b.order[b.turn].tickCooldowns()
}

func (b *Battle) logf(format string, args ...any) {
	b.Log = append(b.Log, fmt.Sprintf(format, args...))
}
