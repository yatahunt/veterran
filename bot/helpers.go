package bot

import (
	"github.com/aiseeq/s2l/lib/point"
	"github.com/aiseeq/s2l/lib/scl"
	"github.com/aiseeq/s2l/protocol/api"
)

// Pass assignGroup = 0 to skip group assignement
func GetSCV(ptr point.Pointer, assignGroup scl.GroupID, minHits float64) *scl.Unit {
	scv1 := B.Groups.Get(ScvReserve).Units.ClosestTo(ptr)
	scv2 := B.Groups.Get(Miners).Units.Filter(func(unit *scl.Unit) bool {
		// Not carrying anything and not assigned to mine gas
		return unit.Hits >= minHits && len(unit.BuffIds) == 0 && B.Miners.GasForMiner[unit.Tag] == 0
	}).ClosestTo(ptr)
	if scv2 == nil {
		scv2 = B.Groups.Get(Miners).Units.Filter(func(unit *scl.Unit) bool {
			// If only gas miners left, then ok
			return unit.Hits >= minHits && len(unit.BuffIds) == 0
		}).ClosestTo(ptr)
	}

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
