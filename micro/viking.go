package micro

import (
	"bitbucket.org/aisee/veterran/bot"
	"github.com/aiseeq/s2l/lib/point"
	"github.com/aiseeq/s2l/lib/scl"
	"github.com/aiseeq/s2l/protocol/enums/ability"
	"github.com/aiseeq/s2l/protocol/enums/buff"
	"github.com/aiseeq/s2l/protocol/enums/terran"
)

func VikingsRetreat(u *scl.Unit) bool {
	if (u.Hits < u.HitsMax/2) || u.HasBuff(buff.RavenScramblerMissile) || u.HasBuff(buff.LockOn) {
		B.Groups.Add(bot.MechRetreat, u)
		return true
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
		// Это для тестов
		friends = B.Units.My.OfType(terran.CommandCenter)
	}
	if friends.Empty() {
		return false
	}
	enemiesCenter := B.Enemies.AllReady.Center()
	if enemiesCenter == 0 {
		enemiesCenter = B.Locs.EnemyStart
	}
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
		_ = VikingsRetreat(u) || DefaultManeuver(u) || VikingsAttack(u) || VikingExplore(u)
	}
}
