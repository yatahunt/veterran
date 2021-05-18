package micro

import (
	"github.com/aiseeq/s2l/lib/scl"
	"github.com/aiseeq/s2l/protocol/api"
	"github.com/aiseeq/s2l/protocol/enums/ability"
	"github.com/aiseeq/s2l/protocol/enums/terran"
)

// todo: take old Maneuver() from repo if standard method isn't cool
func HellionMorph(u *scl.Unit) bool {
	// Transform into hellbats vs zerg in defense, armory exists, not on main base
	if B.EnemyRace == api.Race_Zerg && u.UnitType == terran.Hellion /*&& PlayDefensive*/ &&
		B.Units.My[terran.Armory].First(scl.Ready) != nil && B.Grid.HeightAt(u) != B.Grid.HeightAt(B.Locs.MyStart) {
		u.Command(ability.Morph_Hellbat)
		return true
	}
	return false
}

func HellionAttack(u *scl.Unit) bool {
	if Targets.Ground.Exists() {
		u.Attack(Targets.ArmedGroundLight, Targets.ArmedGround, Targets.Ground)
		return true
	}
	return false
}

func HellionsLogic(us scl.Units) {
	for _, u := range us {
		_ = DefaultRetreat(u) || DefaultManeuver(u) || HellionMorph(u) || HellionAttack(u) || DefaultExplore(u)
	}
}
