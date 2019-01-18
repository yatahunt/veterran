package main

import (
	"bitbucket.org/aisee/sc2lib"
	"github.com/chippydip/go-sc2ai/api"
)

// Pass assignGroup = 0 to skip group assignement
func (b *bot) GetSCV(pos scl.Point, assignGroup scl.GroupID, minHits float64) *scl.Unit {
	scv1 := b.Groups.Get(ScvReserve).Units.ClosestTo(pos)
	scv2 := b.Groups.Get(Miners).Units.Filter(func(unit *scl.Unit) bool {
		return unit.IsGathering() && unit.Hits >= minHits
	}).ClosestTo(pos)

	scv := scv1
	if scv1 != nil && scv2 != nil {
		scv = scl.Units{scv1, scv2}.ClosestTo(pos)
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
	units := b.Units.Units()
	for _, unit := range units {
		if unit.IsStructure() && unit.TargetAbility() == abilityID {
			count++
		}
	}
	return count
}
