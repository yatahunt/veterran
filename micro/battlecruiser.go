package micro

import (
	"bitbucket.org/aisee/veterran/bot"
	"github.com/aiseeq/s2l/lib/scl"
	"github.com/aiseeq/s2l/protocol/api"
	"github.com/aiseeq/s2l/protocol/enums/ability"
	"github.com/aiseeq/s2l/protocol/enums/buff"
)

func BattlecruiserRetreat(u *scl.Unit) bool {
	if (u.HasAbility(ability.Effect_TacticalJump) && u.Hits < 150) ||
		(!u.HasAbility(ability.Effect_TacticalJump) && u.Hits < u.HitsMax/2) ||
		u.HasBuff(buff.RavenScramblerMissile) {
		B.Groups.Add(bot.MechRetreat, u)
		return true
	}
	return false
}

func BattlecruiserCast(u *scl.Unit, yamatoTargets map[api.UnitTag]int) bool {
	if Targets.ForYamato.Exists() && u.HasAbility(ability.Effect_YamatoGun) {
		targets := Targets.ForYamato.InRangeOf(u, 4).Filter(func(unit *scl.Unit) bool {
			return unit.Hits-float64(yamatoTargets[unit.Tag]*240) > 0
		})
		if targets.Exists() {
			target := targets.Filter(func(unit *scl.Unit) bool {
				return unit.Hits-float64(yamatoTargets[unit.Tag]*240) <= 240
			}).Max(scl.CmpHits)
			if target == nil {
				target = targets.Max(scl.CmpHits)
			}
			u.CommandTag(ability.Effect_YamatoGun, target.Tag)
			yamatoTargets[target.Tag]++
			return true
		}
	}
	return false
}

func BattlecruisersAttack(u *scl.Unit) bool {
	if Targets.All.Exists() {
		u.Attack(Targets.ForYamato, Targets.AntiAir, Targets.Armed, Targets.All)
		return true
	}
	return false
}

func BattlecruisersLogic(us scl.Units) {
	if us.Empty() {
		return
	}

	yamatoTargets := map[api.UnitTag]int{}
	for _, u := range us {
		if u.TargetAbility() == ability.Effect_YamatoGun {
			yamatoTargets[u.TargetTag()]++
		}
	}

	for _, u := range us {
		_ = BattlecruiserRetreat(u) || u.EvadeEffects() || BattlecruiserCast(u, yamatoTargets) ||
			BattlecruisersAttack(u) || DefaultExplore(u)
	}
}
