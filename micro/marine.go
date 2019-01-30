package micro

import (
	"bitbucket.org/aisee/sc2lib"
	"bitbucket.org/aisee/veterran/bot"
	"github.com/chippydip/go-sc2ai/enums/ability"
	"github.com/chippydip/go-sc2ai/enums/buff"
	"github.com/chippydip/go-sc2ai/enums/protoss"
)

type Marine struct {
	*Unit
}

func NewMarine(u *scl.Unit) *Marine {
	return &Marine{Unit: NewUnit(u)}
}

func MarinesLogic(us scl.Units) {
	for _, u := range us {
		NewMarine(u).Logic()
	}
}

func (u *Marine) Retreat() bool {
	return false
}

func (u *Marine) Maneuver() bool {
	if u.Unit.Maneuver() {
		return true
	}

	// Load into a bunker
	if B.PlayDefensive && u.CanAttack(Targets.Armed, 0).Empty() {
		bunker := bot.GetEmptyBunker(u)
		if bunker != nil {
			if bunker.IsReady() {
				u.CommandTag(ability.Smart, bunker.Tag)
			} else if u.IsFarFrom(bunker) {
				u.CommandPos(ability.Move, bunker)
			}
			return true
		}
	}

	return false
}

func (u *Marine) Cast() bool {
	if B.Upgrades[ability.Research_Stimpack] && u.HasAbility(ability.Effect_Stim_Marine) &&
		!u.HasBuff(buff.Stimpack) && u.CanAttack(Targets.Armed, 2).Sum(scl.CmpHits) >= 200 {
		u.Command(ability.Effect_Stim_Marine)
		return true
	}
	return false
}

func (u *Marine) Attack() bool {
	if Targets.Armed.Exists() || Targets.All.Exists() {
		ics := B.Units.Enemy[protoss.Interceptor]
		if ics.Exists() {
			u.CommandPos(ability.Attack_Attack_23, ics.ClosestTo(u))
		} else {
			u.AttackCustom(scl.DefaultAttackFunc, scl.DefaultMoveFunc, Targets.Armed, Targets.All)
		}
		return true
	}
	return false
}
