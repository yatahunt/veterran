package micro

import (
	"bitbucket.org/aisee/sc2lib/scl"
	"github.com/chippydip/go-sc2ai/enums/ability"
)

func CycloneAttackFunc(u *scl.Unit, priority int, targets scl.Units) bool {
	hasLockOn := u.HasAbility(ability.Effect_LockOn)
	visibleTargets := targets.Filter(scl.Visible)
	if hasLockOn {
		closeTargets := visibleTargets.InRangeOf(u, 2) // Range = 7. Weapons + 2
		if closeTargets.Exists() {
			target := closeTargets.Max(func(unit *scl.Unit) float64 {
				if unit.IsArmored() {
					return unit.Hits * 2
				}
				return unit.Hits
			})
			u.CommandTag(ability.Effect_LockOn, target.Tag)
			return true
		}
		return false
	}
	closeTargets := visibleTargets.InRangeOf(u, 0)
	if closeTargets.Exists() {
		target := closeTargets.Min(func(unit *scl.Unit) float64 {
			return unit.Hits
		})
		u.CommandTag(ability.Attack_Attack_23, target.Tag)
		return true
	}
	return false
}

func CycloneMoveFunc(u *scl.Unit, target *scl.Unit) {
	// Unit need to be closer to the target to shoot? (lock-on range)
	if !u.InRange(target, 2-0.1) || !target.IsVisible() {
		u.AttackMove(target)
	}
}

func CycloneManeuver(u *scl.Unit) bool {
	attackers := B.Enemies.AllReady.CanAttack(u, 2)
	canLock := u.HasAbility(ability.Effect_LockOn)
	if !canLock {
		target := Targets.Armed.ClosestTo(u)
		if target != nil && u.InRange(target, 4) {
			u.GroundFallback(attackers, 2, B.Locs.MyStart-B.Locs.MyStartMinVec*3)
			return true
		}
	} else if !u.IsHalfCool() {
		closeTargets := Targets.Armed.InRangeOf(u, -0.5)
		if attackers.Exists() || closeTargets.Exists() {
			u.GroundFallback(attackers, 2, B.Locs.MyStart-B.Locs.MyStartMinVec*3)
			return true
		}
	}
	return false
}

func CycloneAttack(u *scl.Unit) bool {
	if Targets.All.Exists() {
		u.AttackCustom(CycloneAttackFunc, CycloneMoveFunc, Targets.ArmedFlyingArmored, Targets.ArmedFlying,
			Targets.ArmedArmored, Targets.Armed, Targets.All)
		return true
	}
	return false
}

func CyclonesLogic(us scl.Units) {
	for _, u := range us {
		_ = DefaultRetreat(u) || CycloneManeuver(u) || CycloneAttack(u) || DefaultExplore(u)
	}
}
