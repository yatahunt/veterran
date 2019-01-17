package main

import (
	"bitbucket.org/aisee/sc2lib"
	"github.com/chippydip/go-sc2ai/enums/ability"
	"github.com/chippydip/go-sc2ai/enums/terran"
	"math"
)

func (b *bot) OnUnitCreated(unit *scl.Unit) {
	if unit.UnitType == terran.SCV {
		b.Groups.Add(Miners, unit)
		return
	}
	if unit.UnitType == terran.Reaper {
		b.Groups.Add(Reapers, unit)
		return
	}
	if unit.UnitType == terran.Cyclone {
		b.Groups.Add(Cyclones, unit)
		return
	}
	if unit.UnitType == terran.WidowMine {
		b.Groups.Add(Mines, unit)
		return
	}
	if unit.UnitType == terran.SiegeTank || unit.UnitType == terran.SiegeTankSieged {
		b.Groups.Add(Tanks, unit)
		return
	}
	if unit.UnitType == terran.Raven {
		b.Groups.Add(Ravens, unit)
		return
	}
	if unit.UnitType == terran.Battlecruiser {
		b.Groups.Add(Battlecruisers, unit)
		return
	}
	if unit.UnitType == terran.CommandCenter {
		findTurretPositionFor = unit
		// No return! Add it to UnderConstruction group if needed
	}
	if unit.IsStructure() && unit.BuildProgress < 1 {
		b.Groups.Add(UnderConstruction, unit)
		return
	}
}

func (b *bot) Builders() {
	builders := b.Groups.Get(Builders).Units
	enemies := b.EnemyUnits.Units()
	for _, u := range builders {
		enemy := enemies.First(func(unit *scl.Unit) bool {
			return unit.GroundDPS() > 5 && unit.InRange(u, 0.5)
		})
		if enemy != nil || u.Hits < 21 {
			u.Command(ability.Halt_TerranBuild)
			u.CommandQueue(ability.Stop_Stop_4)
		}
	}

	// Move idle or misused builders into miners
	idleBuilder := b.Groups.Get(Builders).Units.First(func(unit *scl.Unit) bool {
		return unit.IsIdle() || unit.IsGathering() || unit.IsReturning()
	})
	if idleBuilder != nil {
		b.Groups.Add(Miners, idleBuilder)
	}
}

func (b *bot) Repair() {
	reps := append(b.Groups.Get(Repairers).Units, b.Groups.Get(UnitHealers).Units...)
	for _, u := range reps {
		if u.Hits < 25 || u.IsIdle() {
			b.Groups.Add(Miners, u)
		}
	}

	if b.Minerals == 0 {
		return
	}

	// Repairers
	buildings := b.Groups.Get(Buildings).Units
	for _, building := range buildings {
		ars := building.FindAssignedRepairers(reps)
		maxArs := int(building.Radius * 3)
		buildingIsDamaged := building.Health < building.HealthMax
		noReps := ars.Empty()
		allRepairing := ars.Len() == ars.CanAttack(building, 0).Len()
		lessThanMaxAssigned := ars.Len() < maxArs
		healthDecreasing := building.HPS > 0
		if buildingIsDamaged && (noReps || (allRepairing && lessThanMaxAssigned && healthDecreasing)) {
			rep := b.GetSCV(building.Point(), Repairers, 45)
			if rep != nil {
				rep.CommandTag(ability.Effect_Repair_SCV, building.Tag)
			}
		}
	}

	// ScvHealer
	ccs := b.Units.OfType(terran.CommandCenter, terran.OrbitalCommand, terran.PlanetaryFortress)
	healer := b.Groups.Get(ScvHealer).Units.First()
	damagedSCVs := b.Units[terran.SCV].Filter(func(unit *scl.Unit) bool {
		return unit.Health < unit.HealthMax && ccs.CloserThan(scl.ResourceSpreadDistance, unit.Point()).Exists()
	})
	if damagedSCVs.Exists() && damagedSCVs[0] != healer {
		if healer == nil {
			healer = b.GetSCV(damagedSCVs.Center(), ScvHealer, 45)
		}
		if healer != nil && healer.TargetAbility() != ability.Effect_Repair_SCV {
			healer.CommandTag(ability.Effect_Repair_SCV, damagedSCVs.ClosestTo(healer.Point()).Tag)
		}
	} else if healer != nil {
		b.Groups.Add(Miners, healer)
	}

	// UnitHealers
	mechs := b.Groups.Get(MechHealing).Units
	for _, mech := range mechs {
		if mech.Health == mech.HealthMax {
			b.OnUnitCreated(mech) // Add to corresponding group
			continue
		}
		ars := mech.FindAssignedRepairers(reps)
		maxArs := int(mech.Radius * 4)
		if ars.Len() < maxArs {
			rep := b.GetSCV(mech.Point(), UnitHealers, 45)
			if rep != nil {
				rep.CommandTag(ability.Effect_Repair_SCV, mech.Tag)
			}
		}
	}
}

func (b *bot) Scout() {
	scv := b.Groups.Get(Scout).Units.First()
	if b.EnemyStartLocs.Len() > 1 && scv == nil && b.Loop < 60 {
		scv = b.GetSCV(b.EnemyStartLocs[0], Scout, 45)
		if scv != nil {
			scv.CommandPos(ability.Move, b.EnemyStartLocs[0])
		}
		return
	}

	if scv != nil {
		if scv.IsIdle() {
			// Check N-1 positions
			for _, p := range b.EnemyStartLocs[:b.EnemyStartLocs.Len()-1] {
				if b.IsExplored(p) {
					continue
				}
				scv.CommandPos(ability.Move, p)
				return
			}
			// If N-1 checked and not found, then N is EnemyStartLoc
			b.RecalcEnemyStartLoc(b.EnemyStartLocs[b.EnemyStartLocs.Len()-1])
			b.Groups.Add(ScoutBase, scv) // promote scout
			return
		}

		if buildings := b.EnemyUnits.Units().Filter(scl.Structure); buildings.Exists() {
			for _, p := range b.EnemyStartLocs[:b.EnemyStartLocs.Len()-1] {
				if buildings.CloserThan(20, p).Exists() {
					b.RecalcEnemyStartLoc(p)
					b.Groups.Add(ScoutBase, scv) // promote scout
					return
				}
			}
		}
	}
}

func (b *bot) ScoutBase() {
	if b.Loop > 2688 { // 2:00
		return
	}

	scv := b.Groups.Get(ScoutBase).Units.First()
	if b.EnemyStartLocs.Len() <= 1 && scv == nil && b.Loop < 60 {
		scv = b.GetSCV(b.EnemyStartLoc, Scout, 45)
		if scv != nil {
			b.Groups.Add(ScoutBase, scv)
		}
	}
	if scv == nil {
		return
	}

	enemies := b.AllEnemyUnits.Units().Filter(scl.DpsGt5)
	if enemies.Exists() || b.Loop > 2016 { // 1:30
		b.Groups.Add(Miners, scv) // dismiss scout
	}

	vec := (scv.Point() - b.EnemyStartLoc).Norm().Rotate(math.Pi / 10)
	pos := b.EnemyStartLoc + vec*10
	scv.CommandPos(ability.Move, pos)
}

func (b *bot) Miners() {
	enemies := b.EnemyUnits.Units().Filter(scl.DpsGt5)
	miners := b.Groups.Get(Miners).Units
	for _, miner := range miners {
		if enemies.CloserThan(safeBuildRange, miner.Point()).Exists() {
			b.Groups.Add(MinersRetreat, miner)
		}
	}

	// Retreat
	mrs := b.Groups.Get(MinersRetreat).Units
	for _, miner := range mrs {
		if enemies.CanAttack(miner, safeBuildRange).Empty() {
			b.Groups.Add(Miners, miner)
			continue
		}
		miner.GroundFallback(enemies, 2, b.HomePaths)
	}

	if b.Loop%b.FramesPerOrder != 0 {
		// try to fix destribution bug. Might be caused by AssignedHarvesters lagging
		return
	}
	// Std miners handler
	miners = b.Groups.Get(Miners).Units
	ccs := b.Units.OfType(terran.CommandCenter, terran.OrbitalCommand, terran.PlanetaryFortress).
		Filter(func(unit *scl.Unit) bool {
			return unit.IsReady() && enemies.CanAttack(unit, 0).Empty()
		})
	b.HandleMiners(miners, ccs, 1)

	// If there is ready unsaturated refinery and an scv gathering, send it there
	refs := b.Units[terran.Refinery]
	if b.Minerals / 2 > b.Vespene {
		ref := refs.First(func(unit *scl.Unit) bool { return unit.IsReady() && unit.AssignedHarvesters < 3 })
		if ref != nil {
			// Get scv gathering minerals
			mfs := b.MineralFields.Units()
			scv := b.Groups.Get(Miners).Units.Filter(func(unit *scl.Unit) bool {
				return unit.IsGathering() && unit.IsCloserThan(scl.ResourceSpreadDistance, ref) &&
					mfs.ByTag(unit.TargetTag()) != nil
			}).ClosestTo(ref.Point())
			if scv != nil {
				scv.CommandTag(ability.Smart, ref.Tag)
			}
		}
	} else if b.Minerals < b.Vespene && refs.Exists() {
		mfs := b.MineralFields.Units()
		scv := b.Groups.Get(Miners).Units.First(func(unit *scl.Unit) bool {
			tag := unit.TargetTag()
			return unit.IsGathering() && refs.ByTag(tag) != nil
		})
		if scv != nil {
			scv.CommandTag(ability.Smart, mfs.ClosestTo(scv.Point()).Tag)
		}
	}
}

func (b *bot) Roles() {
	b.Builders()
	b.Repair()
	b.Scout()
	b.ScoutBase()
	b.Miners()
}
