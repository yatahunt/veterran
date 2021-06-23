package micro

import (
	"bitbucket.org/aisee/veterran/bot"
	"github.com/aiseeq/s2l/lib/scl"
	"github.com/aiseeq/s2l/protocol/enums/ability"
)

func ThorRetreat(u *scl.Unit) bool {
	if DefaultRetreat(u) {
		if med := B.Groups.Get(bot.Medivacs).Units.ClosestTo(u); med != nil {
			B.Groups.Add(bot.ThorEvacs, med)
			if med.HasAbility(ability.Effect_MedivacIgniteAfterburners) {
				med.Command(ability.Effect_MedivacIgniteAfterburners)
			}
		}
		return true
	}
	return false
}

func ThorMorph(u *scl.Unit) bool {
	if u.HasAbility(ability.Morph_ThorHighImpactMode) {
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
		_ = ThorRetreat(u) || DefaultManeuver(u) || ThorMorph(u) || ThorAttack(u) || DefaultExplore(u)
	}
}
