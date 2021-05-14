package roles

import (
	"bitbucket.org/aisee/sc2lib/scl"
	"bitbucket.org/aisee/veterran/bot"
	"github.com/chippydip/go-sc2ai/api"
	"github.com/chippydip/go-sc2ai/enums/ability"
	"github.com/chippydip/go-sc2ai/enums/protoss"
	"github.com/chippydip/go-sc2ai/enums/terran"
	"github.com/chippydip/go-sc2ai/enums/zerg"
	"math"
)

var B = bot.B

func BuildingsCheck() {
	builders := B.Groups.Get(bot.Builders).Units
	buildings := B.Groups.Get(bot.UnderConstruction).Units
	enemies := B.Enemies.Visible.Filter(scl.DpsGt5)
	// This is const. Move somewhere else?
	addonsTypes := append(scl.UnitAliases.For(terran.Reactor), scl.UnitAliases.For(terran.TechLab)...)
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
			if scv != nil {
				scv.CommandTag(ability.Smart, building.Tag)
			}
		}

		// Cancel refinery if worker rush is detected and don't build new until enemy is gone
		if B.WorkerRush && building.UnitType == terran.Refinery {
			building.Command(ability.Cancel)
		}
	}
}

func Build() {
	builders := B.Groups.Get(bot.Builders).Units
	enemies := B.Enemies.Visible
	for _, u := range builders {
		enemy := enemies.First(func(unit *scl.Unit) bool {
			return unit.GroundDPS() > 5 && unit.InRange(u, 2)
		})
		if enemy != nil || u.Hits < 21 {
			u.Command(ability.Halt_TerranBuild)
			u.CommandQueue(ability.Stop_Stop)
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
	buildings := append(B.Groups.Get(bot.Buildings).Units, B.Groups.Get(bot.TanksOnExps).Units...)
	for _, building := range buildings {
		ars := building.FindAssignedRepairers(reps)
		maxArs := int(building.Radius * 3)
		buildingIsDamaged := building.Health < building.HealthMax
		noReps := ars.Empty()
		allRepairing := ars.Len() == ars.CanAttack(building, 0).Len()
		lessThanMaxAssigned := ars.Len() < maxArs
		healthDecreasing := building.HPS > 0
		if buildingIsDamaged && (noReps || (allRepairing && lessThanMaxAssigned && healthDecreasing)) {
			rep := bot.GetSCV(building, bot.Repairers, 45)
			if rep != nil {
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

func DoubleHeal() {
	for key, group := range B.DoubleHealers {
		scvs := B.Groups.Get(group).Units
		if scvs.Len() < 2 || (scvs[0].Hits == 45 && scvs[1].Hits == 45) ||
			scvs[0].TargetAbility() != ability.Effect_Repair_SCV ||
			scvs[1].TargetAbility() != ability.Effect_Repair_SCV {
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
	if B.Loop > 2688 { // 2:00
		return
	}

	scv := B.Groups.Get(bot.ScoutBase).Units.First()
	if B.Locs.EnemyStarts.Len() <= 1 && scv == nil && !B.WorkerRush /*&& !PlayDefensive*/ && B.Loop > 896 && B.Loop < 906 {
		// 0:50
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
		B.Groups.Add(bot.Miners, scv)
		return
	}

	enemies := B.Enemies.All.Filter(scl.DpsGt5)
	if enemies.Exists() || B.Loop > 2240 { // 1:40
		B.Groups.Add(bot.Miners, scv) // dismiss scout

		if B.EnemyRace == api.Race_Terran {
			if B.Units.AllEnemy[terran.Barracks].Len() >= 3 {
				bot.EnableDefensivePlay()
			}
		}
		if B.EnemyRace == api.Race_Zerg {
			if B.Units.AllEnemy[zerg.SpawningPool].First(scl.Ready) != nil || B.Units.AllEnemy[zerg.Zergling].Exists() {
				bot.EnableDefensivePlay()
			}
			if B.Units.AllEnemy[zerg.Zergling].Exists() {
				B.LingRush = true
			}
		}
		if B.EnemyRace == api.Race_Protoss {
			if B.Units.AllEnemy[protoss.Gateway].Len() >= 2 {
				bot.EnableDefensivePlay()
			}
		}
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
				B.Units.My.All().CloserThan(3, pos).Exists() {
				continue
			}
			hellion.CommandPosQueue(ability.Move, pos)
		}
	}*/
}

func Mine() {
	enemies := B.Enemies.Visible.Filter(scl.DpsGt5)
	miners := B.Groups.Get(bot.Miners).Units
	/*for _, miner := range miners {
		if enemies.CloserThan(safeBuildRange, miner).Sum(scl.CmpGroundDamage) > miner.Hits {
			B.Groups.Add(bot.MinersRetreat, miner)
		}
	}

	// Retreat
	mrs := B.Groups.Get(bot.MinersRetreat).Units
	for _, miner := range mrs {
		if enemies.CanAttack(miner, safeBuildRange).Empty() {
			B.Groups.Add(bot.Miners, miner)
			continue
		}
		miner.GroundFallback(enemies, 2, B.HomePaths)
	}*/

	if B.Loop%B.FramesPerOrder != 0 {
		// try to fix destribution bug. Might be caused by AssignedHarvesters lagging
		return
	}
	// Std miners handler
	miners = B.Groups.Get(bot.Miners).Units
	ccs := B.Units.My.OfType(terran.CommandCenter, terran.OrbitalCommand, terran.PlanetaryFortress).
		Filter(func(unit *scl.Unit) bool {
			return unit.IsReady() && enemies.CanAttack(unit, 0).Empty()
		})
	B.HandleMiners(miners, ccs, 1.5) // reserve more vespene

	// If there is ready unsaturated refinery and an scv gathering, send it there
	refs := B.Units.My[terran.Refinery]
	if B.Minerals > scl.MinInt(5, ccs.Len())*100 && B.Minerals/2 > B.Vespene {
		ref := refs.First(func(unit *scl.Unit) bool { return unit.IsReady() && unit.AssignedHarvesters < 3 })
		if ref != nil {
			// Get scv gathering minerals
			mfs := B.Units.Minerals.All()
			scv := B.Groups.Get(bot.Miners).Units.Filter(func(unit *scl.Unit) bool {
				return unit.IsGathering() && unit.IsCloserThan(scl.ResourceSpreadDistance, ref) &&
					mfs.ByTag(unit.TargetTag()) != nil
			}).ClosestTo(ref)
			if scv != nil {
				scv.CommandTag(ability.Smart, ref.Tag)
			}
		}
	} else if B.Vespene > scl.MinInt(5, ccs.Len())*100 && B.Minerals < B.Vespene/2 && refs.Exists() {
		cc := ccs.Filter(func(unit *scl.Unit) bool { return unit.AssignedHarvesters < unit.IdealHarvesters })
		if cc != nil {
			scv := B.Groups.Get(bot.Miners).Units.First(func(unit *scl.Unit) bool {
				tag := unit.TargetTag()
				return unit.IsGathering() && refs.ByTag(tag) != nil
			})
			if scv != nil {
				// scv.CommandTag(ability.Smart, mfs.ClosestTo(scv).Tag)
				scv.Command(ability.Stop_Stop)
			}
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

func Roles() {
	Build()
	Repair()
	DoubleHeal()
	Recon()
	ReconBase()
	ReconHellion()
	Mine()
	TanksOnExpansions()
	BuildingsCheck()
}
