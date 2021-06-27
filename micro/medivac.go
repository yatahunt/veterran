package micro

import (
	"bitbucket.org/aisee/veterran/bot"
	"github.com/aiseeq/s2l/lib/scl"
	"github.com/aiseeq/s2l/protocol/enums/ability"
)

func MedivacsLogic(us scl.Units) {
	if us.Empty() {
		return
	}

	patients := append(B.Groups.Get(bot.Marines).Units, B.Groups.Get(bot.Marauders).Units...)
	if patients.Empty() {
		patients = B.Groups.Get(bot.Thors).Units
		if patients.Empty() {
			patients = B.Groups.Get(bot.Miners).Units
		}
	}
	if patients.Empty() {
		return
	}

	injured := patients.Filter(func(unit *scl.Unit) bool { return unit.Hits < unit.HitsMax })
	injured.OrderBy(func(unit *scl.Unit) float64 { return unit.Hits / unit.HitsMax }, false)

	enemiesCenter := B.Enemies.AllReady.Center()
	if enemiesCenter == 0 {
		enemiesCenter = B.Locs.EnemyStart
	}
	firstPatient := patients.ClosestTo(enemiesCenter)

	for _, u := range us {
		if DefaultRetreat(u) || u.EvadeEffects() {
			continue
		}
		// quickfix: I don't know how medivac can grab thor while he is not in ThorEvacs group
		if len(u.Passengers) > 0 {
			u.CommandPos(ability.UnloadAllAt_Medivac, u)
			continue
		}

		enemies := B.Enemies.AllReady.CanAttack(u, 2)
		pos, safe := u.AirEvade(enemies, 2, u)
		if !safe {
			outranged, stronger := u.AssessStrength(enemies)
			if !outranged || stronger {
				safe = true
			} else {
				u.CommandPos(ability.Move, pos)
				continue
			}
		}

		// This should be most damaged unit
		closeInjured := injured.CloserThan(float64(u.Radius)+4, u).First()
		if closeInjured == nil {
			closeInjured = injured.ClosestTo(u)
		}
		if u.Energy >= 5 && u.HasAbility(ability.Effect_Heal) && closeInjured != nil {
			if closeInjured.IsFurtherThan(8, u) &&
				u.HasAbility(ability.Effect_MedivacIgniteAfterburners) {
				u.Command(ability.Effect_MedivacIgniteAfterburners)
				// u.CommandTagQueue(ability.Effect_Heal, closeInjured.Tag)
			} else {
				u.CommandTag(ability.Effect_Heal, closeInjured.Tag)
			}
			continue
		}
		if closeInjured == nil {
			closeInjured = firstPatient
		}

		pos, safe = u.AirEvade(B.Enemies.AllReady.CanAttack(u, 2), 2, closeInjured)
		if !safe {
			u.CommandPos(ability.Move, pos)
		} else {
			u.CommandTag(ability.Move, closeInjured.Tag)
		}
	}
}
