package main

import (
	"bitbucket.org/aisee/sc2lib"
	"github.com/chippydip/go-sc2ai/api"
	"github.com/chippydip/go-sc2ai/enums/ability"
	"github.com/chippydip/go-sc2ai/enums/protoss"
	"github.com/chippydip/go-sc2ai/enums/terran"
	"github.com/chippydip/go-sc2ai/enums/zerg"
	"math"
)

func (b *bot) OnUnitCreated(unit *scl.Unit) {
	if unit.UnitType == terran.SCV {
		b.Groups.Add(Miners, unit)
		return
	}
	if unit.UnitType == terran.Marine {
		b.Groups.Add(Marines, unit)
		return
	}
	if unit.UnitType == terran.Reaper {
		b.Groups.Add(Reapers, unit)
		return
	}
	if unit.UnitType == terran.WidowMine {
		b.Groups.Add(WidowMines, unit)
		return
	}
	if unit.UnitType == terran.Hellion {
		b.Groups.Add(Hellions, unit)
		return
	}
	if unit.UnitType == terran.Cyclone {
		b.Groups.Add(Cyclones, unit)
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
		return unit.IsIdle() || unit.IsGathering() || unit.IsReturning() || (unit.IsMoving() && unit.TargetTag() != 0)
	})
	if idleBuilder != nil {
		b.Groups.Add(Miners, idleBuilder)
	}
}

func (b *bot) Repair() {
	reps := append(b.Groups.Get(Repairers).Units, b.Groups.Get(UnitHealers).Units...)
	for _, u := range reps {
		if u.Hits < 25 || u.IsIdle() || u.IsGathering() || u.IsReturning() || (u.IsMoving() && u.TargetTag() != 0) {
			b.Groups.Add(Miners, u)
		}
	}

	if b.Minerals == 0 || workerRush {
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

func (b *bot) DoubleHeal() {
	for key, group := range doubleHealers {
		scvs := b.Groups.Get(group).Units
		if scvs.Len() < 2 || (scvs[0].Hits == 45 && scvs[1].Hits == 45) ||
			scvs[0].TargetAbility() != ability.Effect_Repair_SCV ||
			scvs[1].TargetAbility() != ability.Effect_Repair_SCV {
			b.Groups.Add(Miners, scvs...)
			if len(doubleHealers) > key+1 {
				doubleHealers = append(doubleHealers[:key], doubleHealers[key+1:]...)
			} else {
				doubleHealers = doubleHealers[:key]
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
		// Workers rush
		if b.EnemyUnits.OfType(terran.SCV, zerg.Drone, protoss.Probe).FurtherThan(40, b.EnemyStartLoc).Len() > 3 {
			b.Groups.Add(Miners, scv)
			return
		}

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
			playDefensive = true // we don't know what enemy is doing
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
	if b.EnemyStartLocs.Len() <= 1 && scv == nil && !workerRush && b.Loop > 896 && b.Loop < 906 { // 0:50
		scv = b.GetSCV(b.EnemyStartLoc, Scout, 45)
		if scv != nil {
			b.Groups.Add(ScoutBase, scv)
		}
	}
	if scv == nil {
		return
	}

	// Workers rush
	if b.EnemyUnits.OfType(terran.SCV, zerg.Drone, protoss.Probe).FurtherThan(40, b.EnemyStartLoc).Len() > 3 {
		b.Groups.Add(Miners, scv)
		return
	}

	enemies := b.AllEnemyUnits.Units().Filter(scl.DpsGt5)
	if enemies.Exists() || b.Loop > 2240 { // 1:40
		b.Groups.Add(Miners, scv) // dismiss scout

		if b.EnemyRace == api.Race_Terran {
			if b.AllEnemyUnits[terran.Barracks].Len() >= 3 {
				playDefensive = true
			}
		}
		if b.EnemyRace == api.Race_Zerg {
			if b.AllEnemyUnits[zerg.SpawningPool].First(scl.Ready) != nil || b.AllEnemyUnits[zerg.Zergling].Exists() {
				playDefensive = true
			}
		}
		if b.EnemyRace == api.Race_Protoss {
			if b.AllEnemyUnits[protoss.Gateway].Len() >= 2 {
				playDefensive = true
			}
		}
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
	if b.Minerals > 200 && b.Minerals/2 > b.Vespene {
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
	} else if b.Vespene > 200 && b.Minerals < b.Vespene && refs.Exists() {
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
	b.DoubleHeal()
	b.Scout()
	b.ScoutBase()
	b.Miners()
	b.BuildingsCheck()
}
