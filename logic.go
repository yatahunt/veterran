package main

import (
	"bitbucket.org/AiSee/sc2lib"
	"github.com/chippydip/go-sc2ai/api"
	"github.com/chippydip/go-sc2ai/enums/ability"
	"github.com/chippydip/go-sc2ai/enums/protoss"
	"github.com/chippydip/go-sc2ai/enums/terran"
	"github.com/chippydip/go-sc2ai/enums/zerg"
)

func (b *bot) heightAt(p scl.Point) byte {
	m := b.info.GameInfo().StartRaw.TerrainHeight
	// m.BitsPerPixel == 8

	addr := int(p.X()) + int(p.Y())*int(m.Size_.X)
	if addr > len(m.Data)-1 {
		return 0
	}
	return m.Data[addr]
}

func (b *bot) isBuildable(p scl.Point) bool {
	m := b.info.GameInfo().StartRaw.PlacementGrid

	addr := int(p.X()) + int(p.Y())*int(m.Size_.X)
	if addr > len(m.Data)-1 || p.X() < 0 || p.Y() < 0 {
		return false
	}
	return m.Data[addr] != 0
}

func (b *bot) is3x3buildable(pos scl.Point) bool {
	if !b.isBuildable(pos) {
		return false
	}
	for _, p := range pos.Neighbours8(1) {
		if !b.isBuildable(p) {
			return false
		}
	}
	return true
}

func (b *bot) getSCV() *scl.Unit {
	return b.units[terran.SCV].First(scl.Gathering)
}

func (b *bot) findBuildingsPositions() {
	homeMinerals := b.mineralFields.Units().CloserThan(10, b.startLocation)
	if homeMinerals.Len() == 0 {
		return // This should not happen
	}
	vec := homeMinerals.Center().Dir(b.startLocation)
	pos := b.startLocation + vec*3.5
	b.positionsForSupplies.Add(pos)
	b.positionsForSupplies.Add(pos.Neighbours4(2)...)

	pos = b.enemyStartLocation.Towards(b.startLocation, 25)
	pos = pos.Closest(b.baseLocations).Towards(b.startLocation, 1)

	pfb := []*api.RequestQueryBuildingPlacement{{
		AbilityId: ability.Build_Barracks,
		TargetPos: pos.To2D()}}
	for _, np := range pos.Neighbours8(4) {
		if b.isBuildable(np) {
			pfb = append(pfb, &api.RequestQueryBuildingPlacement{
				AbilityId: ability.Build_Barracks,
				TargetPos: np.To2D()})
		}
	}
	resp := b.info.Query(api.RequestQuery{Placements: pfb, IgnoreResourceRequirements: true})
	for key, result := range resp.Placements {
		if result.Result == api.ActionResult_Success {
			b.positionsForBarracks.Add(scl.Pt2(pfb[key].TargetPos))
		}
	}
}

func (b *bot) strategy() {
	// Buildings
	suppliesCount := b.units.OfType(terran.SupplyDepot, terran.SupplyDepotLowered).Len()
	if b.canBuy(ability.Build_SupplyDepot) && b.orders[ability.Build_SupplyDepot] == 0 &&
		suppliesCount < b.positionsForSupplies.Len() && b.foodLeft < 6 {
		pos := b.positionsForSupplies[suppliesCount]
		if scv := b.getSCV(); scv != nil {
			scv.CommandPos(&b.actions, ability.Build_SupplyDepot, pos)
		}
	}
	raxPending := b.units[terran.Barracks].Len()
	if b.canBuy(ability.Build_Barracks) && raxPending < 3 && b.positionsForBarracks.Len() > raxPending {
		pos := b.positionsForBarracks[raxPending]
		scv := b.units[terran.SCV].ByTag(b.builder2)
		if raxPending == 0 || raxPending == 2 {
			scv = b.units[terran.SCV].ByTag(b.builder1)
		}
		if scv == nil {
			scv = b.getSCV()
		}
		if scv != nil {
			scv.CommandPos(&b.actions, ability.Build_Barracks, pos)
		}
		return
	}
	if b.canBuy(ability.Build_Refinery) && (raxPending == 1 && b.units[terran.Refinery].Len() == 0 ||
		raxPending == 3 && b.units[terran.Refinery].Len() == 1) {
		// Find first geyser that is close to my base, but it doesn't have Refinery on top of it
		geyser := b.vespeneGeysers.Units().CloserThan(10, b.startLocation).First(func(unit *scl.Unit) bool {
			return b.units[terran.Refinery].CloserThan(1, unit.Point()).Len() == 0
		})
		if scv := b.getSCV(); scv != nil && geyser != nil {
			scv.CommandTag(&b.actions, ability.Build_Refinery, geyser.Tag)
		}
	}

	// Morph
	cc := b.units[terran.CommandCenter].First(scl.Ready, scl.Idle)
	// b.CanBuy(ability.Morph_OrbitalCommand) requires 550 minerals?
	if cc != nil && b.orders[ability.Train_Reaper] >= 2 && b.minerals >= 150 {
		cc.Command(&b.actions, ability.Morph_OrbitalCommand)
	}
	if supply := b.units[terran.SupplyDepot].First(); supply != nil {
		supply.Command(&b.actions, ability.Morph_SupplyDepot_Lower)
	}

	// Cast
	cc = b.units[terran.OrbitalCommand].First(func(unit *scl.Unit) bool { return unit.Energy >= 50 })
	if cc != nil {
		if homeMineral := b.mineralFields.Units().CloserThan(10, b.startLocation).First(); homeMineral != nil {
			cc.CommandTag(&b.actions, ability.Effect_CalldownMULE, homeMineral.Tag)
		}
	}

	// Units
	ccs := b.units.OfType(terran.CommandCenter, terran.OrbitalCommand, terran.PlanetaryFortress)
	cc = ccs.First(scl.Ready, scl.Idle)
	if cc != nil && b.units[terran.SCV].Len() < 20 && b.canBuy(ability.Train_SCV) {
		cc.Command(&b.actions, ability.Train_SCV)
		return
	}
	rax := b.units[terran.Barracks].First(scl.Ready, scl.Idle)
	if rax != nil && b.canBuy(ability.Train_Reaper) {
		rax.Command(&b.actions, ability.Train_Reaper)
	}
}

func (b *bot) tactics() {
	// If there is idle scv, order it to gather minerals
	if scv := b.units[terran.SCV].First(scl.Idle); scv != nil {
		if homeMineral := b.mineralFields.Units().CloserThan(10, b.startLocation).First(); homeMineral != nil {
			scv.CommandTag(&b.actions, ability.Harvest_Gather_SCV, homeMineral.Tag)
		}
	}
	// Don't issue orders too often, or game won't be able to react
	if b.loop%6 == 0 {
		// If there is ready unsaturated refinery and an scv gathering, send it there
		refinery := b.units[terran.Refinery].
			First(func(unit *scl.Unit) bool { return unit.IsReady() && unit.AssignedHarvesters < 3 })
		if refinery != nil {
			if scv := b.getSCV(); scv != nil {
				scv.CommandTag(&b.actions, ability.Harvest_Gather_SCV, refinery.Tag)
			}
		}
	}

	if b.loop == 224 { // 10 sec
		scv := b.getSCV()
		pos := b.positionsForBarracks[0]
		scv.CommandPos(&b.actions, ability.Move, pos)
		b.builder1 = scv.Tag
	}
	if b.loop == 672 { // 30 sec
		scv := b.getSCV()
		pos := b.positionsForBarracks[1]
		scv.CommandPos(&b.actions, ability.Move, pos)
		b.builder2 = scv.Tag
	}

	b.okTargets = nil
	b.goodTargets = nil
	for _, units := range b.enemyUnits {
		for _, unit := range units {
			if !unit.IsFlying && unit.IsNot(zerg.Larva, zerg.Egg, protoss.AdeptPhaseShift) {
				b.okTargets.Add(unit)
				if !unit.IsStructure() {
					b.goodTargets.Add(unit)
				}
			}
		}
	}

	reapers := b.units[terran.Reaper]
	if b.okTargets.Empty() {
		reapers.CommandPos(&b.actions, ability.Attack, b.enemyStartLocation)
	} else {
		for _, reaper := range reapers {
			// retreat
			if b.retreat[reaper.Tag] && reaper.Health > 50 {
				delete(b.retreat, reaper.Tag)
			}
			if reaper.Health < 21 || b.retreat[reaper.Tag] {
				b.retreat[reaper.Tag] = true
				reaper.CommandPos(&b.actions, ability.Move, b.positionsForBarracks[0])
				continue
			}

			// Keep range
			// Weapon is recharging
			if reaper.WeaponCooldown > 1 {
				// There is an enemy
				if closestEnemy := b.goodTargets.ClosestTo(reaper.Point()); closestEnemy != nil {
					// And it is closer than shooting distance - 0.5
					if reaper.InRange(closestEnemy, -0.5) {
						// Retreat a little
						reaper.CommandPos(&b.actions, ability.Move, b.positionsForBarracks[0])
						continue
					}
				}
			}

			// Attack
			if b.goodTargets.Exists() {
				target := b.goodTargets.ClosestTo(reaper.Point())
				// Snapshots couldn't be targeted using tags
				if reaper.Point().Dist2(target.Point()) > 4*4 && target.DisplayType != api.DisplayType_Snapshot {
					// If target is far, attack it as unit, ling will run ignoring everything else
					reaper.CommandTag(&b.actions, ability.Attack, target.Tag)
				} else {
					// Attack as position, ling will choose best target around
					reaper.CommandPos(&b.actions, ability.Attack, target.Point())
				}
			} else {
				target := b.okTargets.ClosestTo(reaper.Point())
				reaper.CommandPos(&b.actions, ability.Attack, target.Point())
			}
		}
	}
}
