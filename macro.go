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

func (b *bot) BuildIfCan(aid api.AbilityID, size scl.BuildingSize, limit, active int) bool {
	buildings := b.Units.Units().Filter(scl.Structure)
	if b.CanBuild(aid, limit, active) {
		enemies := b.AllEnemyUnits.Units()
		for _, pos := range buildPos[size] {
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
		if size == scl.S3x3 {
			log.Debugf("%d: Trying bigger size for 3x3", b.Loop)
			return b.BuildIfCan(aid, scl.S5x3, limit, active)
		}
	}
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

func (b *bot) OrderBuildings(ccs scl.Units) {
	supCount := b.Units.OfType(terran.SupplyDepot, terran.SupplyDepotLowered).Filter(scl.Ready).Len()

	// Supplies
	if b.FoodLeft < 6+b.FoodUsed/20 && b.FoodCap < 200 {
		b.BuildIfCan(ability.Build_SupplyDepot, scl.S2x2, 30, 1+b.FoodUsed/50)
	}

	// First barrack
	if supCount > 0 && b.CanBuild(ability.Build_Barracks, 1, 1) {
		b.BuildFirstBarrack()
	}

	// Refineries
	raxPending := b.Pending(ability.Build_Barracks)
	if b.CanBuy(ability.Build_Refinery) && (raxPending > 0 && b.Pending(ability.Build_Refinery) == 0 ||
		raxPending >= 3 && b.Pending(ability.Build_Refinery) >= 1) && b.Orders[ability.Build_Refinery] < 2 {
		if cc := ccs.First(scl.Ready); cc != nil {
			b.BuildRefinery(cc)
		}
	}

	// More barracks
	if supCount > 0 {
		b.BuildIfCan(ability.Build_Barracks, scl.S5x3, 3*ccs.Len(), 3)
	}

	// Spam CCs =)
	b.BuildIfCan(ability.Build_CommandCenter, scl.S5x5, buildPos[scl.S5x5].Len(), 1)
}

func (b *bot) Morph() {
	cc := b.Units[terran.CommandCenter].First(scl.Ready, scl.Idle)
	// b.CanBuy(ability.Morph_OrbitalCommand) requires 550 minerals?
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
	cc := ccs.First(scl.Ready, scl.Idle)
	if cc != nil && b.Units[terran.SCV].Len() < scl.MinInt(21*ccs.Len(), 70) && b.CanBuy(ability.Train_SCV) {
		b.OrderTrain(cc, ability.Train_SCV)
	}

	rax := b.Units[terran.Barracks].First(scl.Ready, scl.Idle)
	if rax != nil && b.CanBuy(ability.Train_Reaper) {
		b.OrderTrain(rax, ability.Train_Reaper)
	}
}

func (b *bot) Macro() {
	ccs := b.Units.OfType(terran.CommandCenter, terran.OrbitalCommand, terran.PlanetaryFortress)
	b.BuildingsCheck()
	b.OrderBuildings(ccs)
	b.Morph()
	b.Cast()
	b.OrderUnits(ccs)
}
