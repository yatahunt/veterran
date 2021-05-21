package bot

import (
	log "bitbucket.org/aisee/minilog"
	"github.com/aiseeq/s2l/helpers"
	"github.com/aiseeq/s2l/lib/point"
	"github.com/aiseeq/s2l/lib/scl"
	"github.com/aiseeq/s2l/protocol/api"
)

const version = "VeTerran v2.2.1 (glhf)"

type Bot struct {
	*scl.Bot

	Logic func()

	IsRealtime     bool
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
	FindTurretPosFor *scl.Unit

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
		B.Units.My.All().Filter(scl.Structure, scl.NotFlying).Empty()
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
	}*/              // todo: fix later
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
