package main

import (
	"bitbucket.org/aisee/sc2lib"
	"github.com/chippydip/go-sc2ai/enums/ability"
	"github.com/chippydip/go-sc2ai/enums/terran"
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
			u.CommandQueue(ability.Stop_Stop)
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
	reps := b.Groups.Get(Repairers).Units
	for _, u := range reps {
		if u.Hits < 45 || u.IsIdle() {
			b.Groups.Add(Miners, u)
		}
	}

	if b.Minerals == 0 {
		return
	}

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

	healer := b.Groups.Get(ScvHealer).Units.First()
	damagedSCVs := b.Units[terran.SCV].Filter(func(unit *scl.Unit) bool { return unit.Health < unit.HealthMax })
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
}

func (b *bot) Scout() {
	if b.EnemyStartLocs.Len() > 1 && b.Loop == 0 {
		scv := b.Groups.Get(Miners).Units.ClosestTo(b.EnemyStartLocs[0])
		if scv != nil {
			b.Groups.Add(Scout, scv)
			scv.CommandPos(ability.Move, b.EnemyStartLocs[0])
		}
		return
	}

	scv := b.Groups.Get(Scout).Units.First()
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
			b.Groups.Add(Miners, scv) // dismiss scout
			return
		}

		if buildings := b.EnemyUnits.Units().Filter(scl.Structure); buildings.Exists() {
			for _, p := range b.EnemyStartLocs[:b.EnemyStartLocs.Len()-1] {
				if buildings.CloserThan(20, p).Exists() {
					b.RecalcEnemyStartLoc(p)
					b.Groups.Add(Miners, scv) // dismiss scout
					return
				}
			}
		}
	}
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
		pos := miner.GroundEvade(enemies, safeBuildRange, b.StartLoc)
		miner.CommandPos(ability.Move, pos)
	}

	if b.Loop%6 != 0 {
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
	refinery := b.Units[terran.Refinery].
		First(func(unit *scl.Unit) bool { return unit.IsReady() && unit.AssignedHarvesters < 3 })
	if refinery != nil && b.Minerals > b.Vespene {
		// Get scv gathering minerals
		mfs := b.MineralFields.Units()
		scv := b.Groups.Get(Miners).Units.Filter(func(unit *scl.Unit) bool {
			return unit.IsGathering() && mfs.ByTag(unit.TargetTag()) != nil
		}).ClosestTo(refinery.Point())
		if scv != nil {
			scv.CommandTag(ability.Smart, refinery.Tag)
		}
	}
}

func (b *bot) Roles() {
	b.Builders()
	b.Repair()
	b.Scout()
	b.Miners()
}
