package micro

import (
	"bitbucket.org/aisee/veterran/bot"
	"github.com/aiseeq/s2l/lib/scl"
	"github.com/aiseeq/s2l/protocol/enums/ability"
)

func RavensLogic(us scl.Units) {
	if us.Empty() {
		return
	}

	friends := append(B.Groups.Get(bot.Tanks).Units, B.Groups.Get(bot.Cyclones).Units...)
	friends = append(friends, B.Groups.Get(bot.Marines).Units...)
	friends = append(friends, B.Groups.Get(bot.Marauders).Units...)
	if friends.Empty() {
		friends = B.Groups.Get(bot.WidowMines).Units
	}
	if friends.Empty() {
		friends = B.Groups.Get(bot.Battlecruisers).Units
	}
	if friends.Empty() {
		friends = B.Groups.Get(bot.Reapers).Units
	}
	if friends.Empty() {
		return
	}

	enemiesCenter := B.Enemies.AllReady.Center()
	if enemiesCenter == 0 {
		enemiesCenter = B.Locs.EnemyStart
	}
	targets := []*scl.Unit{friends.ClosestTo(enemiesCenter)}
	if target := friends.Filter(func(unit *scl.Unit) bool {
		return unit.IsFurtherThan(8, targets[0])
	}).ClosestTo(enemiesCenter); target != nil {
		targets = append(targets, target)
	}

	us.OrderByDistanceTo(enemiesCenter, false)
	for n, u := range us {
		if u.TargetAbility() == ability.Effect_AutoTurret {
			continue // Let him finish placing
		}

		if DefaultRetreat(u) || u.EvadeEffects() {
			continue
		}

		if u.Energy >= 50 {
			closeEnemies := B.Enemies.AllReady.CloserThan(8, u)
			if closeEnemies.Exists() && closeEnemies.Sum(scl.CmpHits) >= 300 {
				enemy := closeEnemies.ClosestTo(u)
				pos := enemy.Towards(u, float64(enemy.Radius+2))
				pos = B.FindClosestPos(pos, scl.S2x2, ability.Effect_AutoTurret,
					0, 1, 1, scl.IsBuildable, scl.IsPathable)
				if pos != 0 {
					u.CommandPos(ability.Effect_AutoTurret, pos)
					continue
				}
			}
		}

		ravenPos := u.TargetPos()
		if ravenPos == 0 {
			ravenPos = u.Point()
		}
		pos := targets[scl.MinInt(n, len(targets)-1)].Towards(enemiesCenter, 2)
		enemies := B.Enemies.AllReady.CanAttack(u, 2)
		pos, safe := u.AirEvade(enemies, 2, pos)
		if !safe {
			outranged, stronger := u.AssessStrength(enemies)
			if !outranged || stronger {
				safe = true
			}
		}
		if !safe || pos.IsFurtherThan(1, ravenPos) {
			u.CommandPos(ability.Move, pos)
			continue
		}
	}
}
