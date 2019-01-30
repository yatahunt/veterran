package micro

import (
	"bitbucket.org/aisee/sc2lib"
	"bitbucket.org/aisee/veterran/bot"
	"github.com/chippydip/go-sc2ai/enums/ability"
)

type Cyclone struct {
	*Unit
}

func NewCyclone(u *scl.Unit) *Cyclone {
	return &Cyclone{Unit: NewUnit(u)}
}

func CyclonesLogic(us scl.Units) {
	for _, u := range us {
		NewCyclone(u).Logic()
	}
}

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
		u.AttackMove(target, bot.B.HomePaths)
	}
}

func (u *Cyclone) Maneuver() bool {
	attackers := B.Enemies.AllReady.CanAttack(u.Unit.Unit, 2)
	canLock := u.HasAbility(ability.Effect_LockOn)
	/*target := allEnemies.ByTag(cyclone.EngagedTargetTag)
	isLocked := !canLock && target != nil
	canAttack := !isLocked && cyclone.IsCool()*/
	if !canLock {
		target := Targets.Armed.ClosestTo(u)
		if target != nil && u.InRange(target, 4) {
			u.GroundFallback(attackers, 2, B.HomePaths)
			return true
		}
	} else if !u.IsCool() {
		closeTargets := Targets.Armed.InRangeOf(u.Unit.Unit, -0.5)
		if attackers.Exists() || closeTargets.Exists() {
			u.GroundFallback(attackers, 2, B.HomePaths)
			return true
		}
	}
	return false
}

func (u *Cyclone) Attack() bool {
	if Targets.All.Exists() {
		u.AttackCustom(CycloneAttackFunc, CycloneMoveFunc, Targets.ArmedFlying, Targets.Armed, Targets.All)
		return true
	}
	return false
}
