package micro

import (
	"bitbucket.org/aisee/sc2lib"
	"bitbucket.org/aisee/veterran/bot"
	"github.com/chippydip/go-sc2ai/enums/ability"
	"github.com/chippydip/go-sc2ai/enums/effect"
	"math/rand"
)

type WidowMine struct {
	*Unit
}

func NewWidowMine(u *scl.Unit) *WidowMine {
	return &WidowMine{Unit: NewUnit(u)}
}

func WidowMinesLogic(us scl.Units) {
	for _, u := range us {
		NewWidowMine(u).Logic()
	}
}

func WidowMinesRetreatLogic(us scl.Units) {
	for _, u := range us {
		m := NewWidowMine(u)
		if m.Hits < m.HitsMax {
			B.Groups.Add(bot.MechRetreat, m.Unit.Unit)
			continue
		}
		if m.IsBurrowed && m.HasAbility(ability.Smart) {
			B.Groups.Add(bot.WidowMines, m.Unit.Unit)
			continue
		}
		if m.IsIdle() {
			vec := (B.Locs.EnemyStart - m.Point()).Norm()
			p1 := m.Point() - vec*20
			p2 := p1
			if rand.Intn(2) == 1 {
				vec *= 1i
			} else {
				vec *= -1i
			}
			for {
				if !B.IsPathable(p2 + vec*10) {
					break
				}
				p2 += vec * 10
			}

			m.CommandPos(ability.Move, p1)
			m.CommandPosQueue(ability.Move, p2)
			m.CommandQueue(ability.BurrowDown_WidowMine)
		}
	}
}

func (u *WidowMine) Retreat() bool {
	return false
}

func (u *WidowMine) Maneuver() bool {
	attackers := B.Enemies.AllReady.CanAttack(u.Unit.Unit, 2)
	// Someone is attacking mine, but it can't attack anyone
	detected := false
	if u.HPS > 0 {
		// detected = targets.InRangeOf(mine, 0).Empty() - this is wrong? Mine has no weapon?
		detected = Targets.ForMines.First(func(unit *scl.Unit) bool {
			return u.Dist(unit) <= float64(u.Radius+unit.Radius+5)
		}) == nil
	}
	safePos, safe := u.EvadeEffectsPos(u, false, effect.PsiStorm, effect.CorrosiveBile)
	if !safe {
		detected = true
	}

	if u.IsBurrowed && (detected ||
		!u.HasAbility(ability.Smart) || // Reloading
		Targets.ForMines.CloserThan(10, u).Empty() && attackers.Empty()) {
		// No targets or enemies around
		if u.Hits < u.HitsMax/2 {
			B.Groups.Add(bot.MechRetreat, u.Unit.Unit)
		} else {
			B.Groups.Add(bot.WidowMinesRetreat, u.Unit.Unit)
		}
		u.Command(ability.BurrowUp_WidowMine)
		return true
	}

	if !safe {
		u.CommandPos(ability.Move, safePos)
		return true
	}

	targetIsClose := Targets.ForMines.CloserThan(4, u).Exists() // For enemies that can't attack ground
	if !u.IsBurrowed && !detected && (attackers.Exists() || targetIsClose) {
		u.Command(ability.BurrowDown_WidowMine)
		return true
	}

	if Targets.ForMines.Exists() {
		u.CommandPos(ability.Move, Targets.ForMines.ClosestTo(u))
		return true
	}

	return false
}

func (u *WidowMine) Attack() bool {
	return false
}
