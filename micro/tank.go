package micro

import (
	"bitbucket.org/aisee/sc2lib"
	"github.com/chippydip/go-sc2ai/enums/ability"
	"github.com/chippydip/go-sc2ai/enums/terran"
)

type Tank struct {
	*Unit
}

func NewTank(u *scl.Unit) *Tank {
	return &Tank{Unit: NewUnit(u)}
}

func TanksLogic(us scl.Units) {
	for _, u := range us {
		NewTank(u).Logic()
	}
}

func (u *Tank) Retreat() bool {
	if u.UnitType == terran.SiegeTank && u.Hits < u.HitsMax/2 {
		B.Groups.Add(MechRetreat, u.Unit.Unit)
		return true
	}
	return false
}

func (u *Tank) Maneuver() bool {
	if u.UnitType == terran.SiegeTank && !u.IsCool() {
		attackers := B.Enemies.AllReady.CanAttack(u.Unit.Unit, 2)
		closeTargets := Targets.ArmedGround.InRangeOf(u.Unit.Unit, -0.5)
		if attackers.Exists() || closeTargets.Exists() {
			u.GroundFallback(attackers, 2, B.HomePaths)
			return true
		}
	}
	return false
}

func (u *Tank) Cast() bool {
	if u.UnitType == terran.SiegeTank {
		targets := Targets.ArmedGround.InRangeOf(u.Unit.Unit, 0)
		if targets.Empty() {
			targets = Targets.Ground.InRangeOf(u.Unit.Unit, 0)
		}
		farTargets := Targets.ArmedGround.InRangeOf(u.Unit.Unit, 13-7) // Sieged range - mobile range
		if farTargets.Empty() {
			farTargets = Targets.Ground.InRangeOf(u.Unit.Unit, 13-7)
		}

		if targets.Empty() && farTargets.Exists() && B.Enemies.AllReady.CanAttack(u.Unit.Unit, 2).Exists() {
			u.Command(ability.Morph_SiegeMode)
			return true
		}
	}
	if u.UnitType == terran.SiegeTankSieged {
		farTargets := Targets.ArmedGround.InRangeOf(u.Unit.Unit, 2).Filter(func(unit *scl.Unit) bool {
			return unit.IsFurtherThan(float64(u.Radius+unit.Radius+2), u)
		})
		targets := farTargets.InRangeOf(u.Unit.Unit, 0)
		if targets.Empty() {
			targets = Targets.Ground.InRangeOf(u.Unit.Unit, 0)
		}
		// Unsiege if can't attack and only buildings are close to max range
		if targets.Empty() && farTargets.Filter(func(unit *scl.Unit) bool { return !unit.IsStructure() }).Empty() {
			u.Command(ability.Morph_Unsiege)
			return true
		}
	}
	return false
}

func (u *Tank) Attack() bool {
	if Targets.Ground.Exists() {
		if u.UnitType == terran.SiegeTank {
			u.AttackCustom(scl.DefaultAttackFunc, scl.DefaultMoveFunc, Targets.ArmedGround, Targets.Ground)
		} else if u.UnitType == terran.SiegeTankSieged {
			targets := Targets.ArmedGround.InRangeOf(u.Unit.Unit, 0)
			if targets.Empty() {
				targets = Targets.Ground.InRangeOf(u.Unit.Unit, 0)
			}
			if targets.Exists() {
				u.AttackCustom(scl.DefaultAttackFunc, scl.DefaultMoveFunc, targets)
			}
		}
		return true
	}
	return false
}
