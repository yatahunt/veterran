package roles

import (
	"bitbucket.org/aisee/veterran/bot"
	"github.com/aiseeq/s2l/lib/point"
	"github.com/aiseeq/s2l/lib/scl"
	"github.com/aiseeq/s2l/protocol/enums/ability"
	"github.com/aiseeq/s2l/protocol/enums/effect"
	"github.com/aiseeq/s2l/protocol/enums/protoss"
	"github.com/aiseeq/s2l/protocol/enums/terran"
	"github.com/aiseeq/s2l/protocol/enums/zerg"
	"math"
)

var B *bot.Bot

func BuildingsCheck() {
	builders := B.Groups.Get(bot.Builders).Units
	buildings := B.Groups.Get(bot.UnderConstruction).Units
	enemies := B.Enemies.Visible.Filter(scl.DpsGt5)
	// This is const. Move somewhere else?
	addonsTypes := append(B.U.UnitAliases.For(terran.Reactor), B.U.UnitAliases.For(terran.TechLab)...)
	for _, building := range buildings {
		if building.BuildProgress == 1 {
			switch building.UnitType {
			case terran.Barracks:
				fallthrough
			case terran.Factory:
				building.CommandPos(ability.Rally_Building, B.Ramps.My.Top+B.Ramps.My.Vec*3)
				B.Groups.Add(bot.Buildings, building)
			default:
				B.Groups.Add(bot.Buildings, building) // And remove from current group
			}
			continue
		}

		// Cancel building if it will be destroyed soon
		if building.HPS*2.5 > building.Hits {
			building.Command(ability.Cancel_BuildInProgress)
			continue
		}

		// Find SCV to continue work if disrupted
		if building.FindAssignedBuilder(builders) == nil &&
			enemies.CanAttack(building, 0).Empty() &&
			!addonsTypes.Contain(building.UnitType) {
			scv := bot.GetSCV(building, bot.Builders, 45)
			if scv != nil && B.Enemies.Visible.CanAttack(scv, 0.5).Empty() {
				scv.CommandTag(ability.Smart, building.Tag)
			}
		}

		// Cancel refinery if worker rush is detected and don't build new until enemy is gone
		if B.WorkerRush && building.UnitType == terran.Refinery {
			building.Command(ability.Cancel_BuildInProgress)
		}
	}
	if B.WorkerRush {
		// Cancel second barrack if worker rush
		barracks := B.Units.My[terran.Barracks]
		if barracks.Len() > 1 {
			barracks.Min(func(unit *scl.Unit) float64 { return float64(unit.BuildProgress) }).
				Command(ability.Cancel_BuildInProgress)
		}
	}
}

func Build() {
	builders := B.Groups.Get(bot.Builders).Units
	enemies := B.Enemies.Visible.Filter(scl.DpsGt5)
	for _, u := range builders {
		if enemies.CanAttack(u, 2).Exists() || u.Hits < 21 {
			u.Command(ability.Halt_TerranBuild)
			u.CommandQueue(ability.Stop_Stop)
		}
		if B.Loop < scl.TimeToLoop(1, 30) && B.Enemies.Visible.CanAttack(u, 0.5).Exists() {
			// Protect builder from worker rushers
			u.Command(ability.Halt_TerranBuild)
			mfs := B.Units.Minerals.All().CloserThan(scl.ResourceSpreadDistance, B.Locs.MyStart)
			if mfs.Exists() {
				u.CommandTagQueue(ability.Smart, mfs.ClosestTo(u).Tag)
			}
		}
	}

	// Move idle or misused builders into miners
	idleBuilder := B.Groups.Get(bot.Builders).Units.First(func(unit *scl.Unit) bool {
		return unit.IsIdle() || unit.IsGathering() || unit.IsReturning() || (unit.IsMoving() && unit.TargetTag() != 0)
	})
	if idleBuilder != nil {
		B.Groups.Add(bot.Miners, idleBuilder)
	}
}

func Repair() {
	reps := append(B.Groups.Get(bot.Repairers).Units, B.Groups.Get(bot.UnitHealers).Units...)
	for _, u := range reps {
		if u.Hits < 25 || u.IsIdle() || u.IsGathering() || u.IsReturning() || (u.IsMoving() && u.TargetTag() != 0) {
			B.Groups.Add(bot.Miners, u)
		}
	}

	if B.Minerals < 25 || B.WorkerRush {
		return
	}

	// Repairers
	for _, building := range B.Groups.Get(bot.Buildings).Units {
		buildingIsDamaged := building.Health < building.HealthMax
		if !buildingIsDamaged {
			continue
		}

		ars := building.FindAssignedRepairers(reps)
		maxArs := int(building.Radius * 3)
		noReps := ars.Empty()
		allRepairing := ars.Len() == ars.CanAttack(building, 0).Len()
		lessThanMaxAssigned := ars.Len() < maxArs
		healthDecreasing := building.HPS > 0
		if noReps || (allRepairing && lessThanMaxAssigned && healthDecreasing) {
			rep := bot.GetSCV(building, 0, 45)
			if rep != nil && rep.IsSafeToApproach(building) {
				B.Groups.Add(bot.Repairers, rep)
				rep.CommandTag(ability.Effect_Repair_SCV, building.Tag)
			}
		}
	}

	// ScvHealer
	ccs := B.Units.My.OfType(terran.CommandCenter, terran.OrbitalCommand, terran.PlanetaryFortress)
	healer := B.Groups.Get(bot.ScvHealer).Units.First()
	damagedSCVs := B.Units.My[terran.SCV].Filter(func(unit *scl.Unit) bool {
		return unit.Health < unit.HealthMax && ccs.CloserThan(scl.ResourceSpreadDistance, unit).Exists()
	})
	if damagedSCVs.Exists() && damagedSCVs[0] != healer {
		if healer == nil {
			healer = bot.GetSCV(damagedSCVs.Center(), bot.ScvHealer, 45)
		}
		if healer != nil && healer.TargetAbility() != ability.Effect_Repair_SCV {
			healer.CommandTag(ability.Effect_Repair_SCV, damagedSCVs.ClosestTo(healer).Tag)
		}
	} else if healer != nil {
		B.Groups.Add(bot.Miners, healer)
	}

	// UnitHealers
	mechs := B.Groups.Get(bot.MechHealing).Units
	for _, mech := range mechs {
		if mech.Health == mech.HealthMax {
			bot.OnUnitCreated(mech) // Add to corresponding group
			continue
		}
		if mech.IsFlying { // Move flying units away from CC or bunker
			bs := B.Units.My.OfType(terran.CommandCenter, terran.OrbitalCommand, terran.PlanetaryFortress)
			bs.Add(B.Units.My[terran.Bunker]...)
			if b := bs.CloserThan(3, mech).ClosestTo(mech); b != nil {
				mech.CommandPos(ability.Move_Move, b.Towards(mech, 3))
			}
		}
		ars := mech.FindAssignedRepairers(reps)
		maxArs := int(mech.Radius * 4)
		if ars.Len() < maxArs {
			rep := bot.GetSCV(mech, bot.UnitHealers, 45)
			if rep != nil {
				rep.CommandTag(ability.Effect_Repair_SCV, mech.Tag)
			}
		}
	}

	if B.PlayDefensive {
		// Heal units near the bases
		allMechs := B.Units.MyAll.Filter(func(unit *scl.Unit) bool {
			return unit.IsMechanical() && !unit.IsStructure() && unit.IsReady() && unit.Hits < unit.HitsMax
		})
		for _, cc := range B.Units.My.OfType(terran.CommandCenter, terran.OrbitalCommand, terran.PlanetaryFortress) {
			closeMechs := allMechs.CloserThan(scl.ResourceSpreadDistance, cc)
			for _, mech := range closeMechs {
				ars := mech.FindAssignedRepairers(reps)
				maxArs := int(mech.Radius * 4)
				if ars.Len() < maxArs {
					rep := bot.GetSCV(mech, bot.UnitHealers, 45)
					if rep != nil {
						rep.CommandTag(ability.Effect_Repair_SCV, mech.Tag)
					}
				}
			}
		}
	}
}

func DoubleHeal() {
	for key, group := range B.DoubleHealers {
		scvs := B.Groups.Get(group).Units
		enemies := B.Enemies.Visible.Filter(scl.Ground)
		if scvs.Len() < 2 || (scvs[0].Hits == 45 && scvs[1].Hits == 45) ||
			scvs[0].TargetAbility() != ability.Effect_Repair_SCV ||
			scvs[1].TargetAbility() != ability.Effect_Repair_SCV ||
			enemies.CanAttack(scvs[0], 2).Exists() || enemies.CanAttack(scvs[1], 2).Exists() {
			B.Groups.Add(bot.Miners, scvs...)
			if len(B.DoubleHealers) > key+1 {
				B.DoubleHealers = append(B.DoubleHealers[:key], B.DoubleHealers[key+1:]...)
			} else {
				B.DoubleHealers = B.DoubleHealers[:key]
			}
		}
	}
}

func Recon() {
	scv := B.Groups.Get(bot.Scout).Units.First()
	if B.Locs.EnemyStarts.Len() > 1 && scv == nil && B.Loop < 60 {
		scv = bot.GetSCV(B.Locs.EnemyStarts[0], bot.Scout, 45)
		if scv != nil {
			scv.CommandPos(ability.Move, B.Locs.EnemyStarts[0])
		}
		return
	}

	if scv != nil {
		// Workers rush
		// todo: init defence here?
		if B.Units.Enemy.OfType(terran.SCV, zerg.Drone, protoss.Probe).FurtherThan(40, B.Locs.EnemyStart).Len() > 3 {
			B.Groups.Add(bot.Miners, scv)
			return
		}

		if scv.IsIdle() {
			// Check N-1 positions
			for _, p := range B.Locs.EnemyStarts[:B.Locs.EnemyStarts.Len()-1] {
				if B.Grid.IsExplored(p) {
					continue
				}
				scv.CommandPos(ability.Move, p)
				return
			}
			// If N-1 checked and not found, then N is EnemyStartLoc
			bot.RecalcEnemyStartLoc(B.Locs.EnemyStarts[B.Locs.EnemyStarts.Len()-1])
			B.Groups.Add(bot.ScoutBase, scv) // promote scout
			bot.EnableDefensivePlay()        // we don't know what enemy is doing
			return
		}

		if buildings := B.Enemies.Visible.Filter(scl.Structure); buildings.Exists() {
			for _, p := range B.Locs.EnemyStarts[:B.Locs.EnemyStarts.Len()-1] {
				if buildings.CloserThan(20, p).Exists() {
					bot.RecalcEnemyStartLoc(p)
					B.Groups.Add(bot.ScoutBase, scv) // promote scout
					return
				}
			}
		}
	}
}

func ReconBase() {
	if B.Loop > scl.TimeToLoop(3, 0) {
		return
	}

	scv := B.Groups.Get(bot.ScoutBase).Units.First()
	if scv == nil && B.Locs.EnemyStarts.Len() <= 1 && !B.WorkerRush && B.Loop > scl.TimeToLoop(0, 45) &&
		B.Loop < scl.TimeToLoop(1, 0) {
		// 0:50 hire scout
		scv = bot.GetSCV(B.Locs.EnemyStart, bot.Scout, 45)
		if scv != nil {
			B.Groups.Add(bot.ScoutBase, scv)
		}
	}
	if scv == nil {
		return
	}

	// Workers rush
	if B.Units.Enemy.OfType(terran.SCV, zerg.Drone, protoss.Probe).FurtherThan(40, B.Locs.EnemyStart).Len() > 3 {
		// todo: init defence here?
		B.Groups.Add(bot.Miners, scv)
		return
	}

	enemies := B.Enemies.All.Filter(scl.DpsGt5).CanAttack(scv, 2)
	if enemies.Exists() || B.Loop > scl.TimeToLoop(2, 40) {
		B.Groups.Add(bot.Miners, scv) // dismiss scout
	}

	vec := (scv.Point() - B.Locs.EnemyStart).Norm().Rotate(math.Pi / 10)
	pos := B.Locs.EnemyStart + vec*10
	if (pos - scv.TargetPos()).Len() >= 1 {
		scv.CommandPos(ability.Move, pos)
	}
}

func ReconHellion() {
	/*hellion := B.Groups.Get(bot.HellionScout).Units.First()
	if hellion == nil {
		hellions := B.Groups.Get(bot.Hellions).Units
		if hellions.Exists() && (!B.LingRush || hellions.Len() > 2) {
			hellion = hellions.ClosestTo(B.Locs.EnemyStart)
			B.Groups.Add(bot.HellionScout, hellion)
		} else {
			return
		}
	}
	if hellion.UnitType == terran.HellionTank {
		hellion.Command(ability.Morph_Hellion)
		return
	}
	if hellion.IsIdle() {
		// todo: what if all locs are taken?
		// todo: order by dist? Make shortest route?
		// todo: evade enemy forces, but harass workers if no defense
		for _, pos := range B.Locs.EnemyExps {
			if B.IsVisible(pos) ||
				B.Enemies.Visible.CloserThan(3, pos).Exists() ||
				B.Units.MyAll.CloserThan(3, pos).Exists() {
				continue
			}
			hellion.CommandPosQueue(ability.Move, pos)
		}
	}*/
}

func GetLiberatorsCircles() (ps point.Points) {
	for _, e := range B.Obs.RawData.Effects {
		if e.EffectId == effect.LiberatorDefenderZoneSetup || e.EffectId == effect.LiberatorDefenderZone {
			for _, p := range e.Pos {
				ps.Add(point.Pt2(p))
			}
		}
	}
	return
}

func Mine() {
	enemies := B.Enemies.Visible.Filter(scl.DpsGt5)
	miners := B.Groups.Get(bot.Miners).Units

	// Std miners handler
	miners = B.Groups.Get(bot.Miners).Units
	lcs := GetLiberatorsCircles()
	ccs := B.Units.My.OfType(terran.CommandCenter, terran.OrbitalCommand, terran.PlanetaryFortress).
		Filter(func(unit *scl.Unit) bool {
			return unit.IsReady() && enemies.CanAttack(unit, 0).Sum(scl.CmpGroundDPS) < 30 && // one banshee
				(lcs.Empty() || lcs.CloserThan(scl.ResourceSpreadDistance, unit).Empty())
		})
	// Move miners to first gas if not proxy marines build
	if B.Loop < scl.TimeToLoop(1, 10) && len(B.Miners.GasForMiner) < 3 && !B.ProxyMarines {
		if ref := B.Units.My.OfType(B.U.UnitAliases.For(terran.Refinery)...).First(scl.Ready); ref != nil {
			B.RedistributeWorkersToRefineryIfNeeded(ref, miners, 3)
		}
	}
	if B.BruteForce && B.Loop < scl.TimeToLoop(1, 15) && len(B.Miners.GasForMiner) < 6 {
		if ref := B.Units.My.OfType(B.U.UnitAliases.For(terran.Refinery)...).First(func(unit *scl.Unit) bool {
			// Select refinery without miners assigned
			if !unit.IsReady() {
				return false
			}
			for _, gasTag := range B.Miners.GasForMiner {
				if gasTag == unit.Tag {
					return false
				}
			}
			return true
		}); ref != nil {
			B.RedistributeWorkersToRefineryIfNeeded(ref, miners, 3)
		}
	}
	B.HandleMiners(miners, ccs, enemies, 0.5, B.Locs.MyStart-B.Locs.MyStartMinVec*3, B.TurretsMiningPos)

	mules := B.Groups.Get(bot.Mules).Units.Filter(scl.Idle)
	for _, mule := range mules {
		if mf := B.Units.Minerals.All().ClosestTo(mule); mf != nil {
			mule.CommandTag(ability.Smart, mf.Tag)
		}
	}
}

func TanksOnExpansions() {
	tanks := B.Groups.Get(bot.TanksOnExps).Units
	tanksSieged := tanks.Filter(func(unit *scl.Unit) bool { return unit.UnitType == terran.SiegeTankSieged })
	tanksUnsieged := tanks.Filter(func(unit *scl.Unit) bool { return unit.UnitType == terran.SiegeTank })
	if !B.PlayDefensive {
		// Move all unsieged tanks back to army
		for _, tank := range tanksUnsieged {
			B.Groups.Add(bot.Tanks, tank)
		}
		return
	}
	if B.Enemies.All.Filter(scl.DpsGt5).CloserThan(B.DefensiveRange, B.Locs.MyStart).Exists() {
		return // Enemies are too close already
	}

	bunkers := B.Units.My[terran.Bunker]
	bunker := bunkers.Filter(func(unit *scl.Unit) bool {
		return tanksSieged.CloserThan(5, unit).Empty()
	}).ClosestTo(B.Locs.MyStart)
	for _, tank := range tanksUnsieged {
		if bunker == nil {
			B.Groups.Add(bot.Tanks, tank)
			continue
		}
		ccs := B.Units.My.OfType(terran.CommandCenter, terran.OrbitalCommand, terran.PlanetaryFortress)
		cc := ccs.ClosestTo(bunker)
		pos := bunker.Towards(cc, 1.5)
		if tank.IsFarFrom(pos) {
			tank.CommandPos(ability.Move, pos)
		} else if tank.IsIdle() {
			tank.Command(ability.Morph_SiegeMode)
		}
	}

	candidates := B.Groups.Get(bot.Tanks).Units
	if tanks.Len() < bunkers.Len() && candidates.Exists() && bunker != nil {
		tank := candidates.ClosestTo(bunker)
		B.Groups.Add(bot.TanksOnExps, tank)
	}
}

func Roles(b *bot.Bot) {
	B = b // todo: better
	Build()
	Repair()
	DoubleHeal()
	if B.BruteForce {
		Recon()
		ReconBase()
	}
	// ReconHellion()
	Mine()
	// TanksOnExpansions()
	BuildingsCheck()
}
