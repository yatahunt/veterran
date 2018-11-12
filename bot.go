package main

import (
	"bitbucket.org/AiSee/sc2lib"
	"github.com/chippydip/go-sc2ai/api"
	"github.com/chippydip/go-sc2ai/enums/terran"
	"github.com/chippydip/go-sc2ai/search"
)

func (b *bot) initLocations() {
	// My CC is on start position
	b.startLocation = b.units[terran.CommandCenter].First().Point()
	b.enemyStartLocation = scl.Pt2(b.info.GameInfo().StartRaw.StartLocations[0])
}

func (b *bot) initBot() {
	scl.InitUnits(b.info.Data().Units)
	b.initLocations()
	for _, uc := range search.CalculateExpansionLocations(b.info, false) {
		center := uc.Center()
		b.baseLocations = append(b.baseLocations, scl.Pt2(&center))
	}
	b.findBuildingsPositions()
	b.retreat = map[api.UnitTag]bool{}
}

// OnStep is called each game step (every game update by defaul)
func (b *bot) step() {
	defer scl.RecoverPanic()

	b.obs = b.info.Observation().Observation
	b.parseObservation()
	b.parseUnits()
	b.parseOrders()

	if b.obs.GameLoop == 0 {
		b.initBot()
		b.chatSend("VeTerran 0.0.2 (glhf)")
	}

	b.strategy()
	b.tactics()

	if len(b.actions) > 0 {
		b.info.SendActions(b.actions)
		b.actions = nil
	}
}
