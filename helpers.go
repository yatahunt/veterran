package main

import (
	"bitbucket.org/aisee/sc2lib"
	"github.com/chippydip/go-sc2ai/api"
)

// Pass assignGroup = 0 to skip group assignement
func (b *bot) GetSCV(ptr scl.Pointer, assignGroup scl.GroupID, minHits float64) *scl.Unit {
	// refs := b.Units.My.OfType(terran.Refinery, terran.RefineryRich)
	scv1 := b.Groups.Get(ScvReserve).Units.ClosestTo(ptr)
	scv2 := b.Groups.Get(Miners).Units.Filter(func(unit *scl.Unit) bool {
		return !unit.IsReturning() && unit.Hits >= minHits // && refs.ByTag(unit.TargetTag()) == nil
	}).ClosestTo(ptr)

	scv := scv1
	if scv1 != nil && scv2 != nil {
		scv = scl.Units{scv1, scv2}.ClosestTo(ptr)
	} else if scv == nil {
		scv = scv2
	}

	if scv != nil && assignGroup != 0 {
		b.Groups.Add(assignGroup, scv)
	}
	return scv
}

func (b *bot) AlreadyTraining(abilityID api.AbilityID) int {
	count := 0
	units := b.Units.My.All()
	for _, unit := range units {
		if unit.IsStructure() && unit.TargetAbility() == abilityID {
			count++
		}
	}
	return count
}
