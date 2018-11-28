package main

import (
	"bitbucket.org/AiSee/sc2lib"
	"github.com/chippydip/go-sc2ai/api"
	"github.com/chippydip/go-sc2ai/search"
)

func (b *bot) InitBot() {
	scl.InitUnits(b.Info.Data().Units)
	b.InitLocations()
	for _, uc := range search.CalculateExpansionLocations(b.Info, false) {
		center := uc.Center()
		b.ExpLocs = append(b.ExpLocs, scl.Pt2(&center))
	}
	b.FindBuildingsPositions()
	b.Retreat = map[api.UnitTag]bool{}
}

// OnStep is called each game step (every game update by defaul)
func (b *bot) Step() {
	defer scl.RecoverPanic()

	b.Cmds = &scl.CommandsStack{}
	b.Obs = b.Info.Observation().Observation
	b.ParseObservation()
	b.ParseUnits()
	b.ParseOrders()

	if b.Obs.GameLoop == 0 {
		b.InitBot()
		b.InitMining()
		b.ChatSend("VeTerran 0.0.4 (glhf)")
	}

	b.Strategy()
	b.Tactics()

	b.Cmds.Process(&b.Actions)
	if len(b.Actions) > 0 {
		b.Info.SendActions(b.Actions)
		b.Actions = nil
	}
}
