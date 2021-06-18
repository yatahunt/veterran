package micro

import (
	"github.com/aiseeq/s2l/lib/scl"
	"github.com/aiseeq/s2l/protocol/enums/ability"
)

func CycloneAttackFunc(u *scl.Unit, priority int, targets scl.Units) bool {
	visibleTargets := targets.Filter(scl.Visible)
	if u.HasAbility(ability.Effect_LockOn) {
		closeTargets := visibleTargets.InRangeOf(u, 2) // Range = 7. Weapons + 2
		if closeTargets.Exists() {
			target := closeTargets.Max(func(unit *scl.Unit) float64 {
				if unit.IsArmored() {
					return unit.Hits * 2
				}
				return unit.Hits
			})
			u.CommandTag(ability.Effect_LockOn, target.Tag)
			B.CycloneLocks[u.Tag] = target.Tag
			return true
		}
		return false
	}
	closeTargets := visibleTargets.InRangeOf(u, 0)
	if closeTargets.Exists() {
		target := closeTargets.Min(func(unit *scl.Unit) float64 {
			return unit.Hits
		})
		u.CommandTag(ability.Attack_Attack, target.Tag)
		B.U.LastAttack[u.Tag] = B.Loop
		return true
	}
	return false
}

func CycloneManeuver(u *scl.Unit) bool {
	lockedOn := u.HasAbility(ability.Cancel_LockOn)
	if lockedOn {
		target := B.Units.ByTag[B.CycloneLocks[u.Tag]]
		if target != nil {
			// vision - range = 6
			if !u.InRange(target, 6-1-float64(target.Radius)) {
				u.AttackMove(target)
				return true
			}
		}
	}
	if lockedOn || !u.IsCoolToAttack() {
		attackers := B.Enemies.AllReady.CanAttack(u, 4)
		if attackers.Exists() {
			u.GroundFallback(B.Locs.MyStart-B.Locs.MyStartMinVec*3)
			return true
		}
	}
	return u.EvadeEffects()
}

func CycloneAttack(u *scl.Unit) bool {
	if Targets.All.Exists() {
		u.AttackCustom(CycloneAttackFunc, scl.DefaultMoveFunc, Targets.ArmedFlyingArmored, Targets.ArmedFlying,
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
