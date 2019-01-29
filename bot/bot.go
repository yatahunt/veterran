package bot

import (
	"bitbucket.org/aisee/minilog"
	"bitbucket.org/aisee/sc2lib"
	"github.com/chippydip/go-sc2ai/client"
	"strings"
)

const version = "VeTerran v1.4.0 (glhf)"
const SafeBuildRange = 7
const (
	Miners scl.GroupID = iota + 1
	// MinersRetreat
	Builders
	Repairers
	ScvHealer
	UnitHealers
	WorkerRushDefenders
	Scout
	ScoutBase
	ScvReserve
	Marines
	Marauders
	Reapers
	ReapersRetreat
	Cyclones
	WidowMines
	WidowMinesRetreat
	Hellions
	Tanks
	TanksOnExps
	Medivacs
	Ravens
	Battlecruisers
	MechRetreat
	MechHealing
	UnderConstruction
	Buildings
	MaxGroup
)

type Bot struct {
	*scl.Bot

	IsRealtime     bool
	WorkerRush     bool
	LingRush       bool
	PlayDefensive  bool
	DefensiveRange float64
	BuildTurrets   bool
	LastBuildLoop  int

	BuildPos              map[scl.BuildingSize]scl.Points
	FirstBarrack          scl.Points
	TurretsPos            scl.Points
	BunkersPos            scl.Points
	FindTurretPositionFor *scl.Unit

	DoubleHealers []scl.GroupID
}

var B = &Bot{
	PlayDefensive: true,
	BuildPos:      map[scl.BuildingSize]scl.Points{},
}

func RunAgent(info client.AgentInfo) {
	B.Info = info
	B.FramesPerOrder = 3
	B.MaxGroup = MaxGroup
	if B.Info.IsRealtime() {
		B.FramesPerOrder = 6
		log.Info("Realtime mode")
	}
	B.UnitCreatedCallback = B.OnUnitCreated

	for B.Info.IsInGame() {
		Step()

		if err := B.Info.Step(1); err != nil {
			log.Error(err)
			if strings.Contains(err.Error(), "An existing connection was forcibly closed by the remote host.") {
				return
			}
		}
	}
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
		B.Units.AllEnemy.All().First(scl.DpsGt5) != nil
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

	Logic()

	B.Cmds.Process(&B.Actions)
	if len(B.Actions) > 0 {
		B.Info.SendActions(B.Actions)
		B.Actions = nil
	}

	/*if B.Loop % 20 == 0 {
		B.DebugMap()
		B.DebugSend()
	}*/
	return
}
