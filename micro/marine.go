package micro

import (
	"bitbucket.org/aisee/veterran/bot"
	"github.com/aiseeq/s2l/lib/scl"
	"github.com/aiseeq/s2l/protocol/enums/ability"
	"github.com/aiseeq/s2l/protocol/enums/buff"
	"github.com/aiseeq/s2l/protocol/enums/protoss"
)

func LoadInBunker(u *scl.Unit) bool {
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

func MarineStim(u *scl.Unit) bool {
	if B.Upgrades[ability.Research_Stimpack] && u.HasAbility(ability.Effect_Stim_Marine) &&
		!u.HasBuff(buff.Stimpack) && u.CanAttack(Targets.Armed, 2).Sum(scl.CmpHits) >= 200 {
		u.Command(ability.Effect_Stim_Marine)
		return true
	}
	return false
}

func MarineAttack(u *scl.Unit) bool {
	if Targets.All.Exists() {
		ics := B.Units.Enemy[protoss.Interceptor]
		if ics.Exists() {
			u.CommandPos(ability.Attack_Attack, ics.ClosestTo(u))
			B.U.LastAttack[u.Tag] = B.Loop
		} else {
			u.Attack(Targets.Armed, Targets.All)
		}
		return true
	}
	return false
}

func MarinesLogic(us scl.Units) {
	for _, u := range us {
		// If something returns true - break chain
		_ = DefaultManeuver(u) || LoadInBunker(u) || MarineStim(u) || MarineAttack(u) || DefaultExplore(u)
	}
}
