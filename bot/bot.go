package bot

import (
	"bitbucket.org/aisee/sc2lib/point"
	"bitbucket.org/aisee/sc2lib/scl"
)

const version = "VeTerran v2.1.0 (glhf)"

type Bot struct {
	scl.Bot

	Logic func()

	IsRealtime     bool
	WorkerRush     bool
	LingRush       bool
	PlayDefensive  bool
	DefensiveRange float64
	BuildTurrets   bool
	MechPriority   bool

	BuildPos         map[scl.BuildingSize]point.Points
	FirstBarrack     point.Points
	TurretsPos       point.Points
	BunkersPos       point.Points
	FindTurretPosFor *scl.Unit

	DoubleHealers []scl.GroupID
}

var B = &Bot{
	PlayDefensive: true,
	BuildPos:      map[scl.BuildingSize]point.Points{},
}

func ParseData() {
	B.ParseObservation()
	B.ParseUnits()
	B.ParseOrders()
	B.DetectEnemyRace()
	if B.Locs.MyExps.Len() == 0 {
		B.Init()
		FindBuildingsPositions()
	}
	B.FindClusters()
}

func GGCheck() bool {
	return (B.Minerals < 50 &&
		B.Units.My.All().First(func(unit *scl.Unit) bool { return !unit.IsStructure() }) == nil &&
		B.Enemies.All.First(scl.DpsGt5) != nil) ||
		B.Units.My.All().Filter(scl.Structure, scl.NotFlying).Empty()
}

// OnStep is called each game step (every game update by default)
func Step() {
	defer scl.RecoverPanic()

	B.Cmds = &scl.CommandsStack{}
	B.Obs = B.Info.Observation().Observation
	B.Loop = int(B.Obs.GameLoop)
	if B.Loop != 0 && B.Loop-B.LastLoop != 1 && !B.IsRealtime {
		B.FramesPerOrder = 6
		B.IsRealtime = true
		B.Actions.ChatSend(version)
		B.Actions.ChatSend("Realtime mode detected")
	}
	if B.Loop == 8 && !B.IsRealtime {
		B.Actions.ChatSend(version)
	}
	if B.Loop != 0 && B.Loop == B.LastLoop {
		return // Skip frame repeat
	} else {
		B.LastLoop = B.Loop
	}

	ParseData()

	if GGCheck() {
		B.Actions.ChatSend("(gg)")
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

	for _, cr := range B.Info.Observation().Chat {
		if cr.Message == "s" {
			B.SaveState()
		}
	}

	// B.DebugOrders()
	// B.DebugMap()
	// B.DebugRamps()
	// B.DebugSafeGrid(B.Grid, B.SafeGrid)
	// B.DebugWayMap(B.SafeWayMap, true)
	// B.DebugEnemyUnits()
	// B.DebugClusters()
	// B.DebugSend()

	return
}
