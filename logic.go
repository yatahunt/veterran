package main

import (
	"bitbucket.org/AiSee/sc2lib"
	"github.com/chippydip/go-sc2ai/api"
	"github.com/chippydip/go-sc2ai/enums/ability"
	"github.com/chippydip/go-sc2ai/enums/protoss"
	"github.com/chippydip/go-sc2ai/enums/terran"
	"github.com/chippydip/go-sc2ai/enums/zerg"
)

func (b *bot) HeightAt(p scl.Point) byte {
	m := b.Info.GameInfo().StartRaw.TerrainHeight
	// m.BitsPerPixel == 8

	addr := int(p.X()) + int(p.Y())*int(m.Size_.X)
	if addr > len(m.Data)-1 {
		return 0
	}
	return m.Data[addr]
}

func (b *bot) IsBuildable(p scl.Point) bool {
	m := b.Info.GameInfo().StartRaw.PlacementGrid

	addr := int(p.X()) + int(p.Y())*int(m.Size_.X)
	if addr > len(m.Data)-1 || p.X() < 0 || p.Y() < 0 {
		return false
	}
	return m.Data[addr] != 0
}

func (b *bot) Is3x3buildable(pos scl.Point) bool {
	if !b.IsBuildable(pos) {
		return false
	}
	for _, p := range pos.Neighbours8(1) {
		if !b.IsBuildable(p) {
			return false
		}
	}
	return true
}

func (b *bot) GetSCV() *scl.Unit {
	return b.Units[terran.SCV].First(scl.Gathering)
}

func (b *bot) FindBuildingsPositions() {
	homeMinerals := b.MineralFields.Units().CloserThan(10, b.StartLocation)
	if homeMinerals.Len() == 0 {
		return // This should not happen
	}
	vec := homeMinerals.Center().Dir(b.StartLocation)
	pos := b.StartLocation + vec*3.5
	b.PositionsForSupplies.Add(pos)
	b.PositionsForSupplies.Add(pos.Neighbours4(2)...)

	pos = b.EnemyStartLocation.Towards(b.StartLocation, 25)
	pos = pos.Closest(b.BaseLocations).Towards(b.StartLocation, 1)

	pfb := []*api.RequestQueryBuildingPlacement{{
		AbilityId: ability.Build_Barracks,
		TargetPos: pos.To2D()}}
	for _, np := range pos.Neighbours8(4) {
		if b.IsBuildable(np) {
			pfb = append(pfb, &api.RequestQueryBuildingPlacement{
				AbilityId: ability.Build_Barracks,
				TargetPos: np.To2D()})
		}
	}
	resp := b.Info.Query(api.RequestQuery{Placements: pfb, IgnoreResourceRequirements: true})
	for key, result := range resp.Placements {
		if result.Result == api.ActionResult_Success {
			b.PositionsForBarracks.Add(scl.Pt2(pfb[key].TargetPos))
		}
	}
}

func (b *bot) Strategy() {
	// Buildings
	suppliesCount := b.Units.OfType(terran.SupplyDepot, terran.SupplyDepotLowered).Len()
	if b.CanBuy(ability.Build_SupplyDepot) && b.Orders[ability.Build_SupplyDepot] == 0 &&
		suppliesCount < b.PositionsForSupplies.Len() && b.FoodLeft < 6 {
		pos := b.PositionsForSupplies[suppliesCount]
		if scv := b.GetSCV(); scv != nil {
			scv.CommandPos(&b.Actions, ability.Build_SupplyDepot, pos)
		}
	}
	raxPending := b.Units[terran.Barracks].Len()
	if b.CanBuy(ability.Build_Barracks) && raxPending < 3 && b.PositionsForBarracks.Len() > raxPending {
		pos := b.PositionsForBarracks[raxPending]
		scv := b.Units[terran.SCV].ByTag(b.Builder2)
		if raxPending == 0 || raxPending == 2 {
			scv = b.Units[terran.SCV].ByTag(b.Builder1)
		}
		if scv == nil {
			scv = b.GetSCV()
		}
		if scv != nil {
			scv.CommandPos(&b.Actions, ability.Build_Barracks, pos)
		}
		return
	}
	if b.CanBuy(ability.Build_Refinery) && (raxPending == 1 && b.Units[terran.Refinery].Len() == 0 ||
		raxPending == 3 && b.Units[terran.Refinery].Len() == 1) {
		// Find first geyser that is close to my base, but it doesn't have Refinery on top of it
		geyser := b.VespeneGeysers.Units().CloserThan(10, b.StartLocation).First(func(unit *scl.Unit) bool {
			return b.Units[terran.Refinery].CloserThan(1, unit.Point()).Len() == 0
		})
		if scv := b.GetSCV(); scv != nil && geyser != nil {
			scv.CommandTag(&b.Actions, ability.Build_Refinery, geyser.Tag)
		}
	}

	// Morph
	cc := b.Units[terran.CommandCenter].First(scl.Ready, scl.Idle)
	// b.CanBuy(ability.Morph_OrbitalCommand) requires 550 minerals?
	if cc != nil && b.Orders[ability.Train_Reaper] >= 2 && b.Minerals >= 150 {
		cc.Command(&b.Actions, ability.Morph_OrbitalCommand)
	}
	if supply := b.Units[terran.SupplyDepot].First(); supply != nil {
		supply.Command(&b.Actions, ability.Morph_SupplyDepot_Lower)
	}

	// Cast
	cc = b.Units[terran.OrbitalCommand].First(func(unit *scl.Unit) bool { return unit.Energy >= 50 })
	if cc != nil {
		if homeMineral := b.MineralFields.Units().CloserThan(10, b.StartLocation).First(); homeMineral != nil {
			cc.CommandTag(&b.Actions, ability.Effect_CalldownMULE, homeMineral.Tag)
		}
	}

	// Units
	ccs := b.Units.OfType(terran.CommandCenter, terran.OrbitalCommand, terran.PlanetaryFortress)
	cc = ccs.First(scl.Ready, scl.Idle)
	if cc != nil && b.Units[terran.SCV].Len() < 20 && b.CanBuy(ability.Train_SCV) {
		cc.Command(&b.Actions, ability.Train_SCV)
		return
	}
	rax := b.Units[terran.Barracks].First(scl.Ready, scl.Idle)
	if rax != nil && b.CanBuy(ability.Train_Reaper) {
		rax.Command(&b.Actions, ability.Train_Reaper)
	}
}

func (b *bot) Tactics() {
	// If there is idle scv, order it to gather minerals
	if scv := b.Units[terran.SCV].First(scl.Idle); scv != nil {
		if homeMineral := b.MineralFields.Units().CloserThan(10, b.StartLocation).First(); homeMineral != nil {
			scv.CommandTag(&b.Actions, ability.Harvest_Gather_SCV, homeMineral.Tag)
		}
	}
	// Don't issue orders too often, or game won't be able to react
	if b.Loop%6 == 0 {
		// If there is ready unsaturated refinery and an scv gathering, send it there
		refinery := b.Units[terran.Refinery].
			First(func(unit *scl.Unit) bool { return unit.IsReady() && unit.AssignedHarvesters < 3 })
		if refinery != nil {
			if scv := b.GetSCV(); scv != nil {
				scv.CommandTag(&b.Actions, ability.Harvest_Gather_SCV, refinery.Tag)
			}
		}
	}

	if b.Loop == 224 { // 10 sec
		scv := b.GetSCV()
		pos := b.PositionsForBarracks[0]
		scv.CommandPos(&b.Actions, ability.Move, pos)
		b.Builder1 = scv.Tag
	}
	if b.Loop == 672 { // 30 sec
		scv := b.GetSCV()
		pos := b.PositionsForBarracks[1]
		scv.CommandPos(&b.Actions, ability.Move, pos)
		b.Builder2 = scv.Tag
	}

	okTargets := scl.Units{}
	goodTargets := scl.Units{}
	for _, units := range b.EnemyUnits {
		for _, unit := range units {
			if !unit.IsFlying && unit.IsNot(zerg.Larva, zerg.Egg, protoss.AdeptPhaseShift) {
				okTargets.Add(unit)
				if !unit.IsStructure() {
					goodTargets.Add(unit)
				}
			}
		}
	}

	reapers := b.Units[terran.Reaper]
	if okTargets.Empty() {
		reapers.CommandPos(&b.Actions, ability.Attack, b.EnemyStartLocation)
	} else {
		for _, reaper := range reapers {
			// retreat
			if b.Retreat[reaper.Tag] && reaper.Health > 50 {
				delete(b.Retreat, reaper.Tag)
			}
			if reaper.Health < 21 || b.Retreat[reaper.Tag] {
				b.Retreat[reaper.Tag] = true
				reaper.CommandPos(&b.Actions, ability.Move, b.PositionsForBarracks[0])
				continue
			}

			// Keep range
			// Weapon is recharging
			if reaper.WeaponCooldown > 1 {
				// There is an enemy
				if closestEnemy := goodTargets.ClosestTo(reaper.Point()); closestEnemy != nil {
					// And it is closer than shooting distance - 0.5
					if reaper.InRange(closestEnemy, -0.5) {
						// Retreat a little
						reaper.CommandPos(&b.Actions, ability.Move, b.PositionsForBarracks[0])
						continue
					}
				}
			}

			// Attack
			if goodTargets.Exists() {
				target := goodTargets.ClosestTo(reaper.Point())
				// Snapshots couldn't be targeted using tags
				if reaper.Point().Dist2(target.Point()) > 4*4 && target.DisplayType != api.DisplayType_Snapshot {
					// If target is far, attack it as unit, ling will run ignoring everything else
					reaper.CommandTag(&b.Actions, ability.Attack, target.Tag)
				} else {
					// Attack as position, ling will choose best target around
					reaper.CommandPos(&b.Actions, ability.Attack, target.Point())
				}
			} else {
				target := okTargets.ClosestTo(reaper.Point())
				reaper.CommandPos(&b.Actions, ability.Attack, target.Point())
			}
		}
	}
}
