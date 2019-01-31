package micro

import (
	"bitbucket.org/aisee/sc2lib"
	"github.com/chippydip/go-sc2ai/enums/ability"
	"github.com/chippydip/go-sc2ai/enums/buff"
)

func MarauderStim(u *scl.Unit) bool {
	if B.Upgrades[ability.Research_Stimpack] && u.HasAbility(ability.Effect_Stim_Marauder) &&
		!u.HasBuff(buff.StimpackMarauder) && u.CanAttack(Targets.ArmedGround, 2).Sum(scl.CmpHits) >= 200 {
		u.Command(ability.Effect_Stim_Marauder)
		return true
	}
	return false
}

func MarauderAttack(u *scl.Unit) bool {
	if Targets.Ground.Exists() {
		u.Attack(Targets.ArmedGroundArmored, Targets.ArmedGround, Targets.Ground)
		return true
	}
	return false
}

func MaraudersLogic(us scl.Units) {
	for _, u := range us {
		_ = DefaultManeuver(u) || MarauderStim(u) || MarauderAttack(u) || DefaultExplore(u)
	}
}
