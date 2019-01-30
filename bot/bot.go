package bot

import (
	"bitbucket.org/aisee/sc2lib"
)

const version = "VeTerran v2.0.0 (glhf)"

type Bot struct {
	scl.Bot

	Logic func()

	IsRealtime     bool
	WorkerRush     bool
	LingRush       bool
	PlayDefensive  bool
	DefensiveRange float64
	BuildTurrets   bool

	BuildPos         map[scl.BuildingSize]scl.Points
	FirstBarrack     scl.Points
	TurretsPos       scl.Points
	BunkersPos       scl.Points
	FindTurretPosFor *scl.Unit

	DoubleHealers []scl.GroupID
}

var B = &Bot{
	PlayDefensive: true,
	BuildPos:      map[scl.BuildingSize]scl.Points{},
}

func InitBot() {
	scl.InitUnits(B.Info.Data().Units)
	scl.InitUpgrades(B.Info.Data().Upgrades)
	scl.InitEffects(B.Info.Data().Effects)
	B.InitLocations()
	B.FindExpansions()
	B.InitMining()
	B.FindRamps()
	B.InitRamps()
	go B.RenewPaths()

	FindBuildingsPositions()

	/*for _, ramp := range B.Ramps {
		B.Debug2x2Buildings(B.FindRamp2x2Positions(ramp)...)
		B.Debug3x3Buildings(B.FindRampBarracksPositions(ramp))
	}*/

	/*start := time.Now()
	for x := 1; x < 100; x++ {
		B.Path(B.Ramps.My.Top, B.EnemyRamp.Top)
	}
	path, dist := B.Path(B.Ramps.My.Top, B.EnemyRamp.Top)
	log.Info(time.Now().Sub(start), dist, path)
	B.DebugPath(path)
	B.DebugSend()*/

	/*start := time.Now()
	paths := B.FindPaths(B.Ramps.My.Top)
	log.Info(time.Now().Sub(start), paths)
	path := paths.From(B.EnemyRamp.Top)
	B.DebugPath(path)
	B.DebugSend()*/

	/*start := time.Now()
	path := B.HomePaths.From(B.EnemyRamp.Top)
	log.Info(time.Now().Sub(start))
	B.DebugPath(path)
	B.DebugSend()*/

	/*B.DebugMap()
	B.DebugSend()*/
}

func GGCheck() bool {
	return B.Minerals < 50 &&
		B.Units.My.All().First(func(unit *scl.Unit) bool { return !unit.IsStructure() }) == nil &&
		B.Enemies.All.First(scl.DpsGt5) != nil
}

// OnStep is called each game step (every game update by default)
func Step() {
	defer scl.RecoverPanic()

	B.Cmds = &scl.CommandsStack{}
	B.Obs = B.Info.Observation().Observation
	B.ParseObservation()
	if B.Loop != 0 && B.Loop-B.LastLoop != 1 && !B.IsRealtime {
		B.FramesPerOrder = 6
		B.IsRealtime = true
		B.ChatSend(version)
		B.ChatSend("Realtime mode detected")
	}
	if B.Loop == 8 && !B.IsRealtime {
		B.ChatSend(version)
	}
	if B.Loop != 0 && B.Loop == B.LastLoop {
		return // Skip frame repeat
	} else {
		B.LastLoop = B.Loop
	}

	B.ParseUnits()
	B.ParseOrders()
	B.DetectEnemyRace()

	if B.Locs.MyExps.Len() == 0 {
		InitBot()
	}

	if GGCheck() {
		B.ChatSend("(gg)")
		B.Info.SendActions(B.Actions)
		B.Info.LeaveGame()
		return
	}

	B.Logic()

	B.Cmds.Process(&B.Actions)
	if len(B.Actions) > 0 {
		// log.Info(B.Loop, len(B.Actions), B.Actions)
		B.Info.SendActions(B.Actions)
		B.Actions = nil
	}

	/*if B.Loop % 20 == 0 {
		B.DebugMap()
		B.DebugSend()
	}*/
	return
}
