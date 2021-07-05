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
	/*{
		Name:    "Priority order workers",
		Ability: ability.Train_SCV,
		Premise: func() bool {
			return (B.CcAfterRax || B.CcBeforeRax) && B.Minerals < 150
		},
		Limit:  func() int { return 42 },
		Active: func() int { return 2 },
		Method: TrainScv,
	},*/
	{
		Name:    "Expansion CCs",
		Ability: ability.Build_CommandCenter,
		Premise: func() bool {
			return B.Enemies.All.Filter(scl.DpsGt5).CloserThan(B.DefensiveRange, B.Locs.MyStart).Empty()
		},
		Limit: func() int {
			if B.CcAfterRax || B.CcBeforeRax {
				return 2
			}
			if B.BruteForce && B.Loop < scl.TimeToLoop(2, 40) {
				return 0
			}
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
	},
	{
		Name:    "Supplies",
		Ability: ability.Build_SupplyDepot,
		Premise: func() bool {
			if B.CcAfterRax || B.CcBeforeRax {
				if B.Pending(ability.Train_SCV) < 14 {
					return false
				}
				if B.Units.My.OfType(B.U.UnitAliases.For(terran.SupplyDepot)...).Len() == 1 &&
					(B.Units.My.OfType(B.U.UnitAliases.For(terran.Barracks)...).Empty() ||
						B.Units.My.OfType(B.U.UnitAliases.For(terran.CommandCenter)...).Len() < 2) {
					return false
				}
			}
			return B.FoodLeft < 6+B.FoodUsed/20 && B.FoodCap < 200
		},
		Limit:  func() int { return 30 },
		Active: func() int { return 1 + B.FoodUsed/50 },
	},
	{
		Name:    "First Barrack",
		Ability: ability.Build_Barracks,
		Premise: func() bool {
			ccs := B.Units.My.OfType(B.U.UnitAliases.For(terran.CommandCenter)...)
			if B.CcBeforeRax && ccs.Len() == 1 {
				return false
			}
			return !B.ProxyReapers && !B.ProxyMarines &&
				B.Units.My.OfType(B.U.UnitAliases.For(terran.SupplyDepot)...).First(scl.Ready) != nil &&
				B.Units.My.OfType(B.U.UnitAliases.For(terran.Barracks)...).Empty()
		},
		Limit:  BuildOne,
		Active: BuildOne,
		Method: BuildFirstBarrack,
	},
	{
		Name:    "Proxy Barracks",
		Ability: ability.Build_Barracks,
		Premise: func() bool {
			return (B.ProxyReapers || B.ProxyMarines) &&
				B.Units.My.OfType(B.U.UnitAliases.For(terran.SupplyDepot)...).First(scl.Ready) != nil
		},
		Limit:  func() int { return 2 },
		Active: func() int { return 2 },
		Method: BuildProxyBarrack,
	},
	{
		Name:    "Barracks reactor for brute force",
		Ability: ability.Build_Reactor_Barracks,
		Premise: func() bool {
			rax := B.Units.My[terran.Barracks].First(scl.Ready, scl.NoAddon, scl.Idle)
			return B.BruteForce &&
				((B.Vespene >= 50 && rax != nil && B.Enemies.Visible.CloserThan(SafeBuildRange, rax).Empty()) ||
					B.Units.My[terran.BarracksFlying].First() != nil)
		},
		Limit:  BuildOne,
		Active: BuildOne,
		Method: func() {
			if rax := B.Units.My[terran.BarracksFlying].First(); rax != nil {
				rax.CommandPos(ability.Build_Reactor_Barracks, B.FirstBarrack[1])
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
					rax.Command(ability.Build_Reactor_Barracks)
				}
			}
		},
	},
	{
		Name:    "Refinery",
		Ability: ability.Build_Refinery,
		Premise: func() bool {
			if B.WorkerRush {
				return false
			}
			ccs := B.Units.My.OfType(B.U.UnitAliases.For(terran.CommandCenter)...)
			if (B.CcBeforeRax || B.CcAfterRax) && ccs.Len() == 1 {
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
					return refPending < ccs.Len()+1 || B.BruteForce
				}
				return true
			}
			return false
		},
		Limit: func() int {
			if B.CcBeforeRax {
				ccs := B.Units.My.OfType(B.U.UnitAliases.For(terran.CommandCenter)...).Filter(scl.Ready).Len()
				if ccs == 1 {
					return 1
				}
				if ccs == 2 {
					return 3
				}
			}
			return 20
		},
		Active: func() int {
			ccs := B.Units.My.OfType(B.U.UnitAliases.For(terran.CommandCenter)...).Filter(scl.Ready).Len()
			if (B.CcAfterRax || B.CcBeforeRax) && ccs < 3 {
				return 1
			}
			return 2
		},
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
			if B.Units.My.OfType(B.U.UnitAliases.For(terran.Factory)...).Len() >= 1 && B.Minerals <= 400 {
				return false // Don't build second if not plenty of resources
			}
			if B.Minerals > 800 {
				return true
			}
			return B.Units.My[terran.Factory].First(scl.Ready, scl.Unused) == nil
		},
		Limit: func() int {
			buildFacts := B.Units.My.OfType(B.U.UnitAliases.For(terran.CommandCenter)...).Filter(scl.Ready).Len()
			if B.EnemyRace == api.Race_Zerg && buildFacts > 1 {
				buildFacts--
			}
			starports := B.Units.My.OfType(B.U.UnitAliases.For(terran.Starport)...).Len()
			return scl.MinInt(4, scl.MinInt(buildFacts, starports+1))
		},
		Active:  BuildOne,
		Unlocks: FactoryBuildOrder,
	},
	{
		Name:    "Starport",
		Ability: ability.Build_Starport,
		Premise: func() bool {
			if B.Units.My.OfType(B.U.UnitAliases.For(terran.Starport)...).Len() >= 1 && B.Minerals <= 400 {
				return false // Don't build second if not plenty of resources
			}
			if B.Minerals > 800 {
				return true
			}
			return B.Units.My[terran.Starport].First(scl.Ready, scl.Unused) == nil
		},
		Limit: func() int {
			ccs := B.Units.My.OfType(B.U.UnitAliases.For(terran.CommandCenter)...).Filter(scl.Ready).Len()
			factories := B.Units.My.OfType(B.U.UnitAliases.For(terran.Factory)...).Len()
			return scl.MinInt(4, scl.MinInt(ccs, factories+1))
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
			return B.PlayDefensive && B.Units.My[terran.Marine].Len() >= 2
		},
		Limit:  func() int { return B.BunkersPos.Len() },
		Active: func() int { return B.BunkersPos.Len() },
		// WaitRes: Yes,
	},
	{
		Name:    "Armory",
		Ability: ability.Build_Armory,
		Premise: func() bool {
			cyclones := B.PendingAliases(ability.Train_Cyclone)
			tanks := B.PendingAliases(ability.Train_SiegeTank)
			return (cyclones > 0 || tanks > 0) && B.Units.My[terran.EngineeringBay].First(scl.Ready) != nil
		},
		// WaitRes: Yes,
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
			if B.CcAfterRax && B.Units.My[terran.Barracks].First(scl.Unused) != nil {
				return false
			}
			if B.Units.My.OfType(B.U.UnitAliases.For(terran.Barracks)...).Len() >= 2 && B.Minerals <= 450 {
				return false // Don't build third if not plenty of resources
			}
			if B.Minerals > 800 {
				return true
			}
			return !B.WorkerRush && B.Units.My[terran.BarracksFlying].Empty() &&
				B.Units.My[terran.Barracks].First(scl.Ready, scl.Unused) == nil
		},
		Limit: func() int {
			ccs := B.Units.My.OfType(B.U.UnitAliases.For(terran.CommandCenter)...)
			if B.BruteForce {
				return scl.MinInt(4, ccs.Len())
			}
			return scl.MinInt(4, ccs.Len()+1)
		},
		Active: func() int {
			return 2
		},
	},
	{
		Name:    "Engineering Bay",
		Ability: ability.Build_EngineeringBay,
		Premise: func() bool {
			if B.CcAfterRax || B.CcBeforeRax {
				return B.Units.My.OfType(B.U.UnitAliases.For(terran.Factory)...).First(scl.Ready) != nil
			}
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
		Limit:  func() int { return B.TurretsPos.Len() },
		Active: func() int { return B.TurretsPos.Len() },
		// WaitRes: Yes,
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
		Limit: func() int {
			return B.Units.My.OfType(B.U.UnitAliases.For(terran.Barracks)...).Len() - 1
		},
		Active: BuildOne,
		Method: func() {
			if rax := B.Units.My[terran.BarracksFlying].First(); rax != nil {
				rax.CommandPos(ability.Build_TechLab_Barracks, B.FirstBarrack[1])
				return
			}

			raxes := B.Units.My[terran.Barracks]
			rax := raxes.First(scl.Ready, scl.NoAddon, scl.Idle)
			if B.BruteForce || raxes.Len() >= 3 {
				rax.Command(ability.Build_TechLab_Barracks)
				return
			}
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
		Limit:  BuildOne,
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
			return allFactories.Len() // - 1
		},
		Active: BuildOne,
		Method: func() {
			B.Units.My[terran.Factory].First(scl.Ready, scl.NoAddon, scl.Idle).Command(ability.Build_TechLab_Factory)
		},
	},
	/*{
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
	},*/
}

var StarportBuildOrder = BuildNodes{
	{
		Name:    "Starport Tech Lab",
		Ability: ability.Build_TechLab_Starport,
		Premise: func() bool {
			if B.BruteForce && B.Pending(ability.Train_Medivac) == 0 {
				return false
			}
			starport := B.Units.My[terran.Starport].First(scl.Ready, scl.NoAddon, scl.Idle)
			return (starport != nil) && B.Enemies.Visible.CloserThan(SafeBuildRange, starport).Empty()
		},
		Limit: func() int {
			allStarports := B.Units.My.OfType(B.U.UnitAliases.For(terran.Starport)...)
			if allStarports.Len() == 1 {
				return 1
			}
			return allStarports.Len() // - 1
		},
		Active: BuildOne,
		Method: func() {
			B.Units.My[terran.Starport].First(scl.Ready, scl.NoAddon, scl.Idle).Command(ability.Build_TechLab_Starport)
		},
	},
	/*{
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
	},*/
	{
		Name:    "Fusion Core",
		Ability: ability.Build_FusionCore,
		Premise: func() bool {
			return B.Units.My[terran.Raven].Exists() && B.Units.My[terran.Starport].Filter(scl.Ready).Len() >= 2
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
	if techReq != 0 && B.Units.My.OfType(B.U.UnitAliases.For(techReq)...).First(scl.Ready) == nil {
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
	if aid == ability.Build_MissileTurret || aid == ability.Build_Bunker {
		// Build only if CC exists or in construction nearby
		ccs := B.Units.My.OfType(terran.CommandCenter, terran.OrbitalCommand, terran.PlanetaryFortress)
		positions = positions.Filter(func(pt point.Point) bool {
			return ccs.CloserThan(scl.ResourceSpreadDistance, pt).Exists()
		})
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
			if !scv.IsSafeToApproach(pos) {
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
	scv := B.Units.My[terran.SCV].CloserThan(5, pos).ClosestTo(pos)
	if scv == nil {
		scv = bot.GetSCV(pos, 0, 45)
	}
	if scv != nil {
		B.Groups.Add(bot.Builders, scv)
		OrderBuild(scv, pos, ability.Build_Barracks)
	}
}

func BuildProxyBarrack() {
	pos := B.Locs.EnemyExps[B.Units.My[terran.Barracks].Len()+2]
	scv := B.Units.My[terran.SCV].CloserThan(5, pos).ClosestTo(pos)
	if scv == nil {
		scv = bot.GetSCV(pos, 0, 45)
	}
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
		enemies := B.Enemies.All.Filter(scl.DpsGt5)
		if enemies.CloserThan(SafeBuildRange, geyser).Exists() || enemies.First(func(unit *scl.Unit) bool {
			return unit.IsCloserThan(unit.GroundRange()+4, geyser)
		}) != nil {
			return
		}
		scv := bot.GetSCV(geyser, 0, 45)
		if scv != nil {
			if !scv.IsSafeToApproach(geyser) {
				log.Debugf("Can't find safe path from %v to build %v @ %v",
					scv.Pos, B.U.Types[B.U.AbilityUnit[ability.Build_Refinery]].Name, geyser.Point())
				return
			}
			B.Groups.Add(bot.Builders, scv)
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
