package micro

import (
	"bitbucket.org/aisee/sc2lib"
	"github.com/chippydip/go-sc2ai/api"
	"github.com/chippydip/go-sc2ai/enums/ability"
	"github.com/chippydip/go-sc2ai/enums/terran"
)

type Hellion struct {
	*Unit
}

func NewHellion(u *scl.Unit) *Hellion {
	return &Hellion{Unit: NewUnit(u)}
}

func HellionsLogic(us scl.Units) {
	for _, u := range us {
		h := NewHellion(u)
		h.Logic(h)
	}
}

// todo: take old Maneuver() from repo if standard method isn't cool
func (u *Hellion) Maneuver() bool {
	if !u.IsCool() {
		attackers := B.Enemies.AllReady.CanAttack(u.Unit.Unit, 2)
		closeTargets := Targets.ArmedGround.InRangeOf(u.Unit.Unit, -0.5)
		if attackers.Exists() || closeTargets.Exists() {
			u.GroundFallback(attackers, 2, B.HomePaths)
			return true
		}
	}
	return false
}

func (u *Hellion) Cast() bool {
	// Transform into hellbats vs zerg in defense, armory exists, not on main base
	if B.EnemyRace == api.Race_Zerg && u.UnitType == terran.Hellion /*&& PlayDefensive*/ &&
		B.Units.My[terran.Armory].First(scl.Ready) != nil && B.HeightAt(u) != B.HeightAt(B.Locs.MyStart) {
		u.Command(ability.Morph_Hellbat)
		return true
	}
	return false
}

func (u *Hellion) Attack() bool {
	if Targets.ArmedGround.Exists() || Targets.Ground.Exists() {
		u.AttackCustom(scl.DefaultAttackFunc, scl.DefaultMoveFunc, Targets.ArmedGround, Targets.Ground)
		return true
	}
	return false
}
