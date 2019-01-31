package micro

import (
	"bitbucket.org/aisee/sc2lib"
	"github.com/chippydip/go-sc2ai/api"
	"github.com/chippydip/go-sc2ai/enums/ability"
	"github.com/chippydip/go-sc2ai/enums/terran"
)

// todo: take old Maneuver() from repo if standard method isn't cool
func HellionMorph(u *scl.Unit) bool {
	// Transform into hellbats vs zerg in defense, armory exists, not on main base
	if B.EnemyRace == api.Race_Zerg && u.UnitType == terran.Hellion /*&& PlayDefensive*/ &&
		B.Units.My[terran.Armory].First(scl.Ready) != nil && B.HeightAt(u) != B.HeightAt(B.Locs.MyStart) {
		u.Command(ability.Morph_Hellbat)
		return true
	}
	return false
}

func HellionsLogic(us scl.Units) {
	for _, u := range us {
		_ = DefaultRetreat(u) || DefaultManeuver(u) || HellionMorph(u) || DefaultGroundAttack(u) || DefaultExplore(u)
	}
}
