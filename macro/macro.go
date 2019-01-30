package macro

import (
	"bitbucket.org/aisee/minilog"
	"bitbucket.org/aisee/sc2lib"
	"bitbucket.org/aisee/veterran/bot"
	"bitbucket.org/aisee/veterran/micro"
	"github.com/chippydip/go-sc2ai/api"
	"github.com/chippydip/go-sc2ai/enums/ability"
	"github.com/chippydip/go-sc2ai/enums/effect"
	"github.com/chippydip/go-sc2ai/enums/protoss"
	"github.com/chippydip/go-sc2ai/enums/terran"
	"github.com/chippydip/go-sc2ai/enums/zerg"
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

const SafeBuildRange = 7

var B = bot.B
var LastBuildLoop int
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
		Limit:  func() int { return B.BuildPos[scl.S5x5].Len() },
		Active: BuildOne,
		WaitRes: func() bool {
			ccs := B.Units.My.OfType(scl.UnitAliases.For(terran.CommandCenter)...)
			// First orbital is morphing
			if ccs.Len() == 1 && ccs.First().UnitType == terran.OrbitalCommand &&
				B.PendingAliases(ability.Train_Reaper) != 0 {
				return true
			}
			if ccs.Len() <= B.FoodUsed/35 {
				return true
			}
			return false
		},
		Method: func() {
			pos := Build(ability.Build_CommandCenter)
			if pos != 0 && B.PlayDefensive {
				bot.FindBunkerPosition(pos)
			}
		},
	},
	{
		Name:    "Supplies",
		Ability: ability.Build_SupplyDepot,
		Premise: func() bool {
			// it's safe && 1 depo is placed && < 2:00 && only one cc
			if !B.PlayDefensive && B.FoodCap > 20 && B.Loop < 2688 &&
				B.Units.My.OfType(scl.UnitAliases.For(terran.CommandCenter)...).Len() == 1 {
				return false // Wait for a second cc
			}
			return B.FoodLeft < 6+B.FoodUsed/20 && B.FoodCap < 200
		},
		Limit:  func() int { return 30 },
		Active: func() int { return 1 + B.FoodUsed/50 },
	},
	{
		Name:    "Barrack",
		Ability: ability.Build_Barracks,
		Premise: func() bool {
			return B.Units.My.OfType(scl.UnitAliases.For(terran.SupplyDepot)...).First(scl.Ready) != nil &&
				B.Units.My.OfType(scl.UnitAliases.For(terran.Barracks)...).Empty()
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
				ccs := B.Units.My.OfType(scl.UnitAliases.For(terran.CommandCenter)...)
				if raxPending == 0 {
					return false
				}
				if B.Minerals > 350 {
					return true
				}
				if ccs.Len() < 3 {
					return refPending < ccs.Len()
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
			buildFacts := B.Units.My.OfType(scl.UnitAliases.For(terran.CommandCenter)...).Filter(scl.Ready).Len()
			if B.EnemyRace == api.Race_Zerg {
				buildFacts--
			}
			return scl.MinInt(4, buildFacts)
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
			ccs := B.Units.My.OfType(scl.UnitAliases.For(terran.CommandCenter)...)
			if ccs.Len() < 3 && B.Minerals < 500 {
				return 0
			}
			if B.Units.My[terran.FusionCore].First(scl.Ready) == nil {
				return 2
			}
			return scl.MinInt(4, ccs.Len())
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
			// B.Units.My[terran.Factory].First(scl.Ready) != nil // Needs factory
			return B.Units.My[terran.EngineeringBay].First(scl.Ready) != nil
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
			return B.EnemyRace != api.Race_Protoss &&
				B.Units.My[terran.Barracks].First(scl.Ready, scl.Unused) == nil &&
				B.Units.My[terran.BarracksFlying].Empty()
		},
		Limit: func() int {
			// ccs := B.Units.My.OfType(scl.UnitAliases.For(terran.CommandCenter)...)
			// orbitals := B.Units.My.OfType(terran.OrbitalCommand)
			return 2 // scl.MinInt(2, ccs.Len())
		},
		Active: BuildOne,
	},
	{
		Name:    "Engineering Bay",
		Ability: ability.Build_EngineeringBay,
		Premise: func() bool {
			return B.Units.My.OfType(scl.UnitAliases.For(terran.CommandCenter)...).Filter(scl.Ready).Len() >= 2
		},
		Limit:  BuildOne,
		Active: BuildOne,
	},
	{
		Name:    "Missile Turrets",
		Ability: ability.Build_MissileTurret,
		Premise: func() bool {
			//  && B.Units.My[terran.EngineeringBay].First(scl.Ready) != nil
			return B.BuildTurrets
		},
		Limit:   func() int { return B.TurretsPos.Len() },
		Active:  func() int { return B.TurretsPos.Len() },
		WaitRes: Yes,
	},
	{
		Name:    "Barracks reactors",
		Ability: ability.Build_Reactor_Barracks,
		Premise: func() bool {
			ccs := B.Units.My.OfType(scl.UnitAliases.For(terran.CommandCenter)...).Filter(scl.Ready)
			return ccs.Len() > 2 &&
				((B.Vespene >= 100 && B.Units.My[terran.Barracks].First(scl.Ready, scl.NoAddon, scl.Idle) != nil) ||
					B.Units.My[terran.BarracksFlying].First() != nil)
		},
		Limit:  BuildOne, // B.Units.My.OfType(scl.UnitAliases.For(terran.Barracks)...).Len()
		Active: BuildOne,
		Method: func() {
			// todo: group?
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
		Name:    "Barracks techlabs",
		Ability: ability.Build_TechLab_Barracks,
		Premise: func() bool {
			ccs := B.Units.My.OfType(scl.UnitAliases.For(terran.CommandCenter)...).Filter(scl.Ready)
			return ccs.Len() >= 2 && B.Units.My[terran.Barracks].First(scl.Ready, scl.NoAddon, scl.Idle) != nil
		},
		Limit: func() int {
			return B.Units.My.OfType(scl.UnitAliases.For(terran.Barracks)...).Len() - 1
		},
		Active: BuildOne,
		Method: func() {
			rax := B.Units.My[terran.Barracks].First(scl.Ready, scl.NoAddon, scl.Idle)
			if rax.IsCloserThan(3, B.FirstBarrack[0]) {
				return
			}
			rax.Command(ability.Build_TechLab_Barracks)
		},
	},
}

var FactoryBuildOrder = BuildNodes{
	{
		Name:    "Factory Tech Lab",
		Ability: ability.Build_TechLab_Factory,
		Premise: func() bool {
			return B.Units.My[terran.FactoryReactor].Exists() &&
				B.Units.My[terran.Factory].First(scl.Ready, scl.NoAddon, scl.Idle) != nil
		},
		Limit: func() int {
			return B.Units.My[terran.Factory].Len() - B.Units.My[terran.FactoryReactor].Len()
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
			return B.Units.My[terran.Factory].First(scl.Ready, scl.NoAddon, scl.Idle) != nil
		},
		Limit: func() int { // Build one but after tech lab
			return scl.MinInt(1, B.Units.My[terran.Factory].Len()-1)
		},
		Active: BuildOne,
		Method: func() {
			B.Units.My[terran.Factory].First(scl.Ready, scl.NoAddon, scl.Idle).Command(ability.Build_Reactor_Factory)
		},
	},
}

var StarportBuildOrder = BuildNodes{
	/*{
		Name:    "Starport Reactor",
		Ability: ability.Build_Reactor_Starport,
		Premise: func() bool {
			return B.Units.My[terran.Starport].First(scl.Ready, scl.NoAddon, scl.Idle) != nil &&
				B.PendingAliases(ability.Train_Medivac) > 0
		},
		Limit: BuildOne,
		Active: BuildOne,
		Method: func() {
			B.Units.My[terran.Starport].First(scl.Ready, scl.NoAddon, scl.Idle).Command(ability.Build_Reactor_Starport)
		},
	},*/
	{
		Name:    "Starport Tech Lab",
		Ability: ability.Build_TechLab_Starport,
		Premise: func() bool {
			return B.Units.My[terran.Starport].First(scl.Ready, scl.NoAddon, scl.Idle) != nil
		},
		Limit: func() int {
			return B.Units.My[terran.Starport].Len()
		},
		Active: BuildOne,
		Method: func() {
			B.Units.My[terran.Starport].First(scl.Ready, scl.NoAddon, scl.Idle).Command(ability.Build_TechLab_Starport)
		},
	},
	{
		Name:    "Fusion Core",
		Ability: ability.Build_FusionCore,
		Premise: func() bool {
			return B.Units.My[terran.Raven].Exists()
		},
		Limit:  BuildOne,
		Active: BuildOne,
	},
}

func OrderBuild(scv *scl.Unit, pos scl.Point, aid api.AbilityID) {
	scv.CommandPos(aid, pos)
	// scv.Orders = append(scv.Orders, &api.UnitOrder{AbilityId: aid}) // todo: move in commands
	B.DeductResources(aid)
	log.Debugf("%d: Building %v", B.Loop, scl.Types[scl.AbilityUnit[aid]].Name)
}

func OrderTrain(factory *scl.Unit, aid api.AbilityID) {
	factory.Command(aid)
	// factory.Orders = append(factory.Orders, &api.UnitOrder{AbilityId: aid}) // todo: move in commands
	B.DeductResources(aid)
	log.Debugf("%d: Training %v", B.Loop, scl.Types[scl.AbilityUnit[aid]].Name)
}

func Build(aid api.AbilityID) scl.Point {
	size, ok := BuildingsSizes[aid]
	if !ok {
		log.Alertf("Can't find size for %v", scl.Types[scl.AbilityUnit[aid]].Name)
		return 0
	}

	techReq := scl.Types[scl.AbilityUnit[aid]].TechRequirement
	if techReq != 0 && B.Units.My.OfType(scl.UnitAliases.For(techReq)...).Empty() {
		return 0 // Not available because of tech reqs, like: supply is needed for barracks
	}

	var buildersTargets scl.Points
	for _, builder := range B.Groups.Get(micro.Builders).Units {
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

		scv := bot.GetSCV(pos, micro.Builders, 45)
		if scv != nil {
			OrderBuild(scv, pos, aid)
			return pos
		}
		log.Debugf("%d: Failed to find SCV", B.Loop)
		return 0
	}
	log.Debugf("%d: Can't find position for %v", B.Loop, scl.Types[scl.AbilityUnit[aid]].Name)
	return 0
}

func BuildFirstBarrack() {
	pos := B.FirstBarrack[0]
	scv := B.Units.My[terran.SCV].ClosestTo(pos)
	if scv != nil {
		B.Groups.Add(micro.Builders, scv)
		OrderBuild(scv, pos, ability.Build_Barracks)
	}
}

func BuildRefinery(cc *scl.Unit) {
	// Find first geyser that is close to selected cc, but it doesn't have Refinery on top of it
	builders := B.Groups.Get(micro.Builders).Units
	geyser := B.Units.Geysers.All().CloserThan(10, cc).First(func(unit *scl.Unit) bool {
		return B.Units.My[terran.Refinery].CloserThan(1, unit).Len() == 0 &&
			unit.FindAssignedBuilder(builders) == nil
	})
	if geyser != nil {
		scv := bot.GetSCV(geyser, micro.Builders, 45)
		if scv != nil {
			scv.CommandTag(ability.Build_Refinery, geyser.Tag)
			B.DeductResources(ability.Build_Refinery)
			log.Debugf("%d: Building Refinery", B.Loop)
		}
	}
}

func ProcessBuildOrder(buildOrder BuildNodes) {
	for _, node := range buildOrder {
		inLimits := B.Pending(node.Ability) < node.Limit() && B.Orders[node.Ability] < node.Active()
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
		if node.Unlocks != nil && B.Units.My[scl.AbilityUnit[node.Ability]].Exists() {
			ProcessBuildOrder(node.Unlocks)
		}
	}
}

func OrderUpgrades() {
	lab := B.Units.My[terran.BarracksTechLab].First(scl.Ready, scl.Idle)
	if lab != nil {
		B.RequestAvailableAbilities(true, lab) // todo: request true each frame -> HasTrueAbility?
		if !B.Upgrades[ability.Research_ConcussiveShells] && B.PendingAliases(ability.Train_Marauder) >= 2 &&
			lab.HasAbility(ability.Research_ConcussiveShells) && B.CanBuy(ability.Research_ConcussiveShells) {
			lab.Command(ability.Research_ConcussiveShells)
			return
		}
		if !B.Upgrades[ability.Research_CombatShield] && B.Units.My[terran.Marine].Len() >= 4 &&
			lab.HasAbility(ability.Research_CombatShield) && B.CanBuy(ability.Research_CombatShield) {
			lab.Command(ability.Research_CombatShield)
			return
		}
		if (B.Upgrades[ability.Research_ConcussiveShells] || B.PendingAliases(ability.Research_ConcussiveShells) > 0 ||
			B.Upgrades[ability.Research_CombatShield] || B.PendingAliases(ability.Research_CombatShield) > 0) &&
			!B.Upgrades[ability.Research_Stimpack] && lab.HasAbility(ability.Research_Stimpack) &&
			B.CanBuy(ability.Research_Stimpack) {
			lab.Command(ability.Research_Stimpack)
			return
		}
	}

	eng := B.Units.My[terran.EngineeringBay].First(scl.Ready, scl.Idle)
	if eng != nil {
		B.RequestAvailableAbilities(true, eng) // request abilities again because we want to ignore resource reqs
		if B.Units.My[terran.Marine].Len()+B.Units.My[terran.Marauder].Len()*2+B.Units.My[terran.Reaper].Len()*2 >= 8 {
			for _, a := range []api.AbilityID{
				ability.Research_TerranInfantryWeaponsLevel1,
				ability.Research_TerranInfantryArmorLevel1,
				ability.Research_TerranInfantryWeaponsLevel2,
				ability.Research_TerranInfantryArmorLevel2,
				ability.Research_TerranInfantryWeaponsLevel3,
				ability.Research_TerranInfantryArmorLevel3,
			} {
				if B.Upgrades[a] {
					continue
				}
				if eng.HasAbility(a) {
					if B.CanBuy(a) {
						eng.Command(a)
						return
					} else {
						// reserve money for upgrade
						B.DeductResources(a)
					}
					break
				}
			}
		}
		if !B.Upgrades[ability.Research_HiSecAutoTracking] && B.Units.AllEnemy[terran.Banshee].Exists() &&
			eng.HasAbility(ability.Research_HiSecAutoTracking) && B.CanBuy(ability.Research_HiSecAutoTracking) {
			eng.Command(ability.Research_HiSecAutoTracking)
			return
		}
	}

	// todo: aliases
	if arm := B.Units.My[terran.Armory].First(scl.Ready, scl.Idle); arm != nil && B.Units.My.OfType(terran.WidowMine,
		terran.Hellion, terran.Cyclone, terran.SiegeTank, terran.Raven, terran.Battlecruiser).Len() > 4 {
		B.RequestAvailableAbilities(true, arm) // request abilities again because we want to ignore resource reqs
		upgrades := []api.AbilityID{
			ability.Research_TerranVehicleAndShipPlatingLevel1,
			ability.Research_TerranVehicleAndShipPlatingLevel2,
			ability.Research_TerranVehicleAndShipPlatingLevel3,
			ability.Research_TerranVehicleWeaponsLevel1,
			ability.Research_TerranVehicleWeaponsLevel2,
			ability.Research_TerranVehicleWeaponsLevel3,
		}
		if B.Units.My[terran.Battlecruiser].Exists() {
			upgrades = append([]api.AbilityID{
				ability.Research_TerranShipWeaponsLevel1,
				ability.Research_TerranShipWeaponsLevel2,
				ability.Research_TerranShipWeaponsLevel3,
			}, upgrades...)
		}
		for _, a := range upgrades {
			if B.Upgrades[a] {
				continue
			}
			if arm.HasAbility(a) {
				if B.CanBuy(a) {
					arm.Command(a)
					return
				} else {
					// reserve money for upgrade
					B.DeductResources(a)
				}
				break
			}
		}
	}

	lab = B.Units.My[terran.FactoryTechLab].First(scl.Ready, scl.Idle)
	if lab != nil && (B.Units.My[terran.Cyclone].Exists() || B.Units.My[terran.WidowMine].Exists()) {
		B.RequestAvailableAbilities(true, lab)
		if B.PendingAliases(ability.Train_Cyclone) >= 2 &&
			lab.HasAbility(ability.Research_CycloneResearchLockOnDamageUpgrade) &&
			B.CanBuy(ability.Research_CycloneResearchLockOnDamageUpgrade) {
			lab.Command(ability.Research_CycloneResearchLockOnDamageUpgrade)
			return
		}
		if B.PendingAliases(ability.Train_WidowMine) >= 2 && lab.HasAbility(ability.Research_DrillingClaws) &&
			B.CanBuy(ability.Research_DrillingClaws) {
			lab.Command(ability.Research_DrillingClaws)
			return
		}
		if B.PendingAliases(ability.Train_Hellion) >= 4 && lab.HasAbility(ability.Research_InfernalPreigniter) &&
			B.CanBuy(ability.Research_InfernalPreigniter) {
			lab.Command(ability.Research_InfernalPreigniter)
			return
		}
	}

	fc := B.Units.My[terran.FusionCore].First(scl.Ready, scl.Idle)
	if fc != nil && B.Pending(ability.Train_Battlecruiser) > 0 &&
		!B.Upgrades[ability.Research_BattlecruiserWeaponRefit] {
		B.RequestAvailableAbilities(true, fc)
		if fc.HasAbility(ability.Research_BattlecruiserWeaponRefit) &&
			B.CanBuy(ability.Research_BattlecruiserWeaponRefit) {
			fc.Command(ability.Research_BattlecruiserWeaponRefit)
			return
		}
	}
}

func Morph() {
	cc := B.Units.My[terran.CommandCenter].First(scl.Ready, scl.Idle)
	if cc != nil && B.Units.My[terran.Barracks].First(scl.Ready) != nil {
		if B.CanBuy(ability.Morph_OrbitalCommand) {
			OrderTrain(cc, ability.Morph_OrbitalCommand)
		} else if B.Units.My[terran.SCV].Len() >= 16 {
			B.DeductResources(ability.Morph_OrbitalCommand)
		}
	}
	groundEnemies := B.Enemies.All.Filter(scl.NotFlying)
	for _, supply := range B.Units.My[terran.SupplyDepot] {
		if groundEnemies.CloserThan(4, supply).Empty() {
			supply.Command(ability.Morph_SupplyDepot_Lower)
		}
	}
	for _, supply := range B.Units.My[terran.SupplyDepotLowered] {
		if groundEnemies.CloserThan(4, supply).Exists() {
			supply.Command(ability.Morph_SupplyDepot_Raise)
		}
	}
}

func Cast() {
	cc := B.Units.My[terran.OrbitalCommand].
		Filter(func(unit *scl.Unit) bool { return unit.Energy >= 50 }).
		Max(func(unit *scl.Unit) float64 { return float64(unit.Energy) })
	if cc != nil {
		// Scan
		if B.Orders[ability.Effect_Scan] == 0 && B.EffectPoints(effect.ScannerSweep).Empty() {
			allEnemies := B.Enemies.All
			visibleEnemies := allEnemies.Filter(scl.PosVisible)
			units := B.Units.My.All()
			// Reaper wants to see highground
			if B.Units.My[terran.Raven].Empty() {
				if reaper := B.Groups.Get(micro.Reapers).Units.ClosestTo(B.Locs.EnemyStart); reaper != nil {
					if enemy := allEnemies.CanAttack(reaper, 1).ClosestTo(reaper); enemy != nil {
						if !B.IsVisible(enemy) && B.HeightAt(enemy) > B.HeightAt(reaper) {
							pos := enemy.Towards(B.Locs.EnemyStart, 8)
							cc.CommandPos(ability.Effect_Scan, pos)
							log.Debug("Reaper sight scan")
							return
						}
					}
				}
			}

			// Vision for tanks
			tanks := B.Units.My[terran.SiegeTankSieged]
			tanks.OrderByDistanceTo(B.Locs.EnemyStart, false)
			for _, tank := range tanks {
				targets := allEnemies.InRangeOf(tank, 0)
				if targets.Exists() && visibleEnemies.InRangeOf(tank, 0).Empty() {
					target := targets.ClosestTo(B.Locs.EnemyStart)
					cc.CommandPos(ability.Effect_Scan, target)
					log.Debug("Tank sight scan")
				}
			}

			// Lurkers
			if eps := B.EffectPoints(effect.LurkerSpines); eps.Exists() {
				// todo: check if bot already sees the lurker using his position approximation
				cc.CommandPos(ability.Effect_Scan, eps.ClosestTo(B.Locs.EnemyStart))
				log.Debug("Lurker scan")
				return
			}

			// DTs
			if B.EnemyRace == api.Race_Protoss {
				dts := B.Units.Enemy[protoss.DarkTemplar]
				hitByDT := units.First(func(unit *scl.Unit) bool {
					return unit.HitsLost >= 41 && !unit.IsArmored() && !dts.CanAttack(unit, 0).Exists()
				})
				if hitByDT != nil {
					cc.CommandPos(ability.Effect_Scan, hitByDT)
					log.Debug("DT scan")
					return
				}
			}

			// Early banshee without upgrades
			if B.EnemyRace == api.Race_Terran {
				for _, u := range units {
					if u.HitsLost == 12 && allEnemies.CanAttack(u, 2).Empty() {
						cc.CommandPos(ability.Effect_Scan, u)
						log.Debug("Banshee scan")
						return
					}
				}
			}

			// Recon scan at 4:00
			pos := B.Locs.EnemyMainCenter
			if B.EnemyRace == api.Race_Zerg {
				pos = B.Locs.EnemyStart
			}
			if B.Loop >= 5376 && !B.IsExplored(pos) {
				cc.CommandPos(ability.Effect_Scan, pos)
				log.Debug("Recon scan")
				return
			}
		}
		// Mule
		if cc.Energy >= 75 || (B.Loop < 4928 && cc.Energy >= 50) { // 3:40
			ccs := B.Units.My.OfType(terran.CommandCenter, terran.OrbitalCommand,
				terran.PlanetaryFortress).Filter(scl.Ready)
			ccs.OrderByDistanceTo(cc, false)
			for _, target := range ccs {
				homeMineral := B.Units.Minerals.All().CloserThan(scl.ResourceSpreadDistance, target).
					Filter(func(unit *scl.Unit) bool { return unit.MineralContents > 400 }).
					Max(func(unit *scl.Unit) float64 { return float64(unit.MineralContents) })
				if homeMineral != nil {
					cc.CommandTag(ability.Effect_CalldownMULE, homeMineral.Tag)
				}
			}
		}
	}
}

func OrderUnits() {
	mech := false
	if B.EnemyRace != api.Race_Zerg {
		mech = true
	}
	if B.WorkerRush && B.CanBuy(ability.Train_Marine) {
		if rax := B.Units.My[terran.Barracks].First(scl.Ready, scl.Unused); rax != nil {
			if rax.HasReactor() && scl.UnitsOrders[rax.Tag].Loop+B.FramesPerOrder <= B.Loop {
				rax.SpamCmds = true
			}
			OrderTrain(rax, ability.Train_Marine)
		}
	}

	ccs := B.Units.My.OfType(terran.CommandCenter, terran.OrbitalCommand, terran.PlanetaryFortress)
	cc := ccs.First(scl.Ready, scl.Idle)
	refs := B.Units.My[terran.Refinery].Filter(func(unit *scl.Unit) bool {
		return unit.IsReady() && unit.VespeneContents > 0
	})
	if cc != nil && B.Units.My[terran.SCV].Len() < scl.MinInt(21*ccs.Len(), 70-refs.Len()) &&
		B.CanBuy(ability.Train_SCV) && !B.WorkerRush {
		OrderTrain(cc, ability.Train_SCV)
	}

	starport := B.Units.My[terran.Starport].First(scl.Ready, scl.Unused, scl.HasTechlab)
	if starport != nil {
		ravens := B.Pending(ability.Train_Raven)
		if B.Units.My[terran.FusionCore].First(scl.Ready) != nil {
			if B.CanBuy(ability.Train_Battlecruiser) {
				OrderTrain(starport, ability.Train_Battlecruiser)
			} else {
				B.DeductResources(ability.Train_Battlecruiser) // Gather money
			}
		}
		if ravens < 2 {
			if B.CanBuy(ability.Train_Raven) {
				OrderTrain(starport, ability.Train_Raven)
			} else if ravens == 0 {
				B.DeductResources(ability.Train_Raven) // Gather money
			}
		}
	}
	starport = B.Units.My[terran.Starport].First(scl.Ready, scl.Unused, func(unit *scl.Unit) bool {
		return starport == nil || unit.Tag != starport.Tag // Don't select previously selected producer
	})
	if starport != nil {
		if starport.HasReactor() && scl.UnitsOrders[starport.Tag].Loop+B.FramesPerOrder <= B.Loop {
			starport.SpamCmds = true
		}
		medivacs := B.Pending(ability.Train_Medivac)
		infantry := B.Units.My[terran.Marine].Len() + B.Units.My[terran.Marauder].Len()*2
		if (medivacs == 0 || medivacs*8 < infantry) && B.CanBuy(ability.Train_Medivac) {
			OrderTrain(starport, ability.Train_Medivac)
		} else if medivacs == 0 {
			B.DeductResources(ability.Train_Medivac) // Gather money
		}
	}

	factory := B.Units.My[terran.Factory].First(scl.Ready, scl.Unused, scl.HasTechlab)
	if factory != nil {
		cyclones := B.PendingAliases(ability.Train_Cyclone)
		tanks := B.PendingAliases(ability.Train_SiegeTank)

		buyCyclones := B.EnemyProduction.Len(terran.Banshee) > 0 && cyclones == 0
		buyTanks := B.PlayDefensive && tanks == 0
		if !buyCyclones && !buyTanks {
			cyclonesScore := B.EnemyProduction.Score(protoss.WarpPrism, protoss.Phoenix, protoss.VoidRay,
				protoss.Oracle, protoss.Tempest, protoss.Carrier, terran.VikingFighter, terran.Medivac,
				terran.Liberator, terran.Raven, terran.Banshee, terran.Battlecruiser, zerg.Queen, zerg.Mutalisk,
				zerg.Corruptor, zerg.Viper, zerg.Ultralisk, zerg.BroodLord) + 1
			tanksScore := B.EnemyProduction.Score(protoss.Stalker, protoss.Colossus, protoss.PhotonCannon,
				terran.Marine, terran.Reaper, terran.Marauder, terran.Bunker, /*zerg.Zergling, zerg.Baneling,*/
				zerg.Roach, zerg.Ravager, zerg.Hydralisk, zerg.LurkerMP, zerg.SpineCrawler) + 1
			buyCyclones = cyclonesScore/float64(cyclones+1) >= tanksScore/float64(tanks+1)
			buyTanks = !buyCyclones
		}

		if buyCyclones {
			if B.CanBuy(ability.Train_Cyclone) {
				OrderTrain(factory, ability.Train_Cyclone)
			} else if cyclones == 0 || mech {
				B.DeductResources(ability.Train_Cyclone) // Gather money
			}
		} else if buyTanks {
			if B.CanBuy(ability.Train_SiegeTank) {
				OrderTrain(factory, ability.Train_SiegeTank)
			} else if tanks == 0 || mech {
				B.DeductResources(ability.Train_SiegeTank) // Gather money
			}
		}
	}

	factory = B.Units.My[terran.Factory].First(func(unit *scl.Unit) bool {
		return unit.IsReady() && unit.IsUnused() && !unit.HasTechlab() && (factory == nil || unit.Tag != factory.Tag)
	})
	if factory != nil {
		if factory.HasReactor() && scl.UnitsOrders[factory.Tag].Loop+B.FramesPerOrder <= B.Loop {
			// I need to pass this param because else duplicate order will be ignored
			// But I need to be sure that there was no previous order recently
			factory.SpamCmds = true
		}
		mines := B.PendingAliases(ability.Train_WidowMine)
		hellions := B.PendingAliases(ability.Train_Hellion)

		minesScore := B.EnemyProduction.Score(protoss.Stalker, protoss.Archon, protoss.Phoenix, protoss.VoidRay,
			protoss.Oracle, protoss.Tempest, protoss.Carrier, terran.Cyclone, terran.SiegeTank, terran.Thor,
			terran.VikingFighter, terran.Medivac, terran.Liberator, terran.Raven, terran.Banshee,
			terran.Battlecruiser, zerg.Hydralisk, zerg.Queen, zerg.Roach, zerg.Ravager, zerg.Mutalisk, zerg.Corruptor,
			zerg.Viper, zerg.Ultralisk, zerg.BroodLord) + 1
		hellionsScore := B.EnemyProduction.Score(zerg.Zergling, zerg.Baneling, zerg.SwarmHostMP) + 1
		buyMines := minesScore/float64(mines+1) >= hellionsScore/float64(hellions+1)

		if buyMines {
			if B.CanBuy(ability.Train_WidowMine) {
				OrderTrain(factory, ability.Train_WidowMine)
			} else if mines == 0 || mech {
				B.DeductResources(ability.Train_WidowMine) // Gather money
			}
		} else {
			if B.CanBuy(ability.Train_Hellion) {
				OrderTrain(factory, ability.Train_Hellion)
			} else if hellions == 0 || mech {
				B.DeductResources(ability.Train_Hellion) // Gather money
			}
		}
	}

	rax := B.Units.My[terran.Barracks].First(scl.Ready, scl.Unused, scl.HasTechlab)
	if rax != nil {
		marines := B.PendingAliases(ability.Train_Marine)
		marauders := B.PendingAliases(ability.Train_Marauder)
		marinesScore := B.EnemyProduction.Score(protoss.Immortal, protoss.WarpPrism, protoss.Phoenix, protoss.VoidRay,
			protoss.Oracle, protoss.Tempest, protoss.Carrier, terran.VikingFighter, terran.Medivac, terran.Liberator,
			terran.Raven, terran.Banshee, terran.Battlecruiser, zerg.Mutalisk, zerg.Corruptor, zerg.Viper,
			zerg.BroodLord) + 1 //  zerg.Zergling,
		maraudersScore := B.EnemyProduction.Score(protoss.Zealot, protoss.Stalker, protoss.Adept, terran.Reaper,
			terran.Hellion, terran.WidowMine, terran.Cyclone, terran.Thor, zerg.Baneling, zerg.Roach, zerg.Ravager,
			zerg.Ultralisk) + 1
		buyMarauders := marinesScore/float64(marines+1) < maraudersScore/float64(marauders+1)

		if buyMarauders {
			if B.CanBuy(ability.Train_Marauder) {
				OrderTrain(rax, ability.Train_Marauder)
			} else {
				B.DeductResources(ability.Train_Marauder) // Gather money
			}
		}
	}
	rax = B.Units.My[terran.Barracks].First(scl.Ready, scl.Unused, func(unit *scl.Unit) bool {
		return rax == nil || unit.Tag != rax.Tag // Don't select previously selected producer
	})
	if rax != nil {
		if rax.HasReactor() && scl.UnitsOrders[rax.Tag].Loop+B.FramesPerOrder <= B.Loop {
			rax.SpamCmds = true
		}
		// Until 4:00
		// B.Loop < 5376 && (B.Pending(ability.Train_Reaper) < 2 || B.EnemyRace == api.Race_Zerg) &&
		// before 2:40 or if they are not dying until 4:00
		if !B.LingRush && (B.Loop < 3584 || (B.Loop < 5376 && B.Pending(ability.Train_Reaper) > B.Loop/1344)) &&
			B.CanBuy(ability.Train_Reaper) {
			OrderTrain(rax, ability.Train_Reaper)
		} else if /*B.Loop >= 2688 &&*/ B.CanBuy(ability.Train_Marine) { // 2:00
			OrderTrain(rax, ability.Train_Marine)
		}
	}
}

func ReserveSCVs() {
	// Fast first supply
	if B.Units.My.OfType(scl.UnitAliases.For(terran.SupplyDepot)...).Empty() &&
		B.Groups.Get(micro.ScvReserve).Tags.Empty() {
		pos := B.BuildPos[scl.S2x2][0]
		scv := bot.GetSCV(pos, 0, 45) // Get SCV but don't change its group
		if scv != nil && scv.FramesToPos(pos)*B.MineralsPerFrame+float64(B.Minerals)+20 >= 100 {
			B.Groups.Add(micro.ScvReserve, scv)
			scv.CommandPos(ability.Move, pos)
		}
	}
	// Fast expansion
	if B.Units.My.OfType(terran.CommandCenter, terran.OrbitalCommand, terran.PlanetaryFortress).Len() == 1 &&
		B.Minerals >= 350 && B.Groups.Get(micro.ScvReserve).Tags.Empty() /*&& !PlayDefensive*/ && !B.WorkerRush {
		pos := B.Locs.MyExps[0]
		if scv := bot.GetSCV(pos, micro.ScvReserve, 45); scv != nil {
			scv.CommandPos(ability.Move, pos)
		}
	}
}

func Macro() {
	if !B.BuildTurrets && B.Units.Enemy.OfType(terran.Banshee, terran.Ghost, terran.WidowMine, terran.Medivac,
		terran.VikingFighter, terran.Liberator, terran.Battlecruiser, terran.Starport, zerg.Mutalisk, zerg.LurkerMP,
		zerg.Corruptor, zerg.Spire, zerg.GreaterSpire, protoss.DarkTemplar, protoss.WarpPrism, protoss.Phoenix,
		protoss.VoidRay, protoss.Oracle, protoss.Tempest, protoss.Carrier, protoss.Stargate, protoss.DarkShrine).
		Exists() {
		B.BuildTurrets = true
	}
	if B.FindTurretPosFor != nil {
		bot.FindTurretPosition(B.FindTurretPosFor)
		B.FindTurretPosFor = nil
	}

	if LastBuildLoop+B.FramesPerOrder < B.Loop {
		if B.Loop >= 5376 { // 4:00
			OrderUpgrades()
		}
		ProcessBuildOrder(RootBuildOrder)
		Morph()
		OrderUnits()
		ReserveSCVs()
		LastBuildLoop = B.Loop
	}
	Cast()
}
