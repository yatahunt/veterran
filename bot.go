package main

import (
	"bitbucket.org/aisee/sc2lib"
)

func (b *bot) InitBot() {
	scl.InitUnits(b.Info.Data().Units)
	b.InitAliases()
	b.InitLocations()
	b.FindExpansions()
	b.InitMining()
	b.FindRamps()
	b.InitRamps()

	b.FindBuildingsPositions()

	/*for _, ramp := range b.Ramps {
		b.Debug2x2Buildings(b.FindRamp2x2Positions(ramp)...)
		b.Debug3x3Buildings(b.FindRampBarracksPositions(ramp))
	}
	b.DebugSend()*/
}

// OnStep is called each game step (every game update by default)
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
	if b.Loop == 8 {
		b.ChatSend("VeTerran v0.3.2 (glhf)")
	}

	b.Logic()

	b.Cmds.Process(&b.Actions)
	if len(b.Actions) > 0 {
		b.Info.SendActions(b.Actions)
		b.Actions = nil
	}
}
