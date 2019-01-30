package micro

import (
	"bitbucket.org/aisee/sc2lib"
	"bitbucket.org/aisee/veterran/bot"
	"github.com/chippydip/go-sc2ai/enums/ability"
)

type Medivac struct {
	*Unit

	Injured      scl.Units
	FirstPatient *scl.Unit
}

func NewMedivac(u *scl.Unit) *Medivac {
	return &Medivac{Unit: NewUnit(u)}
}

func MedivacsLogic(us scl.Units) {
	if us.Empty() {
		return
	}

	patients := append(B.Groups.Get(bot.Marines).Units, B.Groups.Get(bot.Marauders).Units...)
	if patients.Empty() {
		patients = B.Groups.Get(bot.Miners).Units
	}
	if patients.Empty() {
		return
	}

	injured := patients.Filter(func(unit *scl.Unit) bool { return unit.Hits < unit.HitsMax })
	injured.OrderBy(func(unit *scl.Unit) float64 { return unit.Hits / unit.HitsMax }, false)

	enemiesCenter := B.Enemies.AllReady.Center()
	firstPatient := patients.ClosestTo(enemiesCenter)

	for _, u := range us {
		m := NewMedivac(u)
		m.Injured = injured
		m.FirstPatient = firstPatient
		m.Logic()
	}
}

func (u *Medivac) Maneuver() bool {
	// This should be most damaged unit
	closeInjured := u.Injured.CloserThan(float64(u.Radius)+4, u).First()
	if closeInjured == nil {
		closeInjured = u.Injured.ClosestTo(u)
	}
	if u.Energy >= 5 && u.HasAbility(ability.Effect_Heal) && closeInjured != nil {
		if closeInjured.IsFurtherThan(8, u) &&
			u.HasAbility(ability.Effect_MedivacIgniteAfterburners) {
			u.Command(ability.Effect_MedivacIgniteAfterburners)
			u.CommandTagQueue(ability.Effect_Heal, closeInjured.Tag)
		} else {
			u.CommandTag(ability.Effect_Heal, closeInjured.Tag)
		}
		return true
	}
	if closeInjured == nil {
		closeInjured = u.FirstPatient
	}

	pos, safe := u.AirEvade(B.Enemies.AllReady.CanAttack(u.Unit.Unit, 2), 2, closeInjured)
	if !safe {
		u.CommandPos(ability.Move, pos)
	} else {
		u.CommandTag(ability.Move, closeInjured.Tag)
	}
	return true
}

func (u *Medivac) Attack() bool {
	return false
}
