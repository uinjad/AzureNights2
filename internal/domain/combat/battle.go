package combat

import (
	"errors"
	"fmt"
)

// Phase is the high-level battle outcome state.
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

// Battle is the turn-based state machine. Drive it from outside: when it is the
// player's turn, supply a player action; otherwise call Step to let an enemy
// act. The same loop runs under a TUI today or a headless server later.
type Battle struct {
	order []*Combatant
	turn  int
	Round int
	Phase Phase
	Log   []string

	// AI decides an enemy's action on its turn. Swapping it changes enemy
	// behavior without touching the engine (Strategy pattern). Defaults to BasicAI.
	AI func(b *Battle, enemy *Combatant)
}

// NewBattle seats the player and enemies in initiative order.
func NewBattle(player *Combatant, enemies ...*Combatant) *Battle {
	all := append([]*Combatant{player}, enemies...)
	b := &Battle{
		order: Order(all),
		Round: 1,
		Phase: Ongoing,
		AI:    BasicAI,
	}
	b.logf("⚔ Battle begins!")
	return b
}

// Current is whoever acts now.
func (b *Battle) Current() *Combatant { return b.order[b.turn] }

// IsPlayerTurn reports whether the engine is waiting for player input.
func (b *Battle) IsPlayerTurn() bool {
	return b.Phase == Ongoing && b.Current().Side == SidePlayer
}

// Player returns the single player-controlled combatant.
func (b *Battle) Player() *Combatant {
	for _, c := range b.order {
		if c.Side == SidePlayer {
			return c
		}
	}
	return nil
}

// Enemies returns all enemy combatants in initiative order (alive or not).
func (b *Battle) Enemies() []*Combatant {
	var out []*Combatant
	for _, c := range b.order {
		if c.Side == SideEnemy {
			out = append(out, c)
		}
	}
	return out
}

// PlayerAttack performs a basic attack on the enemy at the given index.
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

// PlayerUseSkill casts a skill on the enemy at the given index.
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

// Step resolves the current enemy's action through the AI and advances the
// turn. It returns ErrNotPlayerTurn's inverse: an error only if it is actually
// the player's turn and the caller should be supplying input instead.
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

// BasicAI is the default enemy behavior: swing at the player.
func BasicAI(b *Battle, enemy *Combatant) {
	b.resolveAttack(enemy, b.Player())
}

func (b *Battle) enemyAt(idx int) (*Combatant, error) {
	enemies := b.Enemies()
	if idx < 0 || idx >= len(enemies) || !enemies[idx].IsAlive() {
		return nil, ErrInvalidTarget
	}
	return enemies[idx], nil
}

func (b *Battle) resolveAttack(attacker, target *Combatant) {
	res := attacker.Attack(target)
	b.logf("%s hits %s for %d.", res.Attacker, res.Target, res.Damage)
	if res.Defeated {
		b.logf("%s is defeated!", res.Target)
	}
	b.afterAction()
}

func (b *Battle) resolveSkill(caster *Combatant, s Skill, target *Combatant) {
	res, err := caster.UseSkill(s, target)
	if err != nil {
		return // callers guard with CanUse; defensive only
	}
	b.logf("%s casts %s on %s for %d.", res.Caster, res.Skill, res.Target, res.Damage)
	if res.Defeated {
		b.logf("%s is defeated!", res.Target)
	}
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
