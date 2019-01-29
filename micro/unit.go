package micro

import (
	"bitbucket.org/aisee/sc2lib"
	"bitbucket.org/aisee/veterran/bot"
	"github.com/chippydip/go-sc2ai/enums/ability"
	"math/rand"
)

type Unit struct {
	*scl.Unit
}

var B = bot.B

func (u *Unit) Maneuver() bool {
	if !u.IsCool() {
		attackers := u.InRangeOf(B.Enemies.AllReady, 2)
		closeTargets := u.CanAttack(Targets.Armed, -0.5)
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
	for _, f := range []func() bool{u.Maneuver, u.Cast, u.Attack, u.Explore} {
		if f() {
			continue
		}
	}
}
