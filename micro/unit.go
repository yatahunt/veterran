package micro

import (
	"bitbucket.org/aisee/sc2lib"
	"github.com/chippydip/go-sc2ai/enums/ability"
	"github.com/chippydip/go-sc2ai/enums/terran"
	"math/rand"
)

type Unit struct {
	*scl.Unit
}

func NewUnit(u *scl.Unit) *Unit {
	return &Unit{u}
}

func (u *Unit) Retreat() bool {
	if u.Hits < u.HitsMax/2 {
		B.Groups.Add(MechRetreat, u.Unit)
		return true
	}
	return false
}

func (u *Unit) Maneuver() bool {
	if !u.IsCool() {
		attackers := B.Enemies.AllReady.CanAttack(u.Unit, 2)
		closeTargets := Targets.Armed.InRangeOf(u.Unit, -0.5)
		if attackers.Exists() || closeTargets.Exists() {
			u.GroundFallback(attackers, 2, B.HomePaths)
			return true
		}
	}
	return false
}

func (u *Unit) Cast() bool {
	return false
}

func (u *Unit) Attack() bool {
	if Targets.Armed.Exists() || Targets.All.Exists() {
		u.AttackCustom(scl.DefaultAttackFunc, scl.DefaultMoveFunc, Targets.Armed, Targets.All)
		return true
	}
	return false
}

func (u *Unit) Explore() bool {
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
	if !B.IsExplored(B.Locs.EnemyStart) {
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

func (u *Unit) Logic() {
	for _, f := range []func() bool{u.Retreat, u.Maneuver, u.Cast, u.Attack, u.Explore} {
		if f() {
			continue
		}
	}
}
