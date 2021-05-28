package bot

import (
	log "bitbucket.org/aisee/minilog"
	"github.com/aiseeq/s2l/helpers"
	"github.com/aiseeq/s2l/lib/point"
	"github.com/aiseeq/s2l/lib/scl"
	"github.com/aiseeq/s2l/protocol/api"
)

const version = "VeTerran v2.2.8 (glhf)"

type Bot struct {
	*scl.Bot

	Logic func()

	WorkerRush     bool
	LingRush       bool
	PlayDefensive  bool
	DefensiveRange float64
	BuildTurrets   bool
	MechPriority   bool
	VersionPosted  bool

	BuildPos         map[scl.BuildingSize]point.Points
	FirstBarrack     point.Points
	TurretsPos       point.Points
	BunkersPos       point.Points

	DoubleHealers []scl.GroupID
}

var B *Bot

func ParseData() {
	B.ParseObservation()
	B.ParseUnits()
	B.ParseOrders()
	B.DetectEnemyRace()
	if len(B.BuildPos) == 0 {
		FindBuildingsPositions()
	}
	B.FindClusters()
}

func GGCheck() bool {
	return (B.Minerals < 50 &&
		B.Units.My.All().First(func(unit *scl.Unit) bool { return !unit.IsStructure() }) == nil &&
		B.Enemies.All.First(scl.DpsGt5) != nil) ||
		B.Units.My.All().Filter(scl.Structure, scl.Ground).Empty()
}

// OnStep is called each game step (every game update by default)
func Step() {
	defer helpers.RecoverPanic()

	B.Cmds = &scl.CommandsStack{} // todo: move this block into the lib
	B.Loop = int(B.Obs.GameLoop)
	/*if B.Loop != 0 && B.Loop-B.LastLoop != 1 && !B.IsRealtime {
		B.FramesPerOrder = 6
		B.IsRealtime = true
		B.Actions.ChatSend(version)
		B.Actions.ChatSend("Realtime mode detected")
	}*/                                  // todo: fix later
	if B.Loop >= 9 && !B.VersionPosted { // && !B.IsRealtime
		B.Actions.ChatSend(version)
		B.VersionPosted = true
	}
	if B.Loop < B.LastLoop+B.FramesPerOrder {
		return // Skip frame repeat
	} else {
		B.LastLoop = B.Loop
	}

	ParseData()

	if GGCheck() {
		B.Actions.ChatSend("(gg)")
		if _, err := B.Client.Action(api.RequestAction{Actions: B.Actions}); err != nil {
			log.Error(err)
		}
		if err := B.Client.LeaveGame(); err != nil {
			log.Error(err)
		}
		return
	}

	B.Logic()

	B.Cmds.Process(&B.Actions)
	if len(B.Actions) > 0 {
		// log.Info(B.Loop, len(B.Actions), B.Actions)
		if resp, err := B.Client.Action(api.RequestAction{Actions: B.Actions}); err != nil {
			log.Error(err)
		} else {
			_ = resp.Result // todo: do something with it?
		}
		B.Actions = nil
	}

	for _, cr := range B.Chat {
		if cr.Message == "s" {
			B.SaveState()
		}
	}

	/*if B.Loop%3 == 0 && B.Loop/3 >= 4 && B.Loop/3 < len(B.Locs.MyExps)+4 {
		FindTurretPosition(B.Locs.MyExps[B.Loop/3-4])
	}
	if B.Loop >= 60 && B.Loop < 63 {
		B.Debug2x2Buildings(B.TurretsPos...)
		B.DebugSend()
		log.Info(B.Loop)
	}*/
	// B.DebugOrders()
	// B.DebugMap()
	// B.DebugRamps()
	// B.Debug3x3Buildings(B.Locs.MyExps...)
	// B.DebugSafeGrid(B.Grid, B.SafeGrid)
	// B.DebugWayMap(B.SafeWayMap, true)
	// B.DebugEnemyUnits()
	// B.DebugClusters()
	/*B.Debug2x2Buildings(B.BuildPos[scl.S2x2]...)
	B.Debug3x3Buildings(B.BuildPos[scl.S3x3]...)
	B.Debug5x3Buildings(B.BuildPos[scl.S5x3]...)
	B.Debug3x3Buildings(B.BuildPos[scl.S5x5]...)
	B.Debug2x2Buildings(B.TurretsPos...)*/
	// B.DebugSend()

	return
}
