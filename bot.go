package main

import (
	"bitbucket.org/aisee/sc2lib"
)

func (b *bot) InitBot() {
	scl.InitUnits(b.Info.Data().Units)
	b.InitLocations()
	b.FindExpansions()
	b.InitMining()
	b.FindRamps()
	b.InitRamps()
	go b.InitPathes()

	b.FindBuildingsPositions()

	/*for _, ramp := range b.Ramps {
		b.Debug2x2Buildings(b.FindRamp2x2Positions(ramp)...)
		b.Debug3x3Buildings(b.FindRampBarracksPositions(ramp))
	}*/

	/*start := time.Now()
	for x := 1; x < 100; x++ {
		b.Path(b.MainRamp.Top, b.EnemyRamp.Top)
	}
	path, dist := b.Path(b.MainRamp.Top, b.EnemyRamp.Top)
	log.Info(time.Now().Sub(start), dist, path)
	b.DebugPath(path)
	b.DebugSend()*/

	/*start := time.Now()
	pathes := b.FindPathes(b.MainRamp.Top)
	log.Info(time.Now().Sub(start), pathes)
	path := pathes.From(b.EnemyRamp.Top)
	b.DebugPath(path)
	b.DebugSend()*/

	/*start := time.Now()
	path := b.HomePathes.From(b.EnemyRamp.Top)
	log.Info(time.Now().Sub(start))
	b.DebugPath(path)
	b.DebugSend()*/

	/*b.DebugMap()
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
	} else if b.Loop % 20 == 0 { // todo: проверку получше
		go b.InitPathes()
		/* b.DebugPath(b.HomePathes.From(b.EnemyRamp.Top))
		b.DebugSend() */
	}
	if b.Loop == 8 {
		b.ChatSend("VeTerran v0.4.1 (glhf)")
	}

	b.Logic()

	b.Cmds.Process(&b.Actions)
	if len(b.Actions) > 0 {
		b.Info.SendActions(b.Actions)
		b.Actions = nil
	}

	/*if b.Loop % 20 == 0 {
		b.DebugMap()
		b.DebugSend()
	}*/
}
