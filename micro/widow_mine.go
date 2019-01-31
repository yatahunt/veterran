package micro

import (
	"bitbucket.org/aisee/sc2lib"
	"bitbucket.org/aisee/veterran/bot"
	"github.com/chippydip/go-sc2ai/enums/ability"
	"github.com/chippydip/go-sc2ai/enums/effect"
	"math/rand"
)

func WidowMineManeuver(u *scl.Unit) bool {
	attackers := B.Enemies.AllReady.CanAttack(u, 2)
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
			B.Groups.Add(bot.MechRetreat, u)
		} else {
			B.Groups.Add(bot.WidowMinesRetreat, u)
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

func WidowMinesLogic(us scl.Units) {
	for _, u := range us {
		_ = WidowMineManeuver(u) || DefaultExplore(u)
	}
}

func WidowMinesRetreatLogic(us scl.Units) {
	for _, u := range us {
		if u.Hits < u.HitsMax {
			B.Groups.Add(bot.MechRetreat, u)
			continue
		}
		if u.IsBurrowed && u.HasAbility(ability.Smart) {
			B.Groups.Add(bot.WidowMines, u)
			continue
		}
		if u.IsIdle() {
			vec := (B.Locs.EnemyStart - u.Point()).Norm()
			p1 := u.Point() - vec*20
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

			u.CommandPos(ability.Move, p1)
			u.CommandPosQueue(ability.Move, p2)
			u.CommandQueue(ability.BurrowDown_WidowMine)
		}
	}
}
