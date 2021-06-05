package micro

import (
	"bitbucket.org/aisee/veterran/bot"
	"github.com/aiseeq/s2l/lib/point"
	"github.com/aiseeq/s2l/lib/scl"
	"github.com/aiseeq/s2l/protocol/enums/ability"
	"github.com/aiseeq/s2l/protocol/enums/buff"
	"github.com/aiseeq/s2l/protocol/enums/terran"
	"math/rand"
)

func DefaultRetreat(u *scl.Unit) bool {
	if (u.Hits < u.HitsMax/2) || u.HasBuff(buff.RavenScramblerMissile) {
		B.Groups.Add(bot.MechRetreat, u)
		return true
	}
	return false
}

func DefaultManeuver(u *scl.Unit) bool {
	if !u.IsCoolToAttack() {
		attackers := B.Enemies.AllReady.CanAttack(u, 4)
		if attackers.Exists() {
			outranged, stronger := u.AssessStrength(attackers)
			if outranged && stronger {
				return false // Attack them
			}
			// Not outranged - we can safely fall back
			// Or we are weaker - fall back (there should be no attack)
			if scl.Ground(u) {
				u.GroundFallback(B.Locs.MyStart - B.Locs.MyStartMinVec*3)
				return true
			} else {
				target := u.TargetPos()
				if target == 0 {
					target = u.Point()
				} else {
					target = u.Point().Towards(target, 1)
				}
				pos, safe := u.AirEvade(attackers, 2, target)
				if !safe {
					u.CommandPos(ability.Move, pos)
					return true
				}
			}
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

func GetDefensivePos(u *scl.Unit) point.Point {
	pos := B.Ramps.My.Top
	bunkers := B.Units.My[terran.Bunker]
	if bunkers.Exists() {
		bunkers.OrderByDistanceTo(B.Locs.MyStart, false)
		pos = bunkers[int(u.Tag)%bunkers.Len()].Point()
	}
	return pos
}

func DefaultExplore(u *scl.Unit) bool {
	if B.PlayDefensive {
		pos := GetDefensivePos(u)
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
