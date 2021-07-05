package bot

import (
	log "bitbucket.org/aisee/minilog"
	"github.com/aiseeq/s2l/helpers"
	"github.com/aiseeq/s2l/lib/point"
	"github.com/aiseeq/s2l/lib/scl"
	"github.com/aiseeq/s2l/protocol/api"
	"github.com/aiseeq/s2l/protocol/enums/zerg"
)

const version = "VeTerran v2.6.5 (glhf)"

type Strategy int

const (
	Default Strategy = iota
	ProxyReapers
	ProxyMarines
	BruteForce
	CcAfterRax
	CcBeforeRax
	MaxStrategyId
)

var StrategyPriority = map[Strategy]float64{ // More is better
	BruteForce:   0.95,
	CcBeforeRax:  0.9,
	ProxyReapers: 0.85,
	CcAfterRax:   0.8,
	Default:      0.75,
	ProxyMarines: 0.7,
}

func (s Strategy) String() string {
	switch s {
	case Default:
		return "Default"
	case ProxyReapers:
		return "Proxy_Reapers"
	case ProxyMarines:
		return "Proxy_Marines"
	case BruteForce:
		return "Brute_Force"
	case CcAfterRax:
		return "CC_After_Rax"
	case CcBeforeRax:
		return "CC_Before_Rax"
	default:
		return "Unknown"
	}
}

type Bot struct {
	*scl.Bot

	Logic func()

	WorkerRush     bool
	PlayDefensive  bool
	DefensiveRange float64
	BuildTurrets   bool
	VersionPosted  bool
	GGPosted       bool
	PanicPosted    bool
	Strategy       Strategy
	ProxyReapers   bool
	ProxyMarines   bool
	BruteForce     bool
	CcAfterRax     bool
	CcBeforeRax    bool

	BuildPos         map[scl.BuildingSize]point.Points
	FirstBarrack     point.Points
	TurretsPos       point.Points
	TurretsMiningPos point.Points
	BunkersPos       point.Points

	DoubleHealers []scl.GroupID
	CycloneLocks  map[api.UnitTag]api.UnitTag

	Stats *GameData

	TestVal float64
}

var B *Bot

func (b *Bot) ParseActionErrors() {
	for _, err := range b.Errors {
		log.Debugf("Action tag: %v, ability: %v, error: %v", err.UnitTag, err.AbilityId, err.Result)
		if err.AbilityId == 318 && err.Result == api.ActionResult_CantBuildLocationInvalid {
			// Probably it's a burrowed zergling. Not a best solution but works most times
			scv := b.Units.ByTag[err.UnitTag]
			if scv == nil {
				log.Error("Wat?")
				continue
			}
			exp := B.Locs.MyExps.ClosestTo(scv)
			ling := scl.Unit{
				Unit: api.Unit{
					DisplayType: api.DisplayType_Hidden,
					Alliance:    api.Alliance_Enemy,
					UnitType:    zerg.ZerglingBurrowed,
					Pos:         exp.Point().To3D(),
					Cloak:       api.CloakState_Cloaked,
					IsBurrowed:  true,
				},
			}
			b.Enemies.All.Add(&ling)
			b.Enemies.AllReady.Add(&ling)
		}
	}
}

func ParseData() {
	B.ParseObservation()
	B.ParseUnits()
	B.ParseOrders()
	B.DetectEnemyRace()
	if len(B.BuildPos) == 0 {
		FindBuildingsPositions()
		B.InitMining(B.TurretsMiningPos)
	}
	B.FindClusters() // Not used yet
	B.ParseActionErrors()
}

func GGCheck() bool {
	return (B.Minerals < 50 &&
		B.Units.MyAll.First(func(unit *scl.Unit) bool { return !unit.IsStructure() }) == nil &&
		B.Enemies.All.First(scl.DpsGt5) != nil) ||
		B.Units.MyAll.Filter(scl.Structure, scl.Ground).Empty()
}

func RecoverPanic() {
	if p := recover(); p != nil {
		helpers.ReportPanic(p)
		if !B.PanicPosted {
			B.Actions.ChatSend("Tag: Panic", api.ActionChat_Team)
			B.PanicPosted = true
		}
		B.Cmds.Process(&B.Actions)
		if len(B.Actions) > 0 {
			// Send actions if there were any before panic has occurred
			_, _ = B.Client.Action(api.RequestAction{Actions: B.Actions})
			B.Actions = nil
		}
	}
}

// OnStep is called each game step (every game update by default)
func Step() {
	defer RecoverPanic()

	B.Cmds = &scl.CommandsStack{} // todo: move this block into the lib
	B.Loop = int(B.Obs.GameLoop)
	if B.Loop >= 9 && !B.VersionPosted {
		B.Actions.ChatSend(version, api.ActionChat_Broadcast)
		B.Actions.ChatSend("Tag: Strategy_"+B.Strategy.String(), api.ActionChat_Team)
		B.VersionPosted = true
	}
	if B.Loop < B.LastLoop+B.FramesPerOrder {
		return // Skip frame repeat
	} else {
		B.LastLoop = B.Loop
	}

	ParseData()
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

	if !B.GGPosted && GGCheck() {
		B.Actions.ChatSend("(gg)", api.ActionChat_Broadcast)
		_, _ = B.Client.Action(api.RequestAction{Actions: B.Actions})
		B.Actions = nil
		B.GGPosted = true
		if err := B.Client.LeaveGame(); err != nil {
			log.Error(err)
		}
	}

	for _, cr := range B.Chat {
		if cr.Message == "s" {
			B.SaveState()
		}
	}

	// B.DebugOrders()
	// B.DebugMap()
	// B.DebugRamps()
	// B.Debug3x3Buildings(B.Locs.MyExps...)
	// B.DebugSafeGrid(B.Grid, B.SafeGrid)
	// B.DebugWayMap(B.SafeWayMap, true)
	// B.DebugSafeGrid(B.ReaperGrid, B.ReaperSafeGrid)
	// B.DebugWayMap(B.ReaperSafeWayMap, true)
	// B.DebugEnemyUnits()
	// B.DebugClusters()
	/*B.Debug2x2Buildings(B.BuildPos[scl.S2x2]...)
	B.Debug3x3Buildings(B.BuildPos[scl.S3x3]...)
	B.Debug5x3Buildings(B.BuildPos[scl.S5x3]...)
	B.Debug3x3Buildings(B.BuildPos[scl.S5x5]...)
	B.Debug3x3Buildings(B.BunkersPos...)
	B.Debug2x2Buildings(B.TurretsPos...)*/
	/*if B.Loop < 3 {
		for _, pos := range B.TurretsPos {
			B.DebugAddUnits(terran.MissileTurret, B.Obs.PlayerCommon.PlayerId, pos, 1)
		}
	}*/
	// B.DebugCircles(*point.PtCircle(B.Ramps.My.Top-B.Ramps.My.Vec*3, 5))
	// B.DebugSend()

	return
}
