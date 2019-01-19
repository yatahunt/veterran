package main

import (
	"bitbucket.org/aisee/minilog"
	"bitbucket.org/aisee/sc2lib"
	"github.com/chippydip/go-sc2ai/api"
	"github.com/chippydip/go-sc2ai/enums/ability"
	"github.com/chippydip/go-sc2ai/enums/effect"
	"github.com/chippydip/go-sc2ai/enums/protoss"
	"github.com/chippydip/go-sc2ai/enums/terran"
	"github.com/chippydip/go-sc2ai/enums/zerg"
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

var BuildingsSizes = map[api.AbilityID]scl.BuildingSize{
	ability.Build_CommandCenter:  scl.S5x5,
	ability.Build_SupplyDepot:    scl.S2x2,
	ability.Build_Barracks:       scl.S5x3,
	ability.Build_Refinery:       scl.S3x3,
	ability.Build_EngineeringBay: scl.S3x3,
	ability.Build_MissileTurret:  scl.S2x2,
	ability.Build_Armory:         scl.S3x3,
	ability.Build_Factory:        scl.S5x3,
	ability.Build_Starport:       scl.S5x3,
	ability.Build_FusionCore:     scl.S3x3,
}

var RootBuildOrder = BuildNodes{
	{
		Name:    "First CC",
		Ability: ability.Build_CommandCenter,
		Limit:   BuildOne,
		Active:  BuildOne,
	},
	{
		Name:    "Expansion CCs",
		Ability: ability.Build_CommandCenter,
		Limit:   func(b *bot) int { return buildPos[scl.S5x5].Len() },
		Active:  BuildOne,
		WaitRes: func(b *bot) bool {
			ccs := b.Units.OfType(scl.UnitAliases.For(terran.CommandCenter)...)
			// First orbital is morphing
			if ccs.Len() == 1 && ccs.First().UnitType == terran.OrbitalCommand {
				return true
			}
			if ccs.Len() <= b.FoodUsed/35 {
				return true
			}
			return false
		},
	},
	{
		Name:    "Supplies",
		Ability: ability.Build_SupplyDepot,
		Premise: func(b *bot) bool { return b.FoodLeft < 4+b.FoodUsed/20 && b.FoodCap < 200 },
		Limit:   func(b *bot) int { return 30 },
		Active:  func(b *bot) int { return 1 + b.FoodUsed/50 },
	},
	{
		Name:    "Barrack",
		Ability: ability.Build_Barracks,
		Premise: func(b *bot) bool {
			return b.Units.OfType(scl.UnitAliases.For(terran.SupplyDepot)...).First(scl.Ready) != nil &&
				b.Units.OfType(scl.UnitAliases.For(terran.Barracks)...).Empty()
		},
		Limit:  BuildOne,
		Active: BuildOne,
		Method: func(b *bot) { b.BuildFirstBarrack() },
	},
	{
		Name:    "Refinery",
		Ability: ability.Build_Refinery,
		Premise: func(b *bot) bool {
			if workerRush {
				return false
			}
			if b.Vespene < b.Minerals*2 {
				raxPending := b.Pending(ability.Build_Barracks)
				refPending := b.Pending(ability.Build_Refinery)
				ccs := b.Units.OfType(scl.UnitAliases.For(terran.CommandCenter)...)
				if raxPending == 0 {
					return false
				}
				if b.Minerals > 500 {
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
			ccs := b.Units.OfType(terran.CommandCenter, terran.OrbitalCommand, terran.PlanetaryFortress)
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
			return b.Units[terran.Factory].First(scl.Ready, scl.Unused) == nil
		},
		Limit: func(b *bot) int {
			ccs := b.Units.OfType(scl.UnitAliases.For(terran.CommandCenter)...).Filter(scl.Ready)
			return scl.MinInt(4, ccs.Len())
		},
		Active:  BuildOne,
		Unlocks: FactoryBuildOrder,
	},
	{
		Name:    "Starport",
		Ability: ability.Build_Starport,
		Premise: func(b *bot) bool {
			return b.Units[terran.Starport].First(scl.Ready, scl.Unused) == nil
		},
		Limit: func(b *bot) int {
			ccs := b.Units.OfType(scl.UnitAliases.For(terran.CommandCenter)...)
			if ccs.Len() < 3 && b.Minerals < 500 {
				return 0
			}
			if b.Units[terran.FusionCore].First(scl.Ready) == nil {
				return 1
			}
			return scl.MinInt(4, ccs.Len())
		},
		Active:  BuildOne,
		Unlocks: StarportBuildOrder,
	},
}

var RaxBuildOrder = BuildNodes{
	{
		Name:    "Armory",
		Ability: ability.Build_Armory,
		Premise: func(b *bot) bool {
			// b.Units[terran.Factory].First(scl.Ready) != nil // Needs factory
			return b.Units[terran.EngineeringBay].First(scl.Ready) != nil
		},
		WaitRes: Yes,
		Limit: func(b *bot) int {
			if b.Units[terran.FusionCore].First(scl.Ready) != nil {
				return 2
			}
			return 1
		},
		Active: BuildOne,
	},
	/*{
		Name:    "Barracks",
		Ability: ability.Build_Barracks,
		Premise: func(b *bot) bool {
			return b.Units[terran.Barracks].First(scl.Ready, scl.Unused) == nil
		},
		Limit: func(b *bot) int {
			ccs := b.Units.OfType(scl.UnitAliases.For(terran.CommandCenter)...)
			// orbitals := b.Units.OfType(terran.OrbitalCommand)
			return scl.MinInt(5, ccs.Len())
		},
		Active: func(b *bot) int { return 2 },
	},*/
	{
		Name:    "Engineering Bay",
		Ability: ability.Build_EngineeringBay,
		Premise: func(b *bot) bool {
			return b.Units.OfType(scl.UnitAliases.For(terran.CommandCenter)...).Filter(scl.Ready).Len() >= 2
		},
		Limit:  BuildOne,
		Active: BuildOne,
	},
	{
		Name:    "Missile Turrets",
		Ability: ability.Build_MissileTurret,
		Premise: func(b *bot) bool {
			//  && b.Units[terran.EngineeringBay].First(scl.Ready) != nil
			return buildTurrets
		},
		Limit:   func(b *bot) int { return turretsPos.Len() },
		Active:  func(b *bot) int { return turretsPos.Len() },
		WaitRes: Yes,
	},
	{
		Name:    "Barracks reactors",
		Ability: ability.Build_Reactor_Barracks,
		Premise: func(b *bot) bool {
			return (b.Vespene >= 200 && b.Units[terran.Barracks].First(scl.Ready, scl.NoAddon, scl.Idle) != nil) ||
				b.Units[terran.BarracksFlying].First() != nil
		},
		Limit:  func(b *bot) int { return b.Units.OfType(scl.UnitAliases.For(terran.Barracks)...).Len() },
		Active: func(b *bot) int { return 2 },
		Method: func(b *bot) {
			// todo: group?
			if rax := b.Units[terran.BarracksFlying].First(); rax != nil {
				rax.CommandPos(ability.Build_Reactor_Barracks, firstBarrackBuildPos[1])
				return
			}

			rax := b.Units[terran.Barracks].First(scl.Ready, scl.NoAddon, scl.Idle)
			if rax.Point().IsCloserThan(3, firstBarrackBuildPos[0]) &&
				firstBarrackBuildPos[0] != firstBarrackBuildPos[1] {
				if b.EnemyUnits.Units().CloserThan(safeBuildRange, rax.Point()).Exists() {
					return
				}
				rax.Command(ability.Lift_Barracks)
			} else {
				rax.Command(ability.Build_Reactor_Barracks)
			}
		},
	},
}

var FactoryBuildOrder = BuildNodes{
	{
		Name:    "Factory Tech Lab",
		Ability: ability.Build_TechLab_Factory,
		Premise: func(b *bot) bool {
			return b.Units[terran.FactoryReactor].Exists() &&
				b.Units[terran.Factory].First(scl.Ready, scl.NoAddon, scl.Idle) != nil
		},
		Limit: func(b *bot) int {
			return b.Units[terran.Factory].Len() - 1
		},
		Active: BuildOne,
		Method: func(b *bot) {
			b.Units[terran.Factory].First(scl.Ready, scl.NoAddon, scl.Idle).Command(ability.Build_TechLab_Factory)
		},
	},
	{
		Name:    "Factory Reactor",
		Ability: ability.Build_Reactor_Factory,
		Premise: func(b *bot) bool {
			return b.Units[terran.Factory].First(scl.Ready, scl.NoAddon, scl.Idle) != nil
		},
		Limit: BuildOne,
		/*func(b *bot) int {
			return (b.Units[terran.Factory].Len() + 1) / 2
		},*/
		Active: BuildOne,
		Method: func(b *bot) {
			b.Units[terran.Factory].First(scl.Ready, scl.NoAddon, scl.Idle).Command(ability.Build_Reactor_Factory)
		},
	},
}

var StarportBuildOrder = BuildNodes{
	{
		Name:    "Starport Tech Lab",
		Ability: ability.Build_TechLab_Starport,
		Premise: func(b *bot) bool {
			return b.Units[terran.Starport].First(scl.Ready, scl.NoAddon, scl.Idle) != nil
		},
		Limit: func(b *bot) int {
			return b.Units[terran.Starport].Len()
		},
		Active: BuildOne,
		Method: func(b *bot) {
			b.Units[terran.Starport].First(scl.Ready, scl.NoAddon, scl.Idle).Command(ability.Build_TechLab_Starport)
		},
	},
	{
		Name:    "Fusion Core",
		Ability: ability.Build_FusionCore,
		Premise: func(b *bot) bool {
			return b.Units[terran.Raven].Exists()
		},
		Limit:  BuildOne,
		Active: BuildOne,
	},
}

func (b *bot) OrderBuild(scv *scl.Unit, pos scl.Point, aid api.AbilityID) {
	scv.CommandPos(aid, pos)
	b.DeductResources(aid)
	log.Debugf("%d: Building %v", b.Loop, scl.Types[scl.AbilityUnit[aid]].Name)
}

func (b *bot) OrderTrain(factory *scl.Unit, aid api.AbilityID) {
	factory.Command(aid)
	b.DeductResources(aid)
	log.Debugf("%d: Training %v", b.Loop, scl.Types[scl.AbilityUnit[aid]].Name)
}

func (b *bot) Build(aid api.AbilityID) bool {
	size, ok := BuildingsSizes[aid]
	if !ok {
		log.Alertf("Can't find size for %v", scl.Types[scl.AbilityUnit[aid]].Name)
		return false
	}

	techReq := scl.Types[scl.AbilityUnit[aid]].TechRequirement
	if techReq != 0 && b.Units.OfType(scl.UnitAliases.For(techReq)...).Empty() {
		return false // Not available because of tech reqs, like: supply is needed for barracks
	}

	buildersTargets := map[scl.Point]bool{}
	for _, builder := range b.Groups.Get(Builders).Units {
		buildersTargets[builder.TargetPos()] = true
	}

	enemies := b.AllEnemyUnits.Units()
	positions := buildPos[size]
	if size == scl.S3x3 {
		// Add larger building positions if there is not enough S3x3 positions
		positions = append(positions, buildPos[scl.S5x3]...)
	}
	if aid == ability.Build_MissileTurret {
		positions = turretsPos
	}
	for _, pos := range positions {
		if buildersTargets[pos] {
			continue // Someone already constructing there
		}
		if !b.IsPosOk(pos, size, 0, scl.IsBuildable, scl.IsNoCreep) {
			continue
		}
		if enemies.CloserThan(safeBuildRange, pos).Exists() {
			continue
		}

		scv := b.GetSCV(pos, Builders, 45)
		if scv != nil {
			b.OrderBuild(scv, pos, aid)
			return true
		}
		log.Debugf("%d: Failed to find SCV", b.Loop)
		return false
	}
	log.Debugf("%d: Can't find position for %v", b.Loop, scl.Types[scl.AbilityUnit[aid]].Name)
	return false
}

func (b *bot) BuildingsCheck() {
	builders := b.Groups.Get(Builders).Units
	buildings := b.Groups.Get(UnderConstruction).Units
	enemies := b.EnemyUnits.Units().Filter(scl.DpsGt5)
	// This is const. Move somewhere else?
	addonsTypes := append(scl.UnitAliases.For(terran.Reactor), scl.UnitAliases.For(terran.TechLab)...)
	for _, building := range buildings {
		if building.BuildProgress == 1 {
			switch building.UnitType {
			case terran.Barracks:
				fallthrough
			case terran.Factory:
				building.CommandPos(ability.Rally_Building, b.MainRamp.Top+b.MainRamp.Vec*3)
				b.Groups.Add(Buildings, building)
			default:
				b.Groups.Add(Buildings, building) // And remove from current group
			}
			continue
		}

		// Cancel building if it will be destroyed soon
		if building.HPS*2.5 > building.Hits {
			building.Command(ability.Cancel)
			continue
		}

		// Find SCV to continue work if disrupted
		if building.FindAssignedBuilder(builders) == nil &&
			enemies.CanAttack(building, 0).Empty() &&
			!addonsTypes.Contain(building.UnitType) {
			scv := b.GetSCV(building.Point(), Builders, 45)
			if scv != nil {
				scv.CommandTag(ability.Smart, building.Tag)
			}
		}

		// Cancel refinery if worker rush is detected and don't build new until enemy is gone
		if workerRush && building.UnitType == terran.Refinery {
			building.Command(ability.Cancel)
		}
	}
}

func (b *bot) BuildFirstBarrack() {
	pos := firstBarrackBuildPos[0]
	scv := b.Units[terran.SCV].ClosestTo(pos)
	if scv != nil {
		b.Groups.Add(Builders, scv)
		b.OrderBuild(scv, pos, ability.Build_Barracks)
	}
}

func (b *bot) BuildRefinery(cc *scl.Unit) {
	// Find first geyser that is close to selected cc, but it doesn't have Refinery on top of it
	builders := b.Groups.Get(Builders).Units
	geyser := b.VespeneGeysers.Units().CloserThan(10, cc.Point()).First(func(unit *scl.Unit) bool {
		return b.Units[terran.Refinery].CloserThan(1, unit.Point()).Len() == 0 &&
			unit.FindAssignedBuilder(builders) == nil
	})
	if geyser != nil {
		scv := b.GetSCV(geyser.Point(), Builders, 45)
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
		if node.Unlocks != nil && b.Units[scl.AbilityUnit[node.Ability]].Exists() {
			b.ProcessBuildOrder(node.Unlocks)
		}
	}
}

func (b *bot) OrderUpgrades() {
	if eng := b.Units[terran.EngineeringBay].First(scl.Ready, scl.Idle); eng != nil {
		b.RequestAvailableAbilities(true, eng) // request abilities again because we want to ignore resource reqs
		if b.Units[terran.Reaper].Len() > 4 {
			for _, a := range []api.AbilityID{
				ability.Research_TerranInfantryWeaponsLevel1,
				ability.Research_TerranInfantryWeaponsLevel2,
				ability.Research_TerranInfantryWeaponsLevel3,
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
		if !b.Upgrades[ability.Research_HiSecAutoTracking] && b.AllEnemyUnits[terran.Banshee].Exists() &&
			eng.HasAbility(ability.Research_HiSecAutoTracking) && b.CanBuy(ability.Research_HiSecAutoTracking) {
			eng.Command(ability.Research_HiSecAutoTracking)
			return
		}
	}

	if arm := b.Units[terran.Armory].First(scl.Ready, scl.Idle); arm != nil {
		b.RequestAvailableAbilities(true, arm) // request abilities again because we want to ignore resource reqs
		upgrades := []api.AbilityID{
			ability.Research_TerranVehicleAndShipPlatingLevel1,
			ability.Research_TerranVehicleAndShipPlatingLevel2,
			ability.Research_TerranVehicleAndShipPlatingLevel3,
		}
		if b.Units[terran.Battlecruiser].Exists() {
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

	lab := b.Units[terran.FactoryTechLab].First(scl.Ready, scl.Idle)
	if lab != nil && (b.Units[terran.Cyclone].Exists() || b.Units[terran.WidowMine].Exists()) {
		b.RequestAvailableAbilities(true, lab)
		if b.Units[terran.Cyclone].Exists() && lab.HasAbility(ability.Research_CycloneResearchLockOnDamageUpgrade) &&
			b.CanBuy(ability.Research_CycloneResearchLockOnDamageUpgrade) {
			lab.Command(ability.Research_CycloneResearchLockOnDamageUpgrade)
			return
		}
		if b.Units[terran.WidowMine].Exists() && lab.HasAbility(ability.Research_DrillingClaws) &&
			b.CanBuy(ability.Research_DrillingClaws) {
			lab.Command(ability.Research_DrillingClaws)
			return
		}
	}

	fc := b.Units[terran.FusionCore].First(scl.Ready, scl.Idle)
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
	cc := b.Units[terran.CommandCenter].First(scl.Ready, scl.Idle)
	if cc != nil && b.Units[terran.Barracks].First(scl.Ready) != nil {
		if b.CanBuy(ability.Morph_OrbitalCommand) {
			b.OrderTrain(cc, ability.Morph_OrbitalCommand)
		} else if b.Units[terran.SCV].Len() >= 16 {
			b.DeductResources(ability.Morph_OrbitalCommand)
		}
	}
	groundEnemies := b.AllEnemyUnits.Units().Filter(scl.NotFlying)
	for _, supply := range b.Units[terran.SupplyDepot] {
		if groundEnemies.CloserThan(4, supply.Point()).Empty() {
			supply.Command(ability.Morph_SupplyDepot_Lower)
		}
	}
	for _, supply := range b.Units[terran.SupplyDepotLowered] {
		if groundEnemies.CloserThan(4, supply.Point()).Exists() {
			supply.Command(ability.Morph_SupplyDepot_Raise)
		}
	}
}

func (b *bot) Cast() {
	cc := b.Units[terran.OrbitalCommand].
		Filter(func(unit *scl.Unit) bool { return unit.Energy >= 50 }).
		Max(func(unit *scl.Unit) float64 { return float64(unit.Energy) })
	if cc != nil {
		// Scan
		if b.Orders[ability.Effect_Scan] == 0 && b.EffectPoints(effect.ScannerSweep).Empty() {
			allEnemies := b.AllEnemyUnits.Units()
			visibleEnemies := allEnemies.Filter(scl.PosVisible)
			units := b.Units.Units()
			// Reaper wants to see highground
			if b.Units[terran.Raven].Empty() {
				if reaper := b.Groups.Get(Reapers).Units.ClosestTo(b.EnemyStartLoc); reaper != nil {
					if enemy := allEnemies.CanAttack(reaper, 1).ClosestTo(reaper.Point()); enemy != nil {
						if !b.IsVisible(enemy.Point()) && b.HeightAt(enemy.Point()) > b.HeightAt(reaper.Point()) {
							pos := enemy.Point().Towards(b.EnemyStartLoc, 8)
							cc.CommandPos(ability.Effect_Scan, pos)
							log.Debug("Reaper sight scan")
							return
						}
					}
				}
			}

			// Vision for tanks
			tanks := b.Units[terran.SiegeTankSieged]
			tanks.OrderBy(func(unit *scl.Unit) float64 {
				return unit.Point().Dist2(b.EnemyStartLoc)
			}, false)
			for _, tank := range tanks {
				targets := allEnemies.InRangeOf(tank, 0)
				if targets.Exists() && visibleEnemies.InRangeOf(tank, 0).Empty() {
					pos := targets.ClosestTo(b.EnemyStartLoc).Point()
					cc.CommandPos(ability.Effect_Scan, pos)
					log.Debug("Tank sight scan")
				}
			}

			// Lurkers
			if eps := b.EffectPoints(effect.LurkerSpines); eps.Exists() {
				// todo: check if bot already sees the lurker using his position approximation
				cc.CommandPos(ability.Effect_Scan, eps.ClosestTo(b.EnemyStartLoc))
				log.Debug("Lurker scan")
				return
			}

			// DTs
			if b.EnemyRace == api.Race_Protoss {
				dts := b.EnemyUnits[protoss.DarkTemplar]
				hitByDT := units.First(func(unit *scl.Unit) bool {
					return unit.HitsLost >= 41 && !unit.IsArmored() && !dts.CanAttack(unit, 0).Exists()
				})
				if hitByDT != nil {
					cc.CommandPos(ability.Effect_Scan, hitByDT.Point())
					log.Debug("DT scan")
					return
				}
			}

			// Early banshee without upgrades
			if b.EnemyRace == api.Race_Terran {
				for _, u := range units {
					if u.HitsLost == 12 && allEnemies.CanAttack(u, 2).Empty() {
						cc.CommandPos(ability.Effect_Scan, u.Point())
						log.Debug("Banshee scan")
						return
					}
				}
			}
		}
		// Mule
		if cc.Energy >= 75 || (b.Loop < 4032 && cc.Energy >= 50) { // 3 min
			homeMineral := b.MineralFields.Units().
				CloserThan(scl.ResourceSpreadDistance, cc.Point()).
				Max(func(unit *scl.Unit) float64 {
					return float64(unit.MineralContents)
				})
			if homeMineral != nil {
				cc.CommandTag(ability.Effect_CalldownMULE, homeMineral.Tag)
			}
		}
	}
}

func (b *bot) OrderUnits() {
	if workerRush && b.CanBuy(ability.Train_Marine) {
		if rax := b.Units[terran.Barracks].First(scl.Ready, scl.Unused); rax != nil {
			b.OrderTrain(rax, ability.Train_Marine)
		}
	}

	ccs := b.Units.OfType(terran.CommandCenter, terran.OrbitalCommand, terran.PlanetaryFortress)
	cc := ccs.First(scl.Ready, scl.Idle)
	refs := b.Units[terran.Refinery].Filter(func(unit *scl.Unit) bool {
		return unit.IsReady() && unit.VespeneContents > 0
	})
	if cc != nil && b.Units[terran.SCV].Len() < scl.MinInt(21*ccs.Len(), 70-refs.Len()) &&
		b.CanBuy(ability.Train_SCV) && !workerRush {
		b.OrderTrain(cc, ability.Train_SCV)
	}

	starport := b.Units[terran.Starport].First(scl.Ready, scl.Unused, scl.HasTechlab)
	if starport != nil {
		ravens := b.Pending(ability.Train_Raven)
		if b.Units[terran.FusionCore].First(scl.Ready) != nil {
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

	factory := b.Units[terran.Factory].First(scl.Ready, scl.Unused, scl.HasTechlab)
	if factory != nil {
		cyclones := b.Units[terran.Cyclone].Len()
		tanks := b.Units.OfType(scl.UnitAliases.For(terran.SiegeTank)...).Len()
		if cyclones <= tanks {
			if b.CanBuy(ability.Train_Cyclone) {
				b.OrderTrain(factory, ability.Train_Cyclone)
			} else if cyclones == 0 {
				b.DeductResources(ability.Train_Cyclone) // Gather money
			}
		} else {
			if b.CanBuy(ability.Train_SiegeTank) {
				b.OrderTrain(factory, ability.Train_SiegeTank)
			} else if tanks == 0 {
				b.DeductResources(ability.Train_SiegeTank) // Gather money
			}
		}
	}
	factory = b.Units[terran.Factory].First(func(unit *scl.Unit) bool {
		return unit.IsReady() && unit.IsUnused() && !unit.HasTechlab()
	})
	if factory != nil && b.CanBuy(ability.Train_WidowMine) && b.PendingAliases(ability.Train_WidowMine) < 8 {
		if scl.UnitsOrders[factory.Tag].Loop+b.FramesPerOrder <= b.Loop {
			// I need to pass this param because else duplicate order will be ignored
			// But I need to be sure that there was no previous order recently
			factory.SpamCmds = true
		}
		b.OrderTrain(factory, ability.Train_WidowMine)
	}

	rax := b.Units[terran.Barracks].First(scl.Ready, scl.Unused)
	if rax != nil {
		if (b.Pending(ability.Train_Reaper) < 4 || b.EnemyRace == api.Race_Zerg) && b.CanBuy(ability.Train_Reaper) {
			if scl.UnitsOrders[rax.Tag].Loop+b.FramesPerOrder <= b.Loop {
				rax.SpamCmds = true
			}
			b.OrderTrain(rax, ability.Train_Reaper)
		}
		if b.Minerals > 600 && b.CanBuy(ability.Train_Marine) {
			if scl.UnitsOrders[rax.Tag].Loop+b.FramesPerOrder <= b.Loop {
				rax.SpamCmds = true
			}
			b.OrderTrain(rax, ability.Train_Marine)
		}
	}
}

func (b *bot) ReserveSCVs() {
	// Fast first supply
	if b.Units.OfType(scl.UnitAliases.For(terran.SupplyDepot)...).Empty() && b.Groups.Get(ScvReserve).Tags.Empty() {
		pos := buildPos[scl.S2x2][0]
		scv := b.GetSCV(pos, 0, 45) // Get SCV but don't change its group
		if scv != nil && scv.FramesToPos(pos)*b.MineralsPerFrame+float64(b.Minerals)+20 >= 100 {
			b.Groups.Add(ScvReserve, scv)
			scv.CommandPos(ability.Move, pos)
		}
	}
}

func (b *bot) Macro() {
	if !buildTurrets && b.EnemyUnits.OfType(terran.Banshee, terran.Ghost, terran.WidowMine, terran.Medivac,
		terran.VikingFighter, terran.Liberator, terran.Battlecruiser, terran.Starport, zerg.Mutalisk, zerg.LurkerMP,
		zerg.Corruptor, zerg.Spire, zerg.GreaterSpire, protoss.DarkTemplar, protoss.WarpPrism, protoss.Phoenix,
		protoss.VoidRay, protoss.Oracle, protoss.Tempest, protoss.Carrier, protoss.Stargate, protoss.DarkShrine).
		Exists() {
		buildTurrets = true
	}
	if findTurretPositionFor != nil {
		b.FindTurretPosition(findTurretPositionFor)
		findTurretPositionFor = nil
	}

	b.BuildingsCheck()
	if lastBuildLoop+b.FramesPerOrder < b.Loop {
		b.OrderUpgrades()
		b.ProcessBuildOrder(RootBuildOrder)
		b.Morph()
		b.OrderUnits()
		b.ReserveSCVs()
		lastBuildLoop = b.Loop
	}
	b.Cast()
}
