package bot

import (
	"bitbucket.org/aisee/sc2lib"
	"bitbucket.org/aisee/veterran/micro"
	"github.com/chippydip/go-sc2ai/api"
)

// Pass assignGroup = 0 to skip group assignement
func GetSCV(ptr scl.Pointer, assignGroup scl.GroupID, minHits float64) *scl.Unit {
	// refs := B.Units.My.OfType(terran.Refinery, terran.RefineryRich)
	scv1 := B.Groups.Get(micro.ScvReserve).Units.ClosestTo(ptr)
	scv2 := B.Groups.Get(micro.Miners).Units.Filter(func(unit *scl.Unit) bool {
		return !unit.IsReturning() && unit.Hits >= minHits // && refs.ByTag(unit.TargetTag()) == nil
	}).ClosestTo(ptr)

	scv := scv1
	if scv1 != nil && scv2 != nil {
		scv = scl.Units{scv1, scv2}.ClosestTo(ptr)
	} else if scv == nil {
		scv = scv2
	}

	if scv != nil && assignGroup != 0 {
		B.Groups.Add(assignGroup, scv)
	}
	return scv
}

func AlreadyTraining(abilityID api.AbilityID) int {
	count := 0
	units := B.Units.My.All()
	for _, unit := range units {
		if unit.IsStructure() && unit.TargetAbility() == abilityID {
			count++
		}
	}
	return count
}
