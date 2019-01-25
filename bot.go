package main

import (
	"bitbucket.org/aisee/sc2lib"
)

func (b *bot) InitBot() {
	scl.InitUnits(b.Info.Data().Units)
	scl.InitUpgrades(b.Info.Data().Upgrades)
	scl.InitEffects(b.Info.Data().Effects)
	b.InitLocations()
	b.FindExpansions()
	b.InitMining()
	b.FindRamps()
	b.InitRamps()
	// b.LoadPaths(b.Info.GameInfo().MapName)
	go b.RenewPaths()

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
	paths := b.FindPaths(b.MainRamp.Top)
	log.Info(time.Now().Sub(start), paths)
	path := paths.From(b.EnemyRamp.Top)
	b.DebugPath(path)
	b.DebugSend()*/

	/*start := time.Now()
	path := b.HomePaths.From(b.EnemyRamp.Top)
	log.Info(time.Now().Sub(start))
	b.DebugPath(path)
	b.DebugSend()*/

	/*b.DebugMap()
	b.DebugSend()*/
}

func (b *bot) GGCheck() bool {
	return b.Minerals < 50 &&
		b.Units.Units().First(func(unit *scl.Unit) bool { return !unit.IsStructure() }) == nil &&
		b.AllEnemyUnits.Units().First(scl.DpsGt5) != nil
}

// OnStep is called each game step (every game update by default)
func (b *bot) Step() bool { // bool = is final
	defer scl.RecoverPanic()

	b.Cmds = &scl.CommandsStack{}
	b.Obs = b.Info.Observation().Observation
	b.ParseObservation()
	if b.Loop != 0 && b.Loop-b.LastLoop != 1 && !isRealtime {
		b.FramesPerOrder = 6
		isRealtime = true
		b.ChatSend("Realtime mode detected")
	}
	if b.Loop != 0 && b.LastLoop == 0 { // Second loop. For some reason chat sometimes doesn't work on the first loop
		b.ChatSend("VeTerran v1.3.0 (glhf)")
	}
	if b.Loop != 0 && b.Loop == b.LastLoop {
		return false // Skip frame repeat
	} else {
		b.LastLoop = b.Loop
	}

	b.ParseUnits()
	b.ParseOrders()
	b.DetectEnemyRace()

	if b.ExpLocs.Len() == 0 {
		b.InitBot()
	}

	if b.GGCheck() {
		b.ChatSend("(gg)")
		b.Info.SendActions(b.Actions)
		return true
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
	return false
}
