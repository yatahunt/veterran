package bot

import (
	"github.com/aiseeq/s2l/lib/point"
	"github.com/aiseeq/s2l/lib/scl"
	"github.com/aiseeq/s2l/protocol/api"
)

// Pass assignGroup = 0 to skip group assignement
func GetSCV(ptr point.Pointer, assignGroup scl.GroupID, minHits float64) *scl.Unit {
	// refs := B.Units.My.OfType(terran.Refinery, terran.RefineryRich)
	scv1 := B.Groups.Get(ScvReserve).Units.ClosestTo(ptr)
	scv2 := B.Groups.Get(Miners).Units.Filter(func(unit *scl.Unit) bool {
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
