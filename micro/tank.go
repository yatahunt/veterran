package micro

import (
	"bitbucket.org/aisee/sc2lib"
	"bitbucket.org/aisee/veterran/bot"
	"github.com/chippydip/go-sc2ai/enums/ability"
	"github.com/chippydip/go-sc2ai/enums/terran"
)

func TankRetreat(u *scl.Unit) bool {
	if u.UnitType == terran.SiegeTank && u.Hits < u.HitsMax/2 {
		B.Groups.Add(bot.MechRetreat, u)
		return true
	}
	return false
}

func TankManeuver(u *scl.Unit) bool {
	if u.UnitType == terran.SiegeTank && !u.IsHalfCool() {
		closeTargets := Targets.ArmedGround.InRangeOf(u, -0.5)
		if closeTargets.Exists() {
			u.GroundFallback(B.Enemies.AllReady, 2, B.HomePaths)
			return true
		}
	}
	return false
}

func TankMorph(u *scl.Unit) bool {
	if u.UnitType == terran.SiegeTank {
		targets := Targets.ArmedGround.InRangeOf(u, 0)
		if targets.Empty() {
			targets = Targets.Ground.InRangeOf(u, 0)
		}
		farTargets := Targets.ArmedGround.InRangeOf(u, 13-7) // Sieged range - mobile range
		if farTargets.Empty() {
			farTargets = Targets.Ground.InRangeOf(u, 13-7)
		}

		if targets.Empty() && farTargets.Exists() && B.Enemies.AllReady.CanAttack(u, 2).Exists() {
			u.Command(ability.Morph_SiegeMode)
			return true
		}
	}
	if u.UnitType == terran.SiegeTankSieged {
		farTargets := Targets.ArmedGround.InRangeOf(u, 2).Filter(func(unit *scl.Unit) bool {
			return unit.IsFurtherThan(float64(u.Radius+unit.Radius+2), u)
		})
		targets := farTargets.InRangeOf(u, 0)
		if targets.Empty() {
			targets = Targets.Ground.InRangeOf(u, 0)
		}
		// Unsiege if can't attack and only buildings are close to max range
		if targets.Empty() && farTargets.Filter(func(unit *scl.Unit) bool { return !unit.IsStructure() }).Empty() {
			u.Command(ability.Morph_Unsiege)
			return true
		}
	}
	return false
}

func TankAttack(u *scl.Unit) bool {
	if Targets.Ground.Exists() {
		if u.UnitType == terran.SiegeTank {
			u.Attack(Targets.ArmedGroundArmored, Targets.ArmedGround, Targets.Ground)
		} else if u.UnitType == terran.SiegeTankSieged {
			targets := Targets.ArmedGroundArmored.InRangeOf(u, 0)
			if targets.Empty() {
				targets = Targets.ArmedGround.InRangeOf(u, 0)
			}
			if targets.Empty() {
				targets = Targets.Ground.InRangeOf(u, 0)
			}
			if targets.Exists() {
				u.Attack(targets)
			}
		}
		return true
	}
	return false
}

func TanksLogic(us scl.Units) {
	for _, u := range us {
		_ = TankRetreat(u) || TankManeuver(u) || TankMorph(u) || TankAttack(u) || DefaultExplore(u)
	}
}
