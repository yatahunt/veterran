package macro

import (
	"bitbucket.org/aisee/minilog"
	"bitbucket.org/aisee/sc2lib"
	"github.com/chippydip/go-sc2ai/api"
	"github.com/chippydip/go-sc2ai/enums/ability"
	"github.com/chippydip/go-sc2ai/enums/effect"
	"github.com/chippydip/go-sc2ai/enums/protoss"
	"github.com/chippydip/go-sc2ai/enums/terran"
	"github.com/chippydip/go-sc2ai/enums/zerg"
	"math"
)

type Booler func(b *bot) bool
type Inter func(b *bot) int
type Voider func(b *bot)
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

func BuildOne(b *bot) int { return 1 }
func Yes(b *bot) bool     { return true }

var BuildTurrets = false
var LastBuildLoop = 0

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
		Premise: func(b *bot) bool {
			return b.Units.AllEnemy.All().Filter(scl.DpsGt5).CloserThan(DefensiveRange, b.Locs.MyStart).Empty()
		},
		Limit:  func(b *bot) int { return BuildPos[scl.S5x5].Len() },
		Active: BuildOne,
		WaitRes: func(b *bot) bool {
			ccs := b.Units.My.OfType(scl.UnitAliases.For(terran.CommandCenter)...)
			// First orbital is morphing
			if ccs.Len() == 1 && ccs.First().UnitType == terran.OrbitalCommand &&
				b.PendingAliases(ability.Train_Reaper) != 0 {
				return true
			}
			if ccs.Len() <= b.FoodUsed/35 {
				return true
			}
			return false
		},
		Method: func(b *bot) {
			pos := b.Build(ability.Build_CommandCenter)
			if pos != 0 && PlayDefensive {
				b.FindBunkerPosition(pos)
			}
		},
	},
	{
		Name:    "Supplies",
		Ability: ability.Build_SupplyDepot,
		Premise: func(b *bot) bool {
			// it's safe && 1 depo is placed && < 2:00 && only one cc
			if !PlayDefensive && b.FoodCap > 20 && b.Loop < 2688 &&
				b.Units.My.OfType(scl.UnitAliases.For(terran.CommandCenter)...).Len() == 1 {
				return false // Wait for a second cc
			}
			return b.FoodLeft < 6+b.FoodUsed/20 && b.FoodCap < 200
		},
		Limit:  func(b *bot) int { return 30 },
		Active: func(b *bot) int { return 1 + b.FoodUsed/50 },
	},
	{
		Name:    "Barrack",
		Ability: ability.Build_Barracks,
		Premise: func(b *bot) bool {
			return b.Units.My.OfType(scl.UnitAliases.For(terran.SupplyDepot)...).First(scl.Ready) != nil &&
				b.Units.My.OfType(scl.UnitAliases.For(terran.Barracks)...).Empty()
		},
		Limit:  BuildOne,
		Active: BuildOne,
		Method: func(b *bot) { b.BuildFirstBarrack() },
	},
	{
		Name:    "Refinery",
		Ability: ability.Build_Refinery,
		Premise: func(b *bot) bool {
			if WorkerRush {
				return false
			}
			if b.Vespene < b.Minerals*2 {
				raxPending := b.Pending(ability.Build_Barracks)
				refPending := b.Pending(ability.Build_Refinery)
				ccs := b.Units.My.OfType(scl.UnitAliases.For(terran.CommandCenter)...)
				if raxPending == 0 {
					return false
				}
				if b.Minerals > 350 {
					return true
				}
				if ccs.Len() < 3 {
					return refPending < ccs.Len()
				}
				return true
			}
			return false
		},
		Limit:  func(b *bot) int { return 20 },
		Active: func(b *bot) int { return 2 },
		Method: func(b *bot) {
			ccs := b.Units.My.OfType(terran.CommandCenter, terran.OrbitalCommand, terran.PlanetaryFortress)
			if cc := ccs.First(scl.Ready); cc != nil {
				b.BuildRefinery(cc)
			}
		},
		Unlocks: RaxBuildOrder,
	},
	{
		Name:    "Factory",
		Ability: ability.Build_Factory,
		Premise: func(b *bot) bool {
			return b.Units.My[terran.Factory].First(scl.Ready, scl.Unused) == nil
		},
		Limit: func(b *bot) int {
			buildFacts := b.Units.My.OfType(scl.UnitAliases.For(terran.CommandCenter)...).Filter(scl.Ready).Len()
			if b.EnemyRace == api.Race_Zerg {
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
		Premise: func(b *bot) bool {
			return b.Units.My[terran.Starport].First(scl.Ready, scl.Unused) == nil
		},
		Limit: func(b *bot) int {
			ccs := b.Units.My.OfType(scl.UnitAliases.For(terran.CommandCenter)...)
			if ccs.Len() < 3 && b.Minerals < 500 {
				return 0
			}
			if b.Units.My[terran.FusionCore].First(scl.Ready) == nil {
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
		Premise: func(b *bot) bool {
			return b.Units.My.OfType(terran.Marine, terran.Reaper).Len() >= 2 &&
				b.Units.AllEnemy.All().Filter(scl.DpsGt5).CloserThan(DefensiveRange, b.Locs.MyStart).Empty()
		},
		Limit:   func(b *bot) int { return bunkersPos.Len() },
		Active:  func(b *bot) int { return bunkersPos.Len() },
		WaitRes: Yes,
	},
	{
		Name:    "Armory",
		Ability: ability.Build_Armory,
		Premise: func(b *bot) bool {
			// b.Units.My[terran.Factory].First(scl.Ready) != nil // Needs factory
			return b.Units.My[terran.EngineeringBay].First(scl.Ready) != nil
		},
		WaitRes: Yes,
		Limit: func(b *bot) int {
			if b.Units.My[terran.FusionCore].First(scl.Ready) != nil {
				return 2
			}
			return 1
		},
		Active: BuildOne,
	},
	{
		Name:    "Barracks",
		Ability: ability.Build_Barracks,
		Premise: func(b *bot) bool {
			return b.EnemyRace != api.Race_Protoss &&
				b.Units.My[terran.Barracks].First(scl.Ready, scl.Unused) == nil &&
				b.Units.My[terran.BarracksFlying].Empty()
		},
		Limit: func(b *bot) int {
			// ccs := b.Units.My.OfType(scl.UnitAliases.For(terran.CommandCenter)...)
			// orbitals := b.Units.My.OfType(terran.OrbitalCommand)
			return 2 // scl.MinInt(2, ccs.Len())
		},
		Active: BuildOne,
	},
	{
		Name:    "Engineering Bay",
		Ability: ability.Build_EngineeringBay,
		Premise: func(b *bot) bool {
			return b.Units.My.OfType(scl.UnitAliases.For(terran.CommandCenter)...).Filter(scl.Ready).Len() >= 2
		},
		Limit:  BuildOne,
		Active: BuildOne,
	},
	{
		Name:    "Missile Turrets",
		Ability: ability.Build_MissileTurret,
		Premise: func(b *bot) bool {
			//  && b.Units.My[terran.EngineeringBay].First(scl.Ready) != nil
			return BuildTurrets
		},
		Limit:   func(b *bot) int { return turretsPos.Len() },
		Active:  func(b *bot) int { return turretsPos.Len() },
		WaitRes: Yes,
	},
	{
		Name:    "Barracks reactors",
		Ability: ability.Build_Reactor_Barracks,
		Premise: func(b *bot) bool {
			ccs := b.Units.My.OfType(scl.UnitAliases.For(terran.CommandCenter)...).Filter(scl.Ready)
			return ccs.Len() > 2 &&
				((b.Vespene >= 100 && b.Units.My[terran.Barracks].First(scl.Ready, scl.NoAddon, scl.Idle) != nil) ||
					b.Units.My[terran.BarracksFlying].First() != nil)
		},
		Limit:  BuildOne, // b.Units.My.OfType(scl.UnitAliases.For(terran.Barracks)...).Len()
		Active: BuildOne,
		Method: func(b *bot) {
			// todo: group?
			if rax := b.Units.My[terran.BarracksFlying].First(); rax != nil {
				rax.CommandPos(ability.Build_Reactor_Barracks, FirstBarrackBuildPos[1])
				return
			}

			rax := b.Units.My[terran.Barracks].First(scl.Ready, scl.NoAddon, scl.Idle)
			if rax.IsCloserThan(3, FirstBarrackBuildPos[0]) {
				if FirstBarrackBuildPos[0] != FirstBarrackBuildPos[1] {
					if b.Units.Enemy.All().CloserThan(safeBuildRange, rax).Exists() {
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
		Premise: func(b *bot) bool {
			ccs := b.Units.My.OfType(scl.UnitAliases.For(terran.CommandCenter)...).Filter(scl.Ready)
			return ccs.Len() >= 2 && b.Units.My[terran.Barracks].First(scl.Ready, scl.NoAddon, scl.Idle) != nil
		},
		Limit: func(b *bot) int {
			return b.Units.My.OfType(scl.UnitAliases.For(terran.Barracks)...).Len() - 1
		},
		Active: BuildOne,
		Method: func(b *bot) {
			rax := b.Units.My[terran.Barracks].First(scl.Ready, scl.NoAddon, scl.Idle)
			if rax.IsCloserThan(3, FirstBarrackBuildPos[0]) {
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
		Premise: func(b *bot) bool {
			return b.Units.My[terran.FactoryReactor].Exists() &&
				b.Units.My[terran.Factory].First(scl.Ready, scl.NoAddon, scl.Idle) != nil
		},
		Limit: func(b *bot) int {
			return b.Units.My[terran.Factory].Len() - b.Units.My[terran.FactoryReactor].Len()
		},
		Active: BuildOne,
		Method: func(b *bot) {
			b.Units.My[terran.Factory].First(scl.Ready, scl.NoAddon, scl.Idle).Command(ability.Build_TechLab_Factory)
		},
	},
	{
		Name:    "Factory Reactor",
		Ability: ability.Build_Reactor_Factory,
		Premise: func(b *bot) bool {
			return b.Units.My[terran.Factory].First(scl.Ready, scl.NoAddon, scl.Idle) != nil
		},
		Limit: func(b *bot) int { // Build one but after tech lab
			return scl.MinInt(1, b.Units.My[terran.Factory].Len()-1)
		},
		Active: BuildOne,
		Method: func(b *bot) {
			b.Units.My[terran.Factory].First(scl.Ready, scl.NoAddon, scl.Idle).Command(ability.Build_Reactor_Factory)
		},
	},
}

var StarportBuildOrder = BuildNodes{
	/*{
		Name:    "Starport Reactor",
		Ability: ability.Build_Reactor_Starport,
		Premise: func(b *bot) bool {
			return b.Units.My[terran.Starport].First(scl.Ready, scl.NoAddon, scl.Idle) != nil &&
				b.PendingAliases(ability.Train_Medivac) > 0
		},
		Limit: BuildOne,
		Active: BuildOne,
		Method: func(b *bot) {
			b.Units.My[terran.Starport].First(scl.Ready, scl.NoAddon, scl.Idle).Command(ability.Build_Reactor_Starport)
		},
	},*/
	{
		Name:    "Starport Tech Lab",
		Ability: ability.Build_TechLab_Starport,
		Premise: func(b *bot) bool {
			return b.Units.My[terran.Starport].First(scl.Ready, scl.NoAddon, scl.Idle) != nil
		},
		Limit: func(b *bot) int {
			return b.Units.My[terran.Starport].Len()
		},
		Active: BuildOne,
		Method: func(b *bot) {
			b.Units.My[terran.Starport].First(scl.Ready, scl.NoAddon, scl.Idle).Command(ability.Build_TechLab_Starport)
		},
	},
	{
		Name:    "Fusion Core",
		Ability: ability.Build_FusionCore,
		Premise: func(b *bot) bool {
			return b.Units.My[terran.Raven].Exists()
		},
		Limit:  BuildOne,
		Active: BuildOne,
	},
}

func (b *bot) OrderBuild(scv *scl.Unit, pos scl.Point, aid api.AbilityID) {
	scv.CommandPos(aid, pos)
	// scv.Orders = append(scv.Orders, &api.UnitOrder{AbilityId: aid}) // todo: move in commands
	b.DeductResources(aid)
	log.Debugf("%d: Building %v", b.Loop, scl.Types[scl.AbilityUnit[aid]].Name)
}

func (b *bot) OrderTrain(factory *scl.Unit, aid api.AbilityID) {
	factory.Command(aid)
	// factory.Orders = append(factory.Orders, &api.UnitOrder{AbilityId: aid}) // todo: move in commands
	b.DeductResources(aid)
	log.Debugf("%d: Training %v", b.Loop, scl.Types[scl.AbilityUnit[aid]].Name)
}

func (b *bot) Build(aid api.AbilityID) scl.Point {
	size, ok := BuildingsSizes[aid]
	if !ok {
		log.Alertf("Can't find size for %v", scl.Types[scl.AbilityUnit[aid]].Name)
		return 0
	}

	techReq := scl.Types[scl.AbilityUnit[aid]].TechRequirement
	if techReq != 0 && b.Units.My.OfType(scl.UnitAliases.For(techReq)...).Empty() {
		return 0 // Not available because of tech reqs, like: supply is needed for barracks
	}

	var buildersTargets scl.Points
	for _, builder := range b.Groups.Get(Builders).Units {
		buildersTargets.Add(builder.TargetPos())
	}

	enemies := b.Units.AllEnemy.All().Filter(scl.DpsGt5)
	positions := BuildPos[size]
	if size == scl.S3x3 {
		// Add larger building positions if there is not enough S3x3 positions
		positions = append(positions, BuildPos[scl.S5x3]...)
	}
	if aid == ability.Build_MissileTurret {
		positions = turretsPos
	}
	if aid == ability.Build_Bunker {
		positions = bunkersPos
	}
	for _, pos := range positions {
		if buildersTargets.CloserThan(math.Sqrt2, pos).Exists() {
			continue // Someone already constructing there
		}
		if !b.IsPosOk(pos, size, 0, scl.IsBuildable, scl.IsNoCreep) {
			continue
		}
		if enemies.CloserThan(safeBuildRange, pos).Exists() {
			continue
		}
		if PlayDefensive && aid == ability.Build_CommandCenter && pos.IsFurtherThan(DefensiveRange, b.Locs.MyStart) {
			continue
		}

		scv := b.GetSCV(pos, Builders, 45)
		if scv != nil {
			b.OrderBuild(scv, pos, aid)
			return pos
		}
		log.Debugf("%d: Failed to find SCV", b.Loop)
		return 0
	}
	log.Debugf("%d: Can't find position for %v", b.Loop, scl.Types[scl.AbilityUnit[aid]].Name)
	return 0
}

func (b *bot) BuildFirstBarrack() {
	pos := FirstBarrackBuildPos[0]
	scv := b.Units.My[terran.SCV].ClosestTo(pos)
	if scv != nil {
		b.Groups.Add(Builders, scv)
		b.OrderBuild(scv, pos, ability.Build_Barracks)
	}
}

func (b *bot) BuildRefinery(cc *scl.Unit) {
	// Find first geyser that is close to selected cc, but it doesn't have Refinery on top of it
	builders := b.Groups.Get(Builders).Units
	geyser := b.Units.Geysers.All().CloserThan(10, cc).First(func(unit *scl.Unit) bool {
		return b.Units.My[terran.Refinery].CloserThan(1, unit).Len() == 0 &&
			unit.FindAssignedBuilder(builders) == nil
	})
	if geyser != nil {
		scv := b.GetSCV(geyser, Builders, 45)
		if scv != nil {
			scv.CommandTag(ability.Build_Refinery, geyser.Tag)
			b.DeductResources(ability.Build_Refinery)
			log.Debugf("%d: Building Refinery", b.Loop)
		}
	}
}

func (b *bot) ProcessBuildOrder(buildOrder BuildNodes) {
	for _, node := range buildOrder {
		inLimits := b.Pending(node.Ability) < node.Limit(b) && b.Orders[node.Ability] < node.Active(b)
		canBuy := b.CanBuy(node.Ability)
		waitRes := node.WaitRes != nil && node.WaitRes(b)
		if (node.Premise == nil || node.Premise(b)) && inLimits && (canBuy || waitRes) {
			if !canBuy && waitRes {
				// reserve money for building
				b.DeductResources(node.Ability)
				continue
			}
			if node.Method != nil {
				node.Method(b)
			} else {
				b.Build(node.Ability)
			}
		}
		if node.Unlocks != nil && b.Units.My[scl.AbilityUnit[node.Ability]].Exists() {
			b.ProcessBuildOrder(node.Unlocks)
		}
	}
}

func (b *bot) OrderUpgrades() {
	lab := b.Units.My[terran.BarracksTechLab].First(scl.Ready, scl.Idle)
	if lab != nil {
		b.RequestAvailableAbilities(true, lab) // todo: request true each frame -> HasTrueAbility?
		if !b.Upgrades[ability.Research_ConcussiveShells] && b.PendingAliases(ability.Train_Marauder) >= 2 &&
			lab.HasAbility(ability.Research_ConcussiveShells) && b.CanBuy(ability.Research_ConcussiveShells) {
			lab.Command(ability.Research_ConcussiveShells)
			return
		}
		if !b.Upgrades[ability.Research_CombatShield] && b.Units.My[terran.Marine].Len() >= 4 &&
			lab.HasAbility(ability.Research_CombatShield) && b.CanBuy(ability.Research_CombatShield) {
			lab.Command(ability.Research_CombatShield)
			return
		}
		if (b.Upgrades[ability.Research_ConcussiveShells] || b.PendingAliases(ability.Research_ConcussiveShells) > 0 ||
			b.Upgrades[ability.Research_CombatShield] || b.PendingAliases(ability.Research_CombatShield) > 0) &&
			!b.Upgrades[ability.Research_Stimpack] && lab.HasAbility(ability.Research_Stimpack) &&
			b.CanBuy(ability.Research_Stimpack) {
			lab.Command(ability.Research_Stimpack)
			return
		}
	}

	eng := b.Units.My[terran.EngineeringBay].First(scl.Ready, scl.Idle)
	if eng != nil {
		b.RequestAvailableAbilities(true, eng) // request abilities again because we want to ignore resource reqs
		if b.Units.My[terran.Marine].Len()+b.Units.My[terran.Marauder].Len()*2+b.Units.My[terran.Reaper].Len()*2 >= 8 {
			for _, a := range []api.AbilityID{
				ability.Research_TerranInfantryWeaponsLevel1,
				ability.Research_TerranInfantryArmorLevel1,
				ability.Research_TerranInfantryWeaponsLevel2,
				ability.Research_TerranInfantryArmorLevel2,
				ability.Research_TerranInfantryWeaponsLevel3,
				ability.Research_TerranInfantryArmorLevel3,
			} {
				if b.Upgrades[a] {
					continue
				}
				if eng.HasAbility(a) {
					if b.CanBuy(a) {
						eng.Command(a)
						return
					} else {
						// reserve money for upgrade
						b.DeductResources(a)
					}
					break
				}
			}
		}
		if !b.Upgrades[ability.Research_HiSecAutoTracking] && b.Units.AllEnemy[terran.Banshee].Exists() &&
			eng.HasAbility(ability.Research_HiSecAutoTracking) && b.CanBuy(ability.Research_HiSecAutoTracking) {
			eng.Command(ability.Research_HiSecAutoTracking)
			return
		}
	}

	// todo: aliases
	if arm := b.Units.My[terran.Armory].First(scl.Ready, scl.Idle); arm != nil && b.Units.My.OfType(terran.WidowMine,
		terran.Hellion, terran.Cyclone, terran.SiegeTank, terran.Raven, terran.Battlecruiser).Len() > 4 {
		b.RequestAvailableAbilities(true, arm) // request abilities again because we want to ignore resource reqs
		upgrades := []api.AbilityID{
			ability.Research_TerranVehicleAndShipPlatingLevel1,
			ability.Research_TerranVehicleAndShipPlatingLevel2,
			ability.Research_TerranVehicleAndShipPlatingLevel3,
			ability.Research_TerranVehicleWeaponsLevel1,
			ability.Research_TerranVehicleWeaponsLevel2,
			ability.Research_TerranVehicleWeaponsLevel3,
		}
		if b.Units.My[terran.Battlecruiser].Exists() {
			upgrades = append([]api.AbilityID{
				ability.Research_TerranShipWeaponsLevel1,
				ability.Research_TerranShipWeaponsLevel2,
				ability.Research_TerranShipWeaponsLevel3,
			}, upgrades...)
		}
		for _, a := range upgrades {
			if b.Upgrades[a] {
				continue
			}
			if arm.HasAbility(a) {
				if b.CanBuy(a) {
					arm.Command(a)
					return
				} else {
					// reserve money for upgrade
					b.DeductResources(a)
				}
				break
			}
		}
	}

	lab = b.Units.My[terran.FactoryTechLab].First(scl.Ready, scl.Idle)
	if lab != nil && (b.Units.My[terran.Cyclone].Exists() || b.Units.My[terran.WidowMine].Exists()) {
		b.RequestAvailableAbilities(true, lab)
		if b.PendingAliases(ability.Train_Cyclone) >= 2 &&
			lab.HasAbility(ability.Research_CycloneResearchLockOnDamageUpgrade) &&
			b.CanBuy(ability.Research_CycloneResearchLockOnDamageUpgrade) {
			lab.Command(ability.Research_CycloneResearchLockOnDamageUpgrade)
			return
		}
		if b.PendingAliases(ability.Train_WidowMine) >= 2 && lab.HasAbility(ability.Research_DrillingClaws) &&
			b.CanBuy(ability.Research_DrillingClaws) {
			lab.Command(ability.Research_DrillingClaws)
			return
		}
		if b.PendingAliases(ability.Train_Hellion) >= 4 && lab.HasAbility(ability.Research_InfernalPreigniter) &&
			b.CanBuy(ability.Research_InfernalPreigniter) {
			lab.Command(ability.Research_InfernalPreigniter)
			return
		}
	}

	fc := b.Units.My[terran.FusionCore].First(scl.Ready, scl.Idle)
	if fc != nil && b.Pending(ability.Train_Battlecruiser) > 0 &&
		!b.Upgrades[ability.Research_BattlecruiserWeaponRefit] {
		b.RequestAvailableAbilities(true, fc)
		if fc.HasAbility(ability.Research_BattlecruiserWeaponRefit) &&
			b.CanBuy(ability.Research_BattlecruiserWeaponRefit) {
			fc.Command(ability.Research_BattlecruiserWeaponRefit)
			return
		}
	}
}

func (b *bot) Morph() {
	cc := b.Units.My[terran.CommandCenter].First(scl.Ready, scl.Idle)
	if cc != nil && b.Units.My[terran.Barracks].First(scl.Ready) != nil {
		if b.CanBuy(ability.Morph_OrbitalCommand) {
			b.OrderTrain(cc, ability.Morph_OrbitalCommand)
		} else if b.Units.My[terran.SCV].Len() >= 16 {
			b.DeductResources(ability.Morph_OrbitalCommand)
		}
	}
	groundEnemies := b.Units.AllEnemy.All().Filter(scl.NotFlying)
	for _, supply := range b.Units.My[terran.SupplyDepot] {
		if groundEnemies.CloserThan(4, supply).Empty() {
			supply.Command(ability.Morph_SupplyDepot_Lower)
		}
	}
	for _, supply := range b.Units.My[terran.SupplyDepotLowered] {
		if groundEnemies.CloserThan(4, supply).Exists() {
			supply.Command(ability.Morph_SupplyDepot_Raise)
		}
	}
}

func (b *bot) Cast() {
	cc := b.Units.My[terran.OrbitalCommand].
		Filter(func(unit *scl.Unit) bool { return unit.Energy >= 50 }).
		Max(func(unit *scl.Unit) float64 { return float64(unit.Energy) })
	if cc != nil {
		// Scan
		if b.Orders[ability.Effect_Scan] == 0 && b.EffectPoints(effect.ScannerSweep).Empty() {
			allEnemies := b.Units.AllEnemy.All()
			visibleEnemies := allEnemies.Filter(scl.PosVisible)
			units := b.Units.My.All()
			// Reaper wants to see highground
			if b.Units.My[terran.Raven].Empty() {
				if reaper := b.Groups.Get(Reapers).Units.ClosestTo(b.Locs.EnemyStart); reaper != nil {
					if enemy := allEnemies.CanAttack(reaper, 1).ClosestTo(reaper); enemy != nil {
						if !b.IsVisible(enemy) && b.HeightAt(enemy) > b.HeightAt(reaper) {
							pos := enemy.Towards(b.Locs.EnemyStart, 8)
							cc.CommandPos(ability.Effect_Scan, pos)
							log.Debug("Reaper sight scan")
							return
						}
					}
				}
			}

			// Vision for tanks
			tanks := b.Units.My[terran.SiegeTankSieged]
			tanks.OrderByDistanceTo(b.Locs.EnemyStart, false)
			for _, tank := range tanks {
				targets := allEnemies.InRangeOf(tank, 0)
				if targets.Exists() && visibleEnemies.InRangeOf(tank, 0).Empty() {
					target := targets.ClosestTo(b.Locs.EnemyStart)
					cc.CommandPos(ability.Effect_Scan, target)
					log.Debug("Tank sight scan")
				}
			}

			// Lurkers
			if eps := b.EffectPoints(effect.LurkerSpines); eps.Exists() {
				// todo: check if bot already sees the lurker using his position approximation
				cc.CommandPos(ability.Effect_Scan, eps.ClosestTo(b.Locs.EnemyStart))
				log.Debug("Lurker scan")
				return
			}

			// DTs
			if b.EnemyRace == api.Race_Protoss {
				dts := b.Units.Enemy[protoss.DarkTemplar]
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
			if b.EnemyRace == api.Race_Terran {
				for _, u := range units {
					if u.HitsLost == 12 && allEnemies.CanAttack(u, 2).Empty() {
						cc.CommandPos(ability.Effect_Scan, u)
						log.Debug("Banshee scan")
						return
					}
				}
			}

			// Recon scan at 4:00
			pos := b.Locs.EnemyMainCenter
			if b.EnemyRace == api.Race_Zerg {
				pos = b.Locs.EnemyStart
			}
			if b.Loop >= 5376 && !b.IsExplored(pos) {
				cc.CommandPos(ability.Effect_Scan, pos)
				log.Debug("Recon scan")
				return
			}
		}
		// Mule
		if cc.Energy >= 75 || (b.Loop < 4928 && cc.Energy >= 50) { // 3:40
			ccs := b.Units.My.OfType(terran.CommandCenter, terran.OrbitalCommand,
				terran.PlanetaryFortress).Filter(scl.Ready)
			ccs.OrderByDistanceTo(cc, false)
			for _, target := range ccs {
				homeMineral := b.Units.Minerals.All().CloserThan(scl.ResourceSpreadDistance, target).
					Filter(func(unit *scl.Unit) bool { return unit.MineralContents > 400 }).
					Max(func(unit *scl.Unit) float64 { return float64(unit.MineralContents) })
				if homeMineral != nil {
					cc.CommandTag(ability.Effect_CalldownMULE, homeMineral.Tag)
				}
			}
		}
	}
}

func (b *bot) OrderUnits() {
	mech := false
	if b.EnemyRace != api.Race_Zerg {
		mech = true
	}
	if (WorkerRush /* || b.getEmptyBunker(scl.Pt0()) != nil*/) && b.CanBuy(ability.Train_Marine) {
		if rax := b.Units.My[terran.Barracks].First(scl.Ready, scl.Unused); rax != nil {
			if rax.HasReactor() && scl.UnitsOrders[rax.Tag].Loop+b.FramesPerOrder <= b.Loop {
				rax.SpamCmds = true
			}
			b.OrderTrain(rax, ability.Train_Marine)
		}
	}

	ccs := b.Units.My.OfType(terran.CommandCenter, terran.OrbitalCommand, terran.PlanetaryFortress)
	cc := ccs.First(scl.Ready, scl.Idle)
	refs := b.Units.My[terran.Refinery].Filter(func(unit *scl.Unit) bool {
		return unit.IsReady() && unit.VespeneContents > 0
	})
	if cc != nil && b.Units.My[terran.SCV].Len() < scl.MinInt(21*ccs.Len(), 70-refs.Len()) &&
		b.CanBuy(ability.Train_SCV) && !WorkerRush {
		b.OrderTrain(cc, ability.Train_SCV)
	}

	starport := b.Units.My[terran.Starport].First(scl.Ready, scl.Unused, scl.HasTechlab)
	if starport != nil {
		ravens := b.Pending(ability.Train_Raven)
		if b.Units.My[terran.FusionCore].First(scl.Ready) != nil {
			if b.CanBuy(ability.Train_Battlecruiser) {
				b.OrderTrain(starport, ability.Train_Battlecruiser)
			} else {
				b.DeductResources(ability.Train_Battlecruiser) // Gather money
			}
		}
		if ravens < 2 {
			if b.CanBuy(ability.Train_Raven) {
				b.OrderTrain(starport, ability.Train_Raven)
			} else if ravens == 0 {
				b.DeductResources(ability.Train_Raven) // Gather money
			}
		}
	}
	starport = b.Units.My[terran.Starport].First(scl.Ready, scl.Unused, func(unit *scl.Unit) bool {
		return starport == nil || unit.Tag != starport.Tag // Don't select previously selected producer
	})
	if starport != nil {
		if starport.HasReactor() && scl.UnitsOrders[starport.Tag].Loop+b.FramesPerOrder <= b.Loop {
			starport.SpamCmds = true
		}
		medivacs := b.Pending(ability.Train_Medivac)
		infantry := b.Units.My[terran.Marine].Len() + b.Units.My[terran.Marauder].Len()*2
		if (medivacs == 0 || medivacs*8 < infantry) && b.CanBuy(ability.Train_Medivac) {
			b.OrderTrain(starport, ability.Train_Medivac)
		} else if medivacs == 0 {
			b.DeductResources(ability.Train_Medivac) // Gather money
		}
	}

	factory := b.Units.My[terran.Factory].First(scl.Ready, scl.Unused, scl.HasTechlab)
	if factory != nil {
		cyclones := b.PendingAliases(ability.Train_Cyclone)
		tanks := b.PendingAliases(ability.Train_SiegeTank)

		buyCyclones := b.EnemyProduction.Len(terran.Banshee) > 0 && cyclones == 0
		buyTanks := PlayDefensive && tanks == 0
		if !buyCyclones && !buyTanks {
			cyclonesScore := b.EnemyProduction.Score(protoss.WarpPrism, protoss.Phoenix, protoss.VoidRay,
				protoss.Oracle, protoss.Tempest, protoss.Carrier, terran.VikingFighter, terran.Medivac,
				terran.Liberator, terran.Raven, terran.Banshee, terran.Battlecruiser, zerg.Queen, zerg.Mutalisk,
				zerg.Corruptor, zerg.Viper, zerg.Ultralisk, zerg.BroodLord) + 1
			tanksScore := b.EnemyProduction.Score(protoss.Stalker, protoss.Colossus, protoss.PhotonCannon,
				terran.Marine, terran.Reaper, terran.Marauder, terran.Bunker, /*zerg.Zergling, zerg.Baneling,*/
				zerg.Roach, zerg.Ravager, zerg.Hydralisk, zerg.LurkerMP, zerg.SpineCrawler) + 1
			buyCyclones = cyclonesScore/float64(cyclones+1) >= tanksScore/float64(tanks+1)
			buyTanks = !buyCyclones
		}

		if buyCyclones {
			if b.CanBuy(ability.Train_Cyclone) {
				b.OrderTrain(factory, ability.Train_Cyclone)
			} else if cyclones == 0 || mech {
				b.DeductResources(ability.Train_Cyclone) // Gather money
			}
		} else if buyTanks {
			if b.CanBuy(ability.Train_SiegeTank) {
				b.OrderTrain(factory, ability.Train_SiegeTank)
			} else if tanks == 0 || mech {
				b.DeductResources(ability.Train_SiegeTank) // Gather money
			}
		}
	}

	factory = b.Units.My[terran.Factory].First(func(unit *scl.Unit) bool {
		return unit.IsReady() && unit.IsUnused() && !unit.HasTechlab() && (factory == nil || unit.Tag != factory.Tag)
	})
	if factory != nil {
		if factory.HasReactor() && scl.UnitsOrders[factory.Tag].Loop+b.FramesPerOrder <= b.Loop {
			// I need to pass this param because else duplicate order will be ignored
			// But I need to be sure that there was no previous order recently
			factory.SpamCmds = true
		}
		mines := b.PendingAliases(ability.Train_WidowMine)
		hellions := b.PendingAliases(ability.Train_Hellion)

		minesScore := b.EnemyProduction.Score(protoss.Stalker, protoss.Archon, protoss.Phoenix, protoss.VoidRay,
			protoss.Oracle, protoss.Tempest, protoss.Carrier, terran.Cyclone, terran.SiegeTank, terran.Thor,
			terran.VikingFighter, terran.Medivac, terran.Liberator, terran.Raven, terran.Banshee,
			terran.Battlecruiser, zerg.Hydralisk, zerg.Queen, zerg.Roach, zerg.Ravager, zerg.Mutalisk, zerg.Corruptor,
			zerg.Viper, zerg.Ultralisk, zerg.BroodLord) + 1
		hellionsScore := b.EnemyProduction.Score(zerg.Zergling, zerg.Baneling, zerg.SwarmHostMP) + 1
		buyMines := minesScore/float64(mines+1) >= hellionsScore/float64(hellions+1)

		if buyMines {
			if b.CanBuy(ability.Train_WidowMine) {
				b.OrderTrain(factory, ability.Train_WidowMine)
			} else if mines == 0 || mech {
				b.DeductResources(ability.Train_WidowMine) // Gather money
			}
		} else {
			if b.CanBuy(ability.Train_Hellion) {
				b.OrderTrain(factory, ability.Train_Hellion)
			} else if hellions == 0 || mech {
				b.DeductResources(ability.Train_Hellion) // Gather money
			}
		}
	}

	rax := b.Units.My[terran.Barracks].First(scl.Ready, scl.Unused, scl.HasTechlab)
	if rax != nil {
		marines := b.PendingAliases(ability.Train_Marine)
		marauders := b.PendingAliases(ability.Train_Marauder)
		marinesScore := b.EnemyProduction.Score(protoss.Immortal, protoss.WarpPrism, protoss.Phoenix, protoss.VoidRay,
			protoss.Oracle, protoss.Tempest, protoss.Carrier, terran.VikingFighter, terran.Medivac, terran.Liberator,
			terran.Raven, terran.Banshee, terran.Battlecruiser, zerg.Mutalisk, zerg.Corruptor, zerg.Viper,
			zerg.BroodLord) + 1 //  zerg.Zergling,
		maraudersScore := b.EnemyProduction.Score(protoss.Zealot, protoss.Stalker, protoss.Adept, terran.Reaper,
			terran.Hellion, terran.WidowMine, terran.Cyclone, terran.Thor, zerg.Baneling, zerg.Roach, zerg.Ravager,
			zerg.Ultralisk) + 1
		buyMarauders := marinesScore/float64(marines+1) < maraudersScore/float64(marauders+1)

		if buyMarauders {
			if b.CanBuy(ability.Train_Marauder) {
				b.OrderTrain(rax, ability.Train_Marauder)
			} else {
				b.DeductResources(ability.Train_Marauder) // Gather money
			}
		}
	}
	rax = b.Units.My[terran.Barracks].First(scl.Ready, scl.Unused, func(unit *scl.Unit) bool {
		return rax == nil || unit.Tag != rax.Tag // Don't select previously selected producer
	})
	if rax != nil {
		if rax.HasReactor() && scl.UnitsOrders[rax.Tag].Loop+b.FramesPerOrder <= b.Loop {
			rax.SpamCmds = true
		}
		// Until 4:00
		// b.Loop < 5376 && (b.Pending(ability.Train_Reaper) < 2 || b.EnemyRace == api.Race_Zerg) &&
		// before 2:40 or if they are not dying until 4:00
		if !LingRush && (b.Loop < 3584 || (b.Loop < 5376 && b.Pending(ability.Train_Reaper) > b.Loop/1344)) &&
			b.CanBuy(ability.Train_Reaper) {
			b.OrderTrain(rax, ability.Train_Reaper)
		} else if /*b.Loop >= 2688 &&*/ b.CanBuy(ability.Train_Marine) { // 2:00
			b.OrderTrain(rax, ability.Train_Marine)
		}
	}
}

func (b *bot) ReserveSCVs() {
	// Fast first supply
	if b.Units.My.OfType(scl.UnitAliases.For(terran.SupplyDepot)...).Empty() && b.Groups.Get(ScvReserve).Tags.Empty() {
		pos := BuildPos[scl.S2x2][0]
		scv := b.GetSCV(pos, 0, 45) // Get SCV but don't change its group
		if scv != nil && scv.FramesToPos(pos)*b.MineralsPerFrame+float64(b.Minerals)+20 >= 100 {
			b.Groups.Add(ScvReserve, scv)
			scv.CommandPos(ability.Move, pos)
		}
	}
	// Fast expansion
	if b.Units.My.OfType(terran.CommandCenter, terran.OrbitalCommand, terran.PlanetaryFortress).Len() == 1 &&
		b.Minerals >= 350 && b.Groups.Get(ScvReserve).Tags.Empty() /*&& !PlayDefensive*/ && !WorkerRush {
		pos := b.Locs.MyExps[0]
		if scv := b.GetSCV(pos, ScvReserve, 45); scv != nil {
			scv.CommandPos(ability.Move, pos)
		}
	}
}

func (b *bot) Macro() {
	if !BuildTurrets && b.Units.Enemy.OfType(terran.Banshee, terran.Ghost, terran.WidowMine, terran.Medivac,
		terran.VikingFighter, terran.Liberator, terran.Battlecruiser, terran.Starport, zerg.Mutalisk, zerg.LurkerMP,
		zerg.Corruptor, zerg.Spire, zerg.GreaterSpire, protoss.DarkTemplar, protoss.WarpPrism, protoss.Phoenix,
		protoss.VoidRay, protoss.Oracle, protoss.Tempest, protoss.Carrier, protoss.Stargate, protoss.DarkShrine).
		Exists() {
		BuildTurrets = true
	}
	if findTurretPositionFor != nil {
		b.FindTurretPosition(findTurretPositionFor)
		findTurretPositionFor = nil
	}

	if lastBuildLoop+b.FramesPerOrder < b.Loop {
		if b.Loop >= 5376 { // 4:00
			b.OrderUpgrades()
		}
		b.ProcessBuildOrder(RootBuildOrder)
		b.Morph()
		b.OrderUnits()
		b.ReserveSCVs()
		lastBuildLoop = b.Loop
	}
	b.Cast()
}
