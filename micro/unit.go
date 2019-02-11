package micro

import (
	"bitbucket.org/aisee/sc2lib/scl"
	"bitbucket.org/aisee/veterran/bot"
	"github.com/chippydip/go-sc2ai/enums/ability"
	"github.com/chippydip/go-sc2ai/enums/terran"
	"math/rand"
)

func DefaultRetreat(u *scl.Unit) bool {
	if u.Hits < u.HitsMax/2 {
		B.Groups.Add(bot.MechRetreat, u)
		return true
	}
	return false
}

func DefaultManeuver(u *scl.Unit) bool {
	if !u.IsHalfCool() {
		closeTargets := Targets.Armed.InRangeOf(u, -0.5)
		if closeTargets.Exists() {
			u.GroundFallback(B.Enemies.AllReady, 2, B.Locs.MyStart-B.Locs.MyStartMinVec*3)
			return true
		}
	}
	return false
}

func DefaultAttack(u *scl.Unit) bool {
	if Targets.All.Exists() {
		u.Attack(Targets.Armed, Targets.All)
		return true
	}
	return false
}

func DefaultExplore(u *scl.Unit) bool {
	if B.PlayDefensive {
		pos := B.Ramps.My.Top
		bunkers := B.Units.My[terran.Bunker]
		if bunkers.Exists() {
			bunkers.OrderByDistanceTo(B.Locs.MyStart, false)
			pos = bunkers[int(u.Tag)%bunkers.Len()].Point()
		}
		if u.IsFarFrom(pos) {
			u.CommandPos(ability.Move, pos)
		}
		return true
	}
	if !B.Grid.IsExplored(B.Locs.EnemyStart) {
		u.CommandPos(ability.Attack, B.Locs.EnemyStart)
	} else {
		// Search for other bases
		if u.IsIdle() {
			pos := B.Locs.EnemyExps[rand.Intn(len(B.Locs.EnemyExps))]
			u.CommandPos(ability.Move, pos)
		}
	}
	return true
}
