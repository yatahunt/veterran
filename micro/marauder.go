package micro

import (
	"bitbucket.org/aisee/sc2lib"
	"github.com/chippydip/go-sc2ai/enums/ability"
	"github.com/chippydip/go-sc2ai/enums/buff"
)

type Marauder struct {
	*Unit
}

func NewMarauder(u *scl.Unit) *Marauder {
	return &Marauder{Unit: NewUnit(u)}
}

func MaraudersLogic(us scl.Units) {
	for _, u := range us {
		m := NewMarauder(u)
		m.Logic(m)
	}
}

func (u *Marauder) Retreat() bool {
	return false
}

func (u *Marauder) Maneuver() bool {
	if !u.IsCool() {
		attackers := B.Enemies.AllReady.CanAttack(u.Unit.Unit, 2)
		closeTargets := Targets.ArmedGround.InRangeOf(u.Unit.Unit, -0.5)
		if attackers.Exists() || closeTargets.Exists() {
			u.GroundFallback(attackers, 2, B.HomePaths)
			return true
		}
	}
	return false
}

func (u *Marauder) Cast() bool {
	if B.Upgrades[ability.Research_Stimpack] && u.HasAbility(ability.Effect_Stim_Marauder) &&
		!u.HasBuff(buff.StimpackMarauder) && u.CanAttack(Targets.ArmedGround, 2).Sum(scl.CmpHits) >= 200 {
		u.Command(ability.Effect_Stim_Marauder)
		return true
	}
	return false
}

func (u *Marauder) Attack() bool {
	if Targets.ArmedGround.Exists() || Targets.Ground.Exists() {
		u.AttackCustom(scl.DefaultAttackFunc, scl.DefaultMoveFunc, Targets.ArmedGround, Targets.Ground)
		return true
	}
	return false
}
