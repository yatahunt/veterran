package main

import (
	"bitbucket.org/aisee/minilog"
	"bitbucket.org/aisee/sc2lib"
	"github.com/chippydip/go-sc2ai/api"
	"github.com/chippydip/go-sc2ai/enums/ability"
	"github.com/chippydip/go-sc2ai/enums/effect"
	"github.com/chippydip/go-sc2ai/enums/terran"
	"math"
)

type Booler func(b *bot) bool
type Inter func(b *bot) int
type Voider func(b *bot)
type BuildNode struct {
	Name    string
	Ability api.AbilityID
	Premise Booler
	WaitRes bool
	Limit   Inter
	Active  Inter
	Method  Voider
	Unlocks BuildNodes // [] ?
}
type BuildNodes []BuildNode

func BuildOne(b *bot) int { return 1 }

var BuildingsSizes = map[api.AbilityID]scl.BuildingSize{
	ability.Build_CommandCenter:  scl.S5x5,
	ability.Build_SupplyDepot:    scl.S2x2,
	ability.Build_Barracks:       scl.S5x3,
	ability.Build_Refinery:       scl.S3x3,
	ability.Build_EngineeringBay: scl.S3x3,
	ability.Build_Armory:         scl.S3x3,
}

var RootBuildOrder = BuildNodes{
	{
		Name:    "First CC",
		Ability: ability.Build_CommandCenter,
		Limit:   BuildOne,
		Active:  BuildOne,
	},
	{
		Name:    "Supplies",
		Ability: ability.Build_SupplyDepot,
		Premise: func(b *bot) bool { return b.FoodLeft < 6+b.FoodUsed/20 && b.FoodCap < 200 },
		Limit:   func(b *bot) int { return 30 },
		Active:  func(b *bot) int { return 1 + b.FoodUsed/50 },
	},
	{
		Name:    "First barrack",
		Ability: ability.Build_Barracks,
		Premise: func(b *bot) bool {
			return b.Units.OfType(terran.SupplyDepot, terran.SupplyDepotLowered).First(scl.Ready) != nil
		},
		Limit:  BuildOne,
		Active: BuildOne,
		Method: func(b *bot) { b.BuildFirstBarrack() },
	},
	{
		Name:    "Refineries",
		Ability: ability.Build_Refinery,
		Premise: func(b *bot) bool {
			raxPending := b.Pending(ability.Build_Barracks)
			refPending := b.Pending(ability.Build_Refinery)
			// todo: limit by gas amount?
			return raxPending > 0 && refPending == 0 || raxPending >= 3 && refPending >= 1
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
		Name:    "Expansion CCs",
		Ability: ability.Build_CommandCenter,
		Limit:   func(b *bot) int { return buildPos[scl.S5x5].Len() },
		Active:  BuildOne,
	},
}

var RaxBuildOrder = BuildNodes{
	// Needs factory
	/*{
		Name:    "Armory",
		Ability: ability.Build_Armory,
		Premise: func(b *bot) bool {
			// todo: on half Weapons upgrade done
			return b.Units[terran.EngineeringBay].Filter(scl.Ready).Len() >= 2
		},
		WaitRes: true,
		Limit:   BuildOne,
		Active:  BuildOne,
	},*/
	{
		Name:    "Barracks",
		Ability: ability.Build_Barracks,
		Limit: func(b *bot) int {
			ccs := b.Units.OfType(scl.UnitAliases.For(terran.CommandCenter)...)
			return 3 * ccs.Len()
		},
		Active: func(b *bot) int { return 3 },
	},
	{
		Name:    "Engineering Bays",
		Ability: ability.Build_EngineeringBay,
		Premise: func(b *bot) bool {
			return b.Units.OfType(scl.UnitAliases.For(terran.CommandCenter)...).Filter(scl.Ready).Len() >= 2
		},
		Limit:  func(b *bot) int { return 1 },
		Active: func(b *bot) int { return 1 },
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

	buildings := b.Units.Units().Filter(scl.Structure)
	enemies := b.AllEnemyUnits.Units()
	positions := buildPos[size]
	if size == scl.S3x3 {
		// Add larger building positions if there is not enough S3x3 positions
		positions = append(positions, buildPos[scl.S5x3]...)
	}
	for _, pos := range positions {
		if buildings.CloserThan(math.Sqrt2, pos).Exists() {
			continue
		}

		bps := b.GetBuildingPoints(pos, size)
		if !b.CheckPoints(bps, scl.IsNoCreep) {
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
		}
		// Find SCV to continue work if disrupted
		if building.FindAssignedBuilder(builders) == nil && enemies.CanAttack(building, 0).Empty() {
			scv := b.GetSCV(building.Point(), Builders, 45)
			if scv != nil {
				scv.CommandTag(ability.Smart, building.Tag)
			}
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
	geyser := b.VespeneGeysers.Units().CloserThan(10, cc.Point()).First(func(unit *scl.Unit) bool {
		return b.Units[terran.Refinery].CloserThan(1, unit.Point()).Len() == 0
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
		if (node.Premise == nil || node.Premise(b)) && inLimits && (canBuy || node.WaitRes) {
			if !canBuy && node.WaitRes {
				// reserve money for building
				b.DeductResources(node.Ability)
				log.Info(node.Ability)
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

func (b *bot) Upgrades() {
	if eng := b.Units[terran.EngineeringBay].First(scl.Ready, scl.Idle); eng != nil {
		b.RequestAvailableAbilities(true, eng) // request abilities again because we want to ignore resource reqs
		for _, a := range []api.AbilityID{
			ability.Research_TerranInfantryWeaponsLevel1, ability.Research_TerranInfantryArmorLevel1,
			ability.Research_TerranInfantryWeaponsLevel2, ability.Research_TerranInfantryArmorLevel2,
			ability.Research_TerranInfantryWeaponsLevel3, ability.Research_TerranInfantryArmorLevel3,
		} {
			if eng.HasAbility(a) {
				if b.CanBuy(a) {
					eng.Command(a)
				} else {
					// reserve money for upgrade
					b.DeductResources(a)
				}
				break
			}
		}
	}
}

func (b *bot) Morph() {
	cc := b.Units[terran.CommandCenter].First(scl.Ready, scl.Idle)
	if cc != nil && b.Orders[ability.Train_Reaper] >= 2 && b.CanBuy(ability.Morph_OrbitalCommand) {
		b.OrderTrain(cc, ability.Morph_OrbitalCommand)
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
	cc := b.Units[terran.OrbitalCommand].First(func(unit *scl.Unit) bool { return unit.Energy >= 50 })
	if cc != nil {
		// Scan
		if b.Orders[ability.Effect_Scan] == 0 && b.EffectPoints(effect.ScannerSweep).Empty() {
			if reaper := b.Groups.Get(Reapers).Units.ClosestTo(b.EnemyStartLoc); reaper != nil {
				if enemy := b.AllEnemyUnits.Units().CanAttack(reaper, 1).ClosestTo(reaper.Point()); enemy != nil {
					if !b.IsVisible(enemy.Point()) {
						pos := enemy.Point().Towards(b.EnemyStartLoc, 10)
						cc.CommandPos(ability.Effect_Scan, pos)
					}
				}
			}
		}
		// Mule
		if cc.Energy >= 75 {
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

func (b *bot) OrderUnits(ccs scl.Units) {
	rax := b.Units[terran.Barracks].First(scl.Ready, scl.Idle)
	if rax != nil && b.CanBuy(ability.Train_Reaper) {
		b.OrderTrain(rax, ability.Train_Reaper)
	}

	cc := ccs.First(scl.Ready, scl.Idle)
	if cc != nil && b.Units[terran.SCV].Len() < scl.MinInt(21*ccs.Len(), 70) && b.CanBuy(ability.Train_SCV) {
		b.OrderTrain(cc, ability.Train_SCV)
	}
}

func (b *bot) Macro() {
	ccs := b.Units.OfType(terran.CommandCenter, terran.OrbitalCommand, terran.PlanetaryFortress)
	b.BuildingsCheck()
	b.Upgrades()
	b.ProcessBuildOrder(RootBuildOrder)
	b.Morph()
	b.Cast()
	b.OrderUnits(ccs)
}
