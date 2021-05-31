package macro

import (
	"bitbucket.org/aisee/minilog"
	"bitbucket.org/aisee/veterran/bot"
	"github.com/aiseeq/s2l/lib/point"
	"github.com/aiseeq/s2l/lib/scl"
	"github.com/aiseeq/s2l/protocol/api"
	"github.com/aiseeq/s2l/protocol/enums/ability"
	"github.com/aiseeq/s2l/protocol/enums/terran"
	"math"
)

type Booler func() bool
type Inter func() int
type Voider func()
type BuildNode struct {
	Name    string
	Ability api.AbilityID
	Premise Booler
	WaitRes Booler
	Limit   Inter
	Active  Inter
	Method  Voider
	Unlocks BuildNodes
}
type BuildNodes []BuildNode

func BuildOne() int { return 1 }
func Yes() bool     { return true }

var BuildingsSizes = map[api.AbilityID]scl.BuildingSize{
	ability.Build_CommandCenter:  scl.S5x5,
	ability.Build_SupplyDepot:    scl.S2x2,
	ability.Build_Barracks:       scl.S5x3,
	ability.Build_Refinery:       scl.S3x3,
	ability.Build_EngineeringBay: scl.S3x3,
	ability.Build_MissileTurret:  scl.S2x2,
	ability.Build_Bunker:         scl.S3x3,
	ability.Build_Armory:         scl.S3x3,
	ability.Build_Factory:        scl.S5x3,
	ability.Build_Starport:       scl.S5x3,
	ability.Build_FusionCore:     scl.S3x3,
}

var RootBuildOrder = BuildNodes{
	{
		Name:    "Expansion CCs",
		Ability: ability.Build_CommandCenter,
		Premise: func() bool {
			return B.Enemies.All.Filter(scl.DpsGt5).CloserThan(B.DefensiveRange, B.Locs.MyStart).Empty()
		},
		Limit:  func() int {
			if B.Loop < scl.TimeToLoop(2, 0) {
				return 1
			} else if B.Loop < scl.TimeToLoop(4, 0) {
				return 2
			}
			return B.BuildPos[scl.S5x5].Len()
		},
		Active: func() int { return 1 + B.Minerals/800 },
		WaitRes: func() bool {
			ccs := B.Units.My.OfType(B.U.UnitAliases.For(terran.CommandCenter)...)
			if ccs.Len() <= B.FoodUsed/40 {
				return true
			}
			return false
		},
		Method: func() {
			pos := Build(ability.Build_CommandCenter)
			if pos != 0 { // && B.PlayDefensive
				bot.FindBunkerPosition(pos)
			}
		},
	},
	{
		Name:    "Supplies",
		Ability: ability.Build_SupplyDepot,
		Premise: func() bool {
			// its safe && 1 depo is placed && < 2:00 && only one cc
			if !B.PlayDefensive && B.FoodCap > 20 && B.Loop < 2688 &&
				B.Units.My.OfType(B.U.UnitAliases.For(terran.CommandCenter)...).Len() == 1 {
				return false // Wait for a second cc
			}
			/*if B.Loop < 1344 && B.FoodUsed < 14 {
				return false // Train SCVs without delay
			}*/
			return B.FoodLeft < 6+B.FoodUsed/20 && B.FoodCap < 200
		},
		Limit:  func() int { return 30 },
		Active: func() int { return 1 + B.FoodUsed/50 },
	},
	{
		Name:    "First Barrack",
		Ability: ability.Build_Barracks,
		Premise: func() bool {
			return B.Units.My.OfType(B.U.UnitAliases.For(terran.SupplyDepot)...).First(scl.Ready) != nil &&
				B.Units.My.OfType(B.U.UnitAliases.For(terran.Barracks)...).Empty()
		},
		Limit:  BuildOne,
		Active: BuildOne,
		Method: func() { BuildFirstBarrack() },
	},
	{
		Name:    "Refinery",
		Ability: ability.Build_Refinery,
		Premise: func() bool {
			if B.WorkerRush {
				return false
			}
			if B.Vespene < B.Minerals*2 {
				raxPending := B.Pending(ability.Build_Barracks)
				refPending := B.Pending(ability.Build_Refinery)
				ccs := B.Units.My.OfType(B.U.UnitAliases.For(terran.CommandCenter)...)
				if raxPending == 0 {
					return false
				}
				if B.Minerals > 350 {
					return true
				}
				if ccs.Len() < 3 {
					return refPending < ccs.Len() + 1
				}
				return true
			}
			return false
		},
		Limit:  func() int { return 20 },
		Active: func() int { return 2 },
		Method: func() {
			ccs := B.Units.My.OfType(terran.CommandCenter, terran.OrbitalCommand, terran.PlanetaryFortress)
			if cc := ccs.First(scl.Ready); cc != nil {
				BuildRefinery(cc)
			}
		},
		Unlocks: RaxBuildOrder,
	},
	{
		Name:    "Factory",
		Ability: ability.Build_Factory,
		Premise: func() bool {
			return B.Units.My[terran.Factory].First(scl.Ready, scl.Unused) == nil
		},
		Limit: func() int {
			buildFacts := B.Units.My.OfType(B.U.UnitAliases.For(terran.CommandCenter)...).Filter(scl.Ready).Len()
			if B.EnemyRace == api.Race_Zerg && buildFacts > 1 {
				buildFacts--
			}
			return scl.MinInt(3, buildFacts)
		},
		Active:  BuildOne,
		Unlocks: FactoryBuildOrder,
	},
	{
		Name:    "Starport",
		Ability: ability.Build_Starport,
		Premise: func() bool {
			return B.Units.My[terran.Starport].First(scl.Ready, scl.Unused) == nil
		},
		Limit: func() int {
			ccs := B.Units.My.OfType(B.U.UnitAliases.For(terran.CommandCenter)...).Filter(scl.Ready).Len()
			/*if ccs.Len() < 3 && B.Minerals < 500 {
				return 0
			}
			if B.Units.My[terran.FusionCore].First(scl.Ready) == nil {
				return 2
			}*/
			return scl.MinInt(3, ccs)
		},
		Active:  BuildOne,
		Unlocks: StarportBuildOrder,
	},
}

var RaxBuildOrder = BuildNodes{
	{
		Name:    "Bunkers",
		Ability: ability.Build_Bunker,
		Premise: func() bool {
			return B.Units.My.OfType(terran.Marine, terran.Reaper).Len() >= 2 &&
				B.Enemies.All.Filter(scl.DpsGt5).CloserThan(B.DefensiveRange, B.Locs.MyStart).Empty()
		},
		Limit:   func() int { return B.BunkersPos.Len() },
		Active:  func() int { return B.BunkersPos.Len() },
		WaitRes: Yes,
	},
	{
		Name:    "Armory",
		Ability: ability.Build_Armory,
		Premise: func() bool {
			cyclones := B.PendingAliases(ability.Train_Cyclone)
			tanks := B.PendingAliases(ability.Train_SiegeTank)
			return cyclones > 0 && tanks > 0 && B.Units.My[terran.EngineeringBay].First(scl.Ready) != nil
		},
		WaitRes: Yes,
		Limit: func() int {
			if B.Units.My[terran.FusionCore].First(scl.Ready) != nil {
				return 2
			}
			return 1
		},
		Active: BuildOne,
	},
	{
		Name:    "Barracks",
		Ability: ability.Build_Barracks,
		Premise: func() bool {
			// B.Units.My[terran.Barracks].First(scl.Ready, scl.Unused) == nil &&
			return !B.WorkerRush && B.Units.My[terran.BarracksFlying].Empty()
		},
		Limit: func() int {
			// ccs := B.Units.My.OfType(B.U.UnitAliases.For(terran.CommandCenter)...)
			return 2 // scl.MinInt(2, ccs.Len())
		},
		Active: func() int {
			return 2
		},
	},
	{
		Name:    "Engineering Bay",
		Ability: ability.Build_EngineeringBay,
		Premise: func() bool {
			return B.Units.My.OfType(B.U.UnitAliases.For(terran.CommandCenter)...).Filter(scl.Ready).Len() >= 2
		},
		Limit:  BuildOne,
		Active: BuildOne,
	},
	{
		Name:    "Missile Turrets",
		Ability: ability.Build_MissileTurret,
		Premise: func() bool {
			return B.BuildTurrets
		},
		Limit:   func() int { return B.TurretsPos.Len() },
		Active:  func() int { return B.TurretsPos.Len() },
		WaitRes: Yes,
	},
	{
		Name:    "Barracks techlabs",
		Ability: ability.Build_TechLab_Barracks,
		Premise: func() bool {
			ccs := B.Units.My.OfType(B.U.UnitAliases.For(terran.CommandCenter)...).Filter(scl.Ready)
			rax := B.Units.My[terran.Barracks].First(scl.Ready, scl.NoAddon, scl.Idle)
			return ccs.Len() >= 2 &&
				((B.Vespene >= 100 && rax != nil && B.Enemies.Visible.CloserThan(SafeBuildRange, rax).Empty()) ||
					B.Units.My[terran.BarracksFlying].First() != nil)
		},
		Limit:  BuildOne, // B.Units.My.OfType(B.U.UnitAliases.For(terran.Barracks)...).Len()
		Active: BuildOne,
		Method: func() {
			// todo: group?
			if rax := B.Units.My[terran.BarracksFlying].First(); rax != nil {
				rax.CommandPos(ability.Build_TechLab_Barracks, B.FirstBarrack[1])
				return
			}

			rax := B.Units.My[terran.Barracks].First(scl.Ready, scl.NoAddon, scl.Idle)
			if rax.IsCloserThan(3, B.FirstBarrack[0]) {
				if B.FirstBarrack[0] != B.FirstBarrack[1] {
					if B.Enemies.Visible.CloserThan(SafeBuildRange, rax).Exists() {
						return
					}
					rax.Command(ability.Lift_Barracks)
				} else {
					rax.Command(ability.Build_TechLab_Barracks)
				}
			}
		},
	},
	{
		Name:    "Barracks reactors",
		Ability: ability.Build_Reactor_Barracks,
		Premise: func() bool {
			ccs := B.Units.My.OfType(B.U.UnitAliases.For(terran.CommandCenter)...).Filter(scl.Ready)
			rax := B.Units.My[terran.Barracks].First(scl.Ready, scl.NoAddon, scl.Idle)
			return ccs.Len() >= 2 && (rax != nil) && B.Enemies.Visible.CloserThan(SafeBuildRange, rax).Empty()
		},
		Limit: func() int {
			return B.Units.My.OfType(B.U.UnitAliases.For(terran.Barracks)...).Len() - 1
		},
		Active: BuildOne,
		Method: func() {
			rax := B.Units.My[terran.Barracks].First(scl.Ready, scl.NoAddon, scl.Idle)
			if rax.IsCloserThan(3, B.FirstBarrack[0]) {
				return
			}
			rax.Command(ability.Build_Reactor_Barracks)
		},
	},
}

var FactoryBuildOrder = BuildNodes{
	{
		Name:    "Factory Tech Lab",
		Ability: ability.Build_TechLab_Factory,
		Premise: func() bool {
			factory := B.Units.My[terran.Factory].First(scl.Ready, scl.NoAddon, scl.Idle)
			return (factory != nil) && B.Enemies.Visible.CloserThan(SafeBuildRange, factory).Empty()
		},
		Limit: func() int {
			allFactories := B.Units.My.OfType(B.U.UnitAliases.For(terran.Factory)...)
			if allFactories.Len() == 1 {
				return 1
			}
			return allFactories.Len() - 1
		},
		Active: BuildOne,
		Method: func() {
			B.Units.My[terran.Factory].First(scl.Ready, scl.NoAddon, scl.Idle).Command(ability.Build_TechLab_Factory)
		},
	},
	{
		Name:    "Factory Reactor",
		Ability: ability.Build_Reactor_Factory,
		Premise: func() bool {
			factory := B.Units.My[terran.Factory].First(scl.Ready, scl.NoAddon, scl.Idle)
			return B.Units.My[terran.FactoryTechLab].Exists() && (factory != nil) &&
				B.Enemies.Visible.CloserThan(SafeBuildRange, factory).Empty()
		},
		Limit: func() int { // Build one but after tech lab
			allFactories := B.Units.My.OfType(B.U.UnitAliases.For(terran.Factory)...)
			if allFactories.Len() == 1 {
				return 0
			}
			return 1
		},
		Active: BuildOne,
		Method: func() {
			B.Units.My[terran.Factory].First(scl.Ready, scl.NoAddon, scl.Idle).Command(ability.Build_Reactor_Factory)
		},
	},
}

var StarportBuildOrder = BuildNodes{
	{
		Name:    "Starport Tech Lab",
		Ability: ability.Build_TechLab_Starport,
		Premise: func() bool {
			starport := B.Units.My[terran.Starport].First(scl.Ready, scl.NoAddon, scl.Idle)
			return (starport != nil) && B.Enemies.Visible.CloserThan(SafeBuildRange, starport).Empty()
		},
		Limit: func() int {
			allStarports := B.Units.My.OfType(B.U.UnitAliases.For(terran.Starport)...)
			if allStarports.Len() == 1 {
				return 1
			}
			return allStarports.Len() - 1
		},
		Active: BuildOne,
		Method: func() {
			B.Units.My[terran.Starport].First(scl.Ready, scl.NoAddon, scl.Idle).Command(ability.Build_TechLab_Starport)
		},
	},
	{
		Name:    "Starport Reactor",
		Ability: ability.Build_Reactor_Starport,
		Premise: func() bool {
			starport := B.Units.My[terran.Starport].First(scl.Ready, scl.NoAddon, scl.Idle)
			return B.Units.My[terran.StarportTechLab].Exists() && (starport != nil) &&
				B.Enemies.Visible.CloserThan(SafeBuildRange, starport).Empty()
		},
		Limit: func() int {
			allStarports := B.Units.My.OfType(B.U.UnitAliases.For(terran.Starport)...)
			if allStarports.Len() == 1 {
				return 0
			}
			return 1
		},
		Active: BuildOne,
		Method: func() {
			B.Units.My[terran.Starport].First(scl.Ready, scl.NoAddon, scl.Idle).Command(ability.Build_Reactor_Starport)
		},
	},
	{
		Name:    "Fusion Core",
		Ability: ability.Build_FusionCore,
		Premise: func() bool {
			return B.Units.My[terran.Raven].Exists() && B.Units.My[terran.Banshee].Exists() &&
				B.Units.My[terran.VikingFighter].Exists()
		},
		Limit:  BuildOne,
		Active: BuildOne,
	},
}

func OrderBuild(scv *scl.Unit, pos point.Point, aid api.AbilityID) {
	scv.CommandPos(aid, pos)
	B.DeductResources(aid)
	log.Debugf("%d: Building %v @ %v", B.Loop, B.U.Types[B.U.AbilityUnit[aid]].Name, pos)
	// B.DebugPoints(pos)
}

func Build(aid api.AbilityID) point.Point {
	size, ok := BuildingsSizes[aid]
	if !ok {
		log.Alertf("Can't find size for %v", B.U.Types[B.U.AbilityUnit[aid]].Name)
		return 0
	}

	techReq := B.U.Types[B.U.AbilityUnit[aid]].TechRequirement
	if techReq != 0 && B.Units.My.OfType(B.U.UnitAliases.For(techReq)...).Empty() {
		log.Debugf("Tech requirement didn't met for %v", B.U.Types[B.U.AbilityUnit[aid]].Name)
		return 0 // Not available because of tech reqs, like: supply is needed for barracks
	}

	buildersTargets := point.Points{}
	for _, builder := range B.Groups.Get(bot.Builders).Units {
		buildersTargets.Add(builder.TargetPos())
	}

	enemies := B.Enemies.All.Filter(scl.DpsGt5)
	positions := B.BuildPos[size]
	if size == scl.S3x3 {
		// Add larger building positions if there is not enough S3x3 positions
		positions = append(positions, B.BuildPos[scl.S5x3]...)
	}
	if aid == ability.Build_MissileTurret {
		positions = B.TurretsPos
	}
	if aid == ability.Build_Bunker {
		positions = B.BunkersPos
	}
	for _, pos := range positions {
		if buildersTargets.CloserThan(math.Sqrt2, pos).Exists() {
			continue // Someone already constructing there
		}
		if !B.IsPosOk(pos, size, 0, scl.IsBuildable, scl.IsNoCreep) {
			continue
		}
		if enemies.CloserThan(SafeBuildRange, pos).Exists() || enemies.First(func(unit *scl.Unit) bool {
			return unit.IsCloserThan(unit.GroundRange()+4, pos)
		}) != nil {
			continue
		}
		if B.PlayDefensive && aid == ability.Build_CommandCenter &&
			pos.IsFurtherThan(B.DefensiveRange, B.Locs.MyStart) {
			continue
		}
		if B.RequestPathing(B.Locs.MyStart, pos) == 0 {
			log.Debugf("Can't find path to build %v @ %v", B.U.Types[B.U.AbilityUnit[aid]].Name, pos)
			continue // No path
		}

		scv := bot.GetSCV(pos, 0, 45)
		if scv != nil {
			if !B.RequestPlacement(aid, pos, scv) {
				log.Debugf("Bad place to build %v @ %v", B.U.Types[B.U.AbilityUnit[aid]].Name, pos)
				continue
			}
			p := point.Pt3(scv.Pos)
			if !B.SafeGrid.IsPathable(p) {
				if pathablePos := B.FindClosestPathable(B.SafeGrid, p); pathablePos != 0 {
					p = pathablePos
				}
			}
			if path, _ := scl.NavPath(B.Grid, B.SafeWayMap, p, pos); path == nil {
				log.Debugf("Can't find safe path from %v to build %v @ %v",
					scv.Pos, B.U.Types[B.U.AbilityUnit[aid]].Name, pos)
				return 0
			}
			B.Groups.Add(bot.Builders, scv)
			OrderBuild(scv, pos, aid)
			return pos
		}
		log.Debugf("%d: Failed to find SCV", B.Loop)
		return 0
	}
	log.Debugf("%d: Can't find position for %v", B.Loop, B.U.Types[B.U.AbilityUnit[aid]].Name)
	return 0
}

func BuildFirstBarrack() {
	pos := B.FirstBarrack[0]
	scv := B.Units.My[terran.SCV].ClosestTo(pos)
	if scv != nil {
		B.Groups.Add(bot.Builders, scv)
		OrderBuild(scv, pos, ability.Build_Barracks)
	}
}

func BuildRefinery(cc *scl.Unit) {
	// Find first geyser that is close to selected cc, but it doesn't have Refinery on top of it
	builders := B.Groups.Get(bot.Builders).Units
	geyser := B.Units.Geysers.All().CloserThan(10, cc).First(func(unit *scl.Unit) bool {
		return B.Units.My[terran.Refinery].CloserThan(1, unit).Len() == 0 &&
			unit.FindAssignedBuilder(builders) == nil
	})
	if geyser != nil {
		scv := bot.GetSCV(geyser, bot.Builders, 45)
		if scv != nil {
			scv.CommandTag(ability.Build_Refinery, geyser.Tag)
			B.DeductResources(ability.Build_Refinery)
			log.Debugf("%d: Building Refinery", B.Loop)
		}
	}
}

func ProcessBuildOrder(buildOrder BuildNodes) {
	for _, node := range buildOrder {
		// - B.Orders[node.Ability] - because B.Pending(node.Ability) returns actual buildings + assigned builders
		inLimits := B.Pending(node.Ability)-B.Orders[node.Ability] < node.Limit() &&
			B.Orders[node.Ability] < node.Active()
		// log.Info(node.Name, " ", B.Pending(node.Ability), node.Limit(), B.Orders[node.Ability], node.Active())
		canBuy := B.CanBuy(node.Ability)
		waitRes := node.WaitRes != nil && node.WaitRes()
		if (node.Premise == nil || node.Premise()) && inLimits && (canBuy || waitRes) {
			if !canBuy && waitRes {
				// reserve money for building
				B.DeductResources(node.Ability)
				continue
			}
			if node.Method != nil {
				node.Method()
			} else {
				Build(node.Ability)
			}
		}
		if node.Unlocks != nil && B.Units.My[B.U.AbilityUnit[node.Ability]].Exists() {
			ProcessBuildOrder(node.Unlocks)
		}
	}
}
