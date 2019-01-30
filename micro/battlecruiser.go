package micro

import (
	"bitbucket.org/aisee/sc2lib"
	"github.com/chippydip/go-sc2ai/api"
	"github.com/chippydip/go-sc2ai/enums/ability"
)

type Battlecruiser struct {
	*Unit

	YamatoTargets map[api.UnitTag]int
}

func NewBattlecruiser(u *scl.Unit) *Battlecruiser {
	return &Battlecruiser{Unit: NewUnit(u)}
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
		bc := NewBattlecruiser(u)
		bc.YamatoTargets = yamatoTargets
		bc.Logic()
	}
}

func (u *Battlecruiser) Retreat() bool {
	if (u.HasAbility(ability.Effect_TacticalJump) && u.Hits < 100) ||
		(!u.HasAbility(ability.Effect_TacticalJump) && u.Hits < u.HitsMax/2) {
		B.Groups.Add(MechRetreat, u.Unit.Unit)
		return true
	}
	return false
}

func (u *Battlecruiser) Maneuver() bool {
	return false
}

func (u *Battlecruiser) Cast() bool {
	if Targets.ForYamato.Exists() && u.HasAbility(ability.Effect_YamatoGun) {
		targets := Targets.ForYamato.InRangeOf(u.Unit.Unit, 4).Filter(func(unit *scl.Unit) bool {
			return unit.Hits-float64(u.YamatoTargets[unit.Tag]*240) > 0
		})
		if targets.Exists() {
			target := targets.Filter(func(unit *scl.Unit) bool {
				return unit.Hits-float64(u.YamatoTargets[unit.Tag]*240) <= 240
			}).Max(scl.CmpHits)
			if target == nil {
				target = targets.Max(scl.CmpHits)
			}
			u.CommandTag(ability.Effect_YamatoGun, target.Tag)
			u.YamatoTargets[target.Tag]++
			return true
		}
	}
	return false
}
