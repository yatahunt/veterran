package main

import (
	"bitbucket.org/AiSee/sc2lib"
	"github.com/chippydip/go-sc2ai/api"
)

func (b *bot) InitBot() {
	scl.InitUnits(b.Info.Data().Units)
	b.InitLocations()
	b.FindExpansions()
	b.InitMining()
	b.FindRamps()
	b.InitRamps()

	b.FindBuildingsPositions()
	b.Retreat = map[api.UnitTag]bool{}
}

// OnStep is called each game step (every game update by defaul)
func (b *bot) Step() {
	defer scl.RecoverPanic()

	b.Cmds = &scl.CommandsStack{}
	b.Obs = b.Info.Observation().Observation
	b.ParseObservation()
	if b.Loop != 0 && b.Loop == b.LastLoop {
		return // Skip frame repeat
	} else {
		b.LastLoop = b.Loop
	}

	b.ParseUnits()
	b.ParseOrders()
	b.DetectEnemyRace()

	if b.ExpLocs.Len() == 0 {
		b.InitBot()
	}
	if b.Loop == 1 {
		b.ChatSend("VeTerran 0.0.5 (glhf)")
	}

	b.Logic()

	b.Cmds.Process(&b.Actions)
	if len(b.Actions) > 0 {
		b.Info.SendActions(b.Actions)
		b.Actions = nil
	}
}
