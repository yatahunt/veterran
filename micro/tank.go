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
		inSight := Targets.Ground.InRangeOf(u, 4-0.5)   // 7+4=11 - sight range

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

func CalcDamageScore(target *scl.Unit, mul float64) float64 {
	var dmg float64
	if target.IsArmored() {
		dmg = 70 * mul
	} else {
		dmg = 40 * mul
	}
	// I've tried to add some points here if target dies but it seems that its not worth it
	// Benchmark: 4004 - old algo, 4459 - current, 3996 - if I add score for kills
	// But then I made another test:
	// 4604 - no score for kill, 4390 - +1 for kill, 4090 - count only real HP, 4073 - double score for kill
	if target.Hits <= dmg {
		// But I think that my benchmark could be bad for measuring all the changes. For instance, how this could be
		// bad? I add score for destroyed units. Ex: 2 same units could be hit by splash. But one has more hits, so
		// direct hit won't kill it but it will kill other unit. In case of +1 later variant will be chosen
		dmg += 1
		// dmg = target.Hits
		// dmg *= 2
	}
	return dmg
}

func GetDamageScore(target *scl.Unit, targets scl.Units) float64 {
	var dmg float64
	for _, subTarget := range targets {
		if target.Tag == subTarget.Tag {
			continue
		}
		if subTarget.IsFurtherThan(1.25+float64(subTarget.Radius), target) {
			continue
		}
		mul := 0.25
		if subTarget.IsCloserThan(0.7812+float64(subTarget.Radius), target) {
			mul = 0.5
			if subTarget.IsCloserThan(0.4687+float64(subTarget.Radius), target) {
				mul = 1
			}
		}
		dmg += CalcDamageScore(subTarget, mul)
	}
	return dmg
}

func FindBestTarget(u *scl.Unit, targets, friends scl.Units) *scl.Unit {
	targets = targets.Filter(func(unit *scl.Unit) bool {
		if !unit.IsVisible() {
			return false
		}
		dist := u.Dist(unit) - float64(u.Radius+unit.Radius)
		return dist >= 2 && dist <= 13
	})
	if targets.Empty() {
		return nil
	}
	friends = friends.CloserThan(16, u)

	var maxDmg float64
	var bestTarget *scl.Unit
	for _, target := range targets {
		dmg := CalcDamageScore(target, 1)
		dmg += GetDamageScore(target, targets)
		dmg -= GetDamageScore(target, friends)

		if dmg > maxDmg {
			bestTarget = target
			maxDmg = dmg
		}
	}
	return bestTarget
}

func TankAttack(u *scl.Unit) bool {
	if Targets.Ground.Exists() {
		if u.UnitType == terran.SiegeTank {
			u.Attack(Targets.ArmedGroundArmored, Targets.ArmedGround, Targets.Ground)
		} else if u.UnitType == terran.SiegeTankSieged {
			if u.WeaponCooldown < 6 && u.IsAlreadyAttackingTargetInRange() {
				// Don't switch targets, it's time to shoot soon
				return true
			}

			target := FindBestTarget(u, Targets.ArmedGroundNotBuildings, Targets.MyGround)
			if target == nil {
				target = FindBestTarget(u, Targets.ArmedGround, Targets.MyGround)
			}
			if target == nil {
				target = FindBestTarget(u, Targets.Ground, Targets.MyGround)
			}
			if target != nil {
				u.CommandTag(ability.Attack_Attack, target.Tag)
				B.U.LastAttack[u.Tag] = B.Loop
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
