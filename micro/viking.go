package micro

import (
	"bitbucket.org/aisee/veterran/bot"
	"github.com/aiseeq/s2l/lib/point"
	"github.com/aiseeq/s2l/lib/scl"
	"github.com/aiseeq/s2l/protocol/enums/ability"
	"github.com/aiseeq/s2l/protocol/enums/buff"
)

func VikingsRetreat(u *scl.Unit) bool {
	if (u.Hits < u.HitsMax/2) || u.HasBuff(buff.RavenScramblerMissile) || u.HasBuff(buff.LockOn) {
		B.Groups.Add(bot.MechRetreat, u)
		return true
	}
	return false
}

func VikingsManeuver(u *scl.Unit) bool {
	if !u.IsHalfCool() {
		vikingPos := u.TargetPos()
		if vikingPos == 0 {
			vikingPos = u.Point()
		}
		pos, safe := u.AirEvade(B.Enemies.AllReady.CanAttack(u, 2), 2, vikingPos)
		if !safe || pos.IsFurtherThan(1, vikingPos) {
			u.CommandPos(ability.Move, pos)
			return true
		}
	}
	return false
}

func VikingsAttack(u *scl.Unit) bool {
	if Targets.ArmedFlyingArmored.Exists() || Targets.Flying.Exists() {
		u.Attack(Targets.ArmedFlyingArmored, Targets.Flying)
		return true
	}
	return false
}

func VikingExplore(u *scl.Unit) bool {
	friends := append(B.Groups.Get(bot.Medivacs).Units, B.Groups.Get(bot.Ravens).Units...)
	friends = append(friends, B.Groups.Get(bot.Banshees).Units...)
	friends = append(friends, B.Groups.Get(bot.Battlecruisers).Units...)
	if friends.Empty() {
		friends = B.Groups.Get(bot.Tanks).Units
	}
	if friends.Empty() {
		friends = B.Groups.Get(bot.Marauders).Units
	}
	if friends.Empty() {
		friends = B.Groups.Get(bot.Reapers).Units
	}
	if friends.Empty() {
		return false
	}
	enemiesCenter := B.Enemies.AllReady.Center()
	target := friends.ClosestTo(enemiesCenter)
	pos := target.Towards(enemiesCenter, 2)
	pos, safe := u.AirEvade(B.Enemies.AllReady.CanAttack(u, 2), 2, pos)
	if !safe || pos.IsFurtherThan(4, point.Pt3(u.Pos)) {
		if !safe {
			pos, _ = u.AirEvade(B.Enemies.AllReady.CanAttack(u, 2), 2, B.Groups.Get(bot.Vikings).Units.Center())
		}
		u.CommandPos(ability.Move, pos)
	}
	return true
}


func VikingsLogic(us scl.Units) {
	if us.Empty() {
		return
	}

	for _, u := range us {
		_ = VikingsRetreat(u) || VikingsManeuver(u) || VikingsAttack(u) || VikingExplore(u)
	}
}
