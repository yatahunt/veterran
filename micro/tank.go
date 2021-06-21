package micro

import (
	"bitbucket.org/aisee/veterran/bot"
	"github.com/aiseeq/s2l/lib/scl"
	"github.com/aiseeq/s2l/protocol/enums/ability"
	"github.com/aiseeq/s2l/protocol/enums/buff"
	"github.com/aiseeq/s2l/protocol/enums/terran"
)

func TankRetreat(u *scl.Unit) bool {
	if u.UnitType == terran.SiegeTank && (u.Hits < u.HitsMax/2 || u.HasBuff(buff.RavenScramblerMissile)) {
		B.Groups.Add(bot.MechRetreat, u)
		return true
	}
	return false
}

func TankManeuver(u *scl.Unit) bool {
	if u.UnitType == terran.SiegeTank {
		return DefaultManeuver(u)
	}
	return false
}

func TankMorph(u *scl.Unit) bool {
	if u.UnitType == terran.SiegeTank {
		targets := Targets.Ground.InRangeOf(u, 0)
		farTargets := Targets.Ground.InRangeOf(u, 13-7) // Sieged range - mobile range
		inSight := Targets.Ground.InRangeOf(u, 4-0.5) // 7+4=11 - sight range

		// Tank can't attack anyone now and there is a far target that can hit tank if it closes or
		// there is a lot of far targets that worth morphing
		if targets.Empty() &&
			((farTargets.Exists() && B.Enemies.AllReady.CanAttack(u, 2).Exists()) ||
				(farTargets.Sum(scl.CmpHits) >= 210 && inSight.Exists())) {
			u.Command(ability.Morph_SiegeMode)
			return true
		}

		// Enter siege mode on defensive position
		if B.PlayDefensive {
			pos := GetDefensivePos(u)
			if !u.IsFarFrom(pos) && u.IsFarFrom(B.Ramps.My.Top-B.Ramps.My.Vec*3) {
				u.Command(ability.Morph_SiegeMode)
				return true
			}
		}
	}
	if u.UnitType == terran.SiegeTankSieged {
		// Keep siege mode on defensive position
		if B.PlayDefensive {
			pos := GetDefensivePos(u)
			if !u.IsFarFrom(pos) {
				return false
			}
		}
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
			targets := Targets.ArmedGroundArmored.Filter(scl.NotStructure).InRangeOf(u, 0)
			if targets.Empty() {
				targets = Targets.ArmedGround.Filter(scl.NotStructure).InRangeOf(u, 0)
			}
			if targets.Empty() {
				targets = Targets.Ground.Filter(scl.NotStructure).InRangeOf(u, 0)
			}
			if targets.Empty() {
				targets = Targets.ArmedGroundArmored.InRangeOf(u, 0)
			}
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
