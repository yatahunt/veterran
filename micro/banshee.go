package micro

import (
	"bitbucket.org/aisee/veterran/bot"
	"github.com/aiseeq/s2l/lib/scl"
	"github.com/aiseeq/s2l/protocol/enums/ability"
	"github.com/aiseeq/s2l/protocol/enums/buff"
)

func BansheeRetreat(u *scl.Unit) bool {
	if (u.Hits < u.HitsMax/2) || u.HasBuff(buff.RavenScramblerMissile) || u.HasBuff(buff.LockOn) {
		B.Groups.Add(bot.MechRetreat, u)
		return true
	}
	return false
}

func BansheesAttack(u *scl.Unit) bool {
	if Targets.All.Exists() {
		u.Attack(Targets.ArmedGroundNoAA, Targets.ArmedGround, Targets.Ground)
		return true
	}
	return false
}

func BansheesLogic(us scl.Units) {
	if us.Empty() {
		return
	}

	for _, u := range us {
		if u.HPS > 0 && u.HasAbility(ability.Behavior_CloakOn_Banshee) {
			u.Command(ability.Behavior_CloakOn_Banshee)
		}

		_ = BansheeRetreat(u) || DefaultManeuver(u) || BansheesAttack(u) || DefaultExplore(u)
	}
}
