package micro

import (
	"github.com/aiseeq/s2l/lib/scl"
	"github.com/aiseeq/s2l/protocol/enums/ability"
)

func ThorMorph(u *scl.Unit) bool {
	if u.HasTrueAbility(ability.Morph_ThorHighImpactMode) {
		u.Command(ability.Morph_ThorHighImpactMode)
		return true
	}
	return false
}

func ThorAttack(u *scl.Unit) bool {
	if Targets.All.Exists() {
		u.Attack(Targets.ArmedFlying, Targets.ArmedGround, Targets.Flying, Targets.Ground)
		return true
	}
	return false
}

func ThorsLogic(us scl.Units) {
	for _, u := range us {
		_ = DefaultRetreat(u) || DefaultManeuver(u) || ThorMorph(u) || ThorAttack(u) || DefaultExplore(u)
	}
}
