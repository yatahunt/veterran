package main

import (
	"bitbucket.org/aisee/sc2lib"
	"github.com/chippydip/go-sc2ai/api"
)

func (b *bot) GetSCV(pos scl.Point, assignGroup scl.GroupID, minHits float64) *scl.Unit {
	scv := b.Groups.Get(Miners).Units.Filter(func(unit *scl.Unit) bool {
		return unit.IsGathering() && unit.Hits >= minHits
	}).ClosestTo(pos)
	if scv != nil {
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
