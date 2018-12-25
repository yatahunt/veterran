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
	"math"
	"math/rand"
)

// todo: строить первый барак с пересадкой и сруза после постройки саплая тем же рабочим
// todo: всё ещё есть проблемы (дёрп) с назначением на продолжение строительства если рабочего убили
// todo: wall closed flag -> no worker defence
// todo: fix morph abilities cost

var workerRush = false
var assault = false
var buildPos = map[scl.BuildingSize]scl.Points{}

const (
	Miners scl.GroupID = iota + 1
	MinersRetreat
	Builders
	Repairers
	ScvHealer
	WorkerRushDefenders
	Scout
	Reapers
	Retreat
	UnderConstruction
	Buildings
	MaxGroup
)
const safeBuildRange = 7

func (b *bot) GetSCV(pos scl.Point, assignGroup scl.GroupID, minHits float64) *scl.Unit {
	scv := b.Groups.Get(Miners).Units.Filter(func(unit *scl.Unit) bool {
		return unit.IsGathering() && unit.Hits >= minHits
	}).ClosestTo(pos)
	if scv != nil {
		b.Groups.Add(assignGroup, scv)
	}
	return scv
}

func (b *bot) AlreadyTraining(abilityID api.AbilityID) int {
	count := 0
	units := b.Units.Units()
	for _, unit := range units {
		if unit.IsStructure() && unit.TargetAbility() == abilityID {
			count++
		}
	}
	return count
}

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

func (b *bot) FindBuildingsPositions() {
	homeMinerals := b.MineralFields.Units().CloserThan(scl.ResourceSpreadDistance, b.StartLoc)
	if homeMinerals.Len() == 0 {
		return // This should not happen
	}
	vec := homeMinerals.Center().Dir(b.StartLoc)
	if vec.Len() == 1 {
		vec = b.StartLoc.Dir(b.MapCenter)
	}

	rp2x2 := b.FindRamp2x2Positions(b.MainRamp)
	rp5x3 := scl.Points{b.FindRampBarracksPosition(b.MainRamp)}
	rbpts := b.GetBuildingPoints(rp5x3[0], scl.S5x3)

	/*pos = b.EnemyStartLoc.Towards(b.StartLoc, 25)
	pos = pos.Closest(b.ExpLocs).Towards(b.StartLoc, 1)

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
			pos5x3.Add(scl.Pt2(pfb[key].TargetPos))
		}
	}*/

	var pf2x2, pf3x3, pf5x3 scl.Points
	slh := b.HeightAt(b.StartLoc)
	start := b.StartLoc + 9
	for y := -3.0; y <= 3; y++ {
		for x := -9.0; x <= 9; x++ {
			pos := start + scl.Pt(3, 2).Mul(x) + scl.Pt(-6, 8).Mul(y)
			if b.HeightAt(pos) == slh && b.IsPosOk(pos, scl.S3x3, 2, scl.IsBuildable) &&
				rbpts.Intersect(b.GetBuildingPoints(pos, scl.S3x3)).Empty() {
				if b.IsPosOk(pos+2-1i, scl.S2x2, 2, scl.IsBuildable) &&
					rbpts.Intersect(b.GetBuildingPoints(pos+2-1i, scl.S2x2)).Empty() {
					pf5x3.Add(pos)
				} else {
					pf3x3.Add(pos)
				}
			}
			pos += 2 - 3i
			if b.HeightAt(pos) == slh && b.IsPosOk(pos, scl.S3x3, 2, scl.IsBuildable) &&
				rbpts.Intersect(b.GetBuildingPoints(pos, scl.S3x3)).Empty() {
				if b.IsPosOk(pos+2-1i, scl.S2x2, 2, scl.IsBuildable) &&
					rbpts.Intersect(b.GetBuildingPoints(pos+2-1i, scl.S2x2)).Empty() {
					pf5x3.Add(pos)
				} else {
					pf3x3.Add(pos)
				}
			}
			pos += 1 - 3i
			if b.HeightAt(pos) == slh && b.IsPosOk(pos, scl.S2x2, 2, scl.IsBuildable) &&
				rbpts.Intersect(b.GetBuildingPoints(pos, scl.S2x2)).Empty() {
				pf2x2.Add(pos)
			}
			pos += 2
			if b.HeightAt(pos) == slh && b.IsPosOk(pos, scl.S2x2, 2, scl.IsBuildable) &&
				rbpts.Intersect(b.GetBuildingPoints(pos, scl.S2x2)).Empty() {
				pf2x2.Add(pos)
			}
		}
	}
	pf2x2.OrderByDistanceTo(b.StartLoc, false)
	pf3x3.OrderByDistanceTo(b.StartLoc, false)
	pf5x3.OrderByDistanceTo(b.StartLoc, false)

	buildPos[scl.S2x2] = append(rp2x2, pf2x2...)
	buildPos[scl.S3x3] = pf3x3
	buildPos[scl.S5x3] = append(rp5x3, pf5x3...)
	buildPos[scl.S5x5] = b.ExpLocs

	/*b.Debug2x2Buildings(buildPos[scl.S2x2]...)
	b.Debug3x3Buildings(buildPos[scl.S3x3]...)
	b.Debug5x3Buildings(buildPos[scl.S5x3]...)
	b.DebugSend()*/
}

func (b *bot) BuildIfCan(aid api.AbilityID, size scl.BuildingSize, limit, active int) bool {
	buildings := b.Units.Units().Filter(scl.Structure)
	if b.CanBuy(aid) && b.Pending(aid) < limit && b.Orders[aid] < active {
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
				scv.CommandPos(aid, pos)
				log.Debugf("%d: Building %v", b.Loop, scl.Types[scl.AbilityUnit[aid]].Name)
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

func (b *bot) Build() {
	ccs := b.Units.OfType(terran.CommandCenter, terran.OrbitalCommand, terran.PlanetaryFortress)
	supCount := b.Units.OfType(terran.SupplyDepot, terran.SupplyDepotLowered).Filter(scl.Ready).Len()

	// Buildings
	if b.FoodLeft < 6+b.FoodUsed/20 && b.FoodCap < 200 &&
		b.BuildIfCan(ability.Build_SupplyDepot, scl.S2x2, 30, 1+b.FoodUsed/50) {
		return
	}
	// First barrack
	if supCount > 0 && b.BuildIfCan(ability.Build_Barracks, scl.S5x3, 1, 1) {
		return
	}
	// Refineries
	raxPending := b.Pending(ability.Build_Barracks)
	if b.CanBuy(ability.Build_Refinery) && (raxPending > 0 && b.Pending(ability.Build_Refinery) == 0 ||
		raxPending >= 3 && b.Pending(ability.Build_Refinery) >= 1) && b.Orders[ability.Build_Refinery] < 2 {
		if cc := ccs.First(scl.Ready); cc != nil {
			// Find first geyser that is close to my base, but it doesn't have Refinery on top of it
			geyser := b.VespeneGeysers.Units().CloserThan(10, cc.Point()).First(func(unit *scl.Unit) bool {
				return b.Units[terran.Refinery].CloserThan(1, unit.Point()).Len() == 0
			})
			if geyser != nil {
				scv := b.GetSCV(geyser.Point(), Builders, 45)
				if scv != nil {
					scv.CommandTag(ability.Build_Refinery, geyser.Tag)
					log.Debugf("%d: Building Refinery", b.Loop)
					return
				}
			}
		}
	}
	// Other barracks
	if supCount > 0 && b.BuildIfCan(ability.Build_Barracks, scl.S5x3, 3*ccs.Len(), 3) {
		return
	}
	if b.BuildIfCan(ability.Build_CommandCenter, scl.S5x5, buildPos[scl.S5x5].Len(), 1) {
		return
	}

	// Morph
	cc := b.Units[terran.CommandCenter].First(scl.Ready, scl.Idle)
	// b.CanBuy(ability.Morph_OrbitalCommand) requires 550 minerals?
	if cc != nil && b.Orders[ability.Train_Reaper] >= 2 && b.Minerals >= 150 {
		cc.Command(ability.Morph_OrbitalCommand)
		return
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

	// Cast
	cc = b.Units[terran.OrbitalCommand].First(func(unit *scl.Unit) bool { return unit.Energy >= 50 })
	if cc != nil {
		// Scan
		if b.Orders[ability.Effect_Scan] == 0 && b.EffectPoints(effect.ScannerSweep).Empty() {
			if reaper := b.Units[terran.Reaper].ClosestTo(b.EnemyStartLoc); reaper != nil {
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

	// Units
	cc = ccs.First(scl.Ready, scl.Idle)
	if cc != nil && b.Units[terran.SCV].Len() < scl.MinInt(21*ccs.Len(), 70) && b.CanBuy(ability.Train_SCV) {
		cc.Command(ability.Train_SCV)
		return
	}
	rax := b.Units[terran.Barracks].First(scl.Ready, scl.Idle)
	if rax != nil && b.CanBuy(ability.Train_Reaper) {
		rax.Command(ability.Train_Reaper)
		return
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

func (b *bot) RecalcEnemyStartLoc(np scl.Point) {
	b.EnemyStartLoc = np
	b.FindExpansions()
	b.InitRamps()
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

func (b *bot) WorkerRushDefence() {
	enemiesRange := 10.0
	if buildings := b.Units.Units().Filter(scl.Structure); buildings.Exists() {
		enemiesRange = math.Max(enemiesRange, buildings.FurthestTo(b.StartLoc).Point().Dist(b.StartLoc)+6)
	}
	enemies := b.EnemyUnits.OfType(terran.SCV, zerg.Drone, protoss.Probe).CloserThan(enemiesRange, b.StartLoc)
	if b.Units.Units().First(scl.DpsGt5) == nil {
		enemies = b.EnemyUnits.Units().Filter(scl.NotFlying).CloserThan(enemiesRange, b.StartLoc)
	}
	alert := enemies.CloserThan(enemiesRange-4, b.StartLoc).Exists()

	army := b.Groups.Get(WorkerRushDefenders).Units
	if army.Exists() && enemies.Empty() {
		b.Groups.Add(Miners, army...)
		return
	}

	if enemies.Len() >= 10 {
		workerRush = true
	}

	balance := 1.0 * army.Sum(scl.CmpGroundDPS) / enemies.Sum(scl.CmpGroundDPS)
	if alert && balance < 1 {
		worker := b.GetSCV(b.StartLoc, WorkerRushDefenders, 20)
		if worker != nil {
			army.Add(worker)
			b.Groups.Add(WorkerRushDefenders, worker)
		}
	}

	for _, unit := range army {
		if unit.Hits < 11 {
			b.Groups.Add(Miners, unit)
			continue
		}
		unit.Attack(enemies)
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

func (b *bot) ReaperFallback(u *scl.Unit, enemies scl.Units, safePos scl.Point) {
	p := u.Point()
	h := u.Bot.HeightAt(p)
	fbp := safePos
	score := 0.0
	for _, e := range enemies {
		score += e.GroundDPS() / (e.Point().Dist2(p) + 1)
	}
	score *= math.Log(math.Abs(p.Dist2(safePos)-4) + math.E)
	for x := 0.0; x < 16; x++ {
		vec := scl.Pt(5, 0).Rotate(math.Pi * 2.0 / 16.0 * x)
		np := p + vec
		np2 := p + vec*2
		maxJump := 1.0
		if u.Health < u.HealthMax {
			maxJump = 2.0
		}
		if (math.Abs(b.HeightAt(np)-h) > maxJump || !b.IsPathable(np)) &&
			(math.Abs(b.HeightAt(np2)-h) > maxJump || !b.IsPathable(np2)) {
			continue
		}
		newScore := 0.0
		for _, e := range enemies {
			newScore += e.GroundDPS() / e.Point().Dist2(np)
		}
		newScore *= math.Log(math.Abs(p.Dist2(safePos)-4) + math.E)
		if newScore < score {
			if b.IsPathable(np) {
				fbp = np
			} else {
				fbp = np2
			}
			score = newScore
		}
	}

	if u.WeaponCooldown > 0 {
		u.SpamCmds = true
	}
	u.CommandPos(ability.Move, fbp)
}

func (b *bot) ThrowMine(reaper *scl.Unit, targets scl.Units) bool {
	closestEnemy := targets.ClosestTo(reaper.Point())
	if closestEnemy != nil && reaper.HasAbility(ability.Effect_KD8Charge) {
		pos := closestEnemy.EstimatePositionAfter(50)
		if pos.IsCloserThan(float64(reaper.Radius) + reaper.GroundRange(), reaper.Point()) {
			reaper.CommandPos(ability.Effect_KD8Charge, pos)
			return true
		}
	}
	return false
}

func (b *bot) Reapers() {
	okTargets := scl.Units{}
	goodTargets := scl.Units{}
	hazards := scl.Units{}
	allEnemies := b.AllEnemyUnits.Units()
	reapers := b.Groups.Get(Reapers).Units
	for _, enemy := range allEnemies {
		if enemy.IsFlying || enemy.Is(zerg.Larva, zerg.Egg, protoss.AdeptPhaseShift, terran.KD8Charge) {
			continue
		}
		// Check if enemies that close to this one and have big range can kill reaper in a second
		enemiesDPS := allEnemies.Filter(func(unit *scl.Unit) bool {
			return unit.IsReady() && unit.GroundRange() >= 4 && unit.IsCloserThan(unit.GroundRange(), enemy)
		}).Sum(func(unit *scl.Unit) float64 {
			return unit.GroundDPS()
		})
		reapersDPS := reapers.CloserThan(15, enemy.Point()).Sum(func(unit *scl.Unit) float64 { return unit.GroundDPS() })
		if enemiesDPS >= 60 {
			if (!assault && (reapersDPS < enemiesDPS*2 || reapers.Len() <= 50)) ||
				(assault && (reapersDPS < enemiesDPS || reapers.Len() <= 25 )) {
				assault = false
				hazards.Add(enemy)
				continue // Evasion will be used
			} else {
				assault = true
			}
		}
		okTargets.Add(enemy)
		if !enemy.IsStructure() || enemy.IsDefensive() {
			goodTargets.Add(enemy)
		}
	}
	/* if goodTargets.Exists() {
		time.Sleep(time.Millisecond * 5)
	} */

	// Main army
	for _, reaper := range reapers {
		if reaper.Hits < 21 {
			b.Groups.Add(Retreat, reaper)
			continue
		}

		// Keep range
		// Weapon is recharging
		if !scl.AttackDelay.IsCool(reaper.UnitType, reaper.WeaponCooldown, reaper.Bot.FramesPerOrder) {
			if b.ThrowMine(reaper, goodTargets) {
				continue
			}
			// There is an enemy
			if closestEnemy := goodTargets.Filter(scl.Visible).ClosestTo(reaper.Point()); closestEnemy != nil {
				// And it is closer than shooting distance - 0.5
				if reaper.InRange(closestEnemy, -0.5) {
					// Retreat a little
					b.ReaperFallback(reaper, goodTargets, b.StartLoc)
					continue
				}
			}
		}

		// Evade dangerous zones
		ep := reaper.GroundEvade(hazards, 2, reaper.Point())
		if ep != reaper.Point() {
			reaper.CommandPos(ability.Move, ep)
			continue
		}

		// Attack
		if goodTargets.Exists() || okTargets.Exists() {
			reaper.Attack(goodTargets, okTargets, hazards)
		} else {
			if !b.IsExplored(b.EnemyStartLoc) {
				reaper.CommandPos(ability.Attack, b.EnemyStartLoc)
			} else {
				// Search for other bases
				if reaper.IsIdle() {
					pos := b.EnemyExpLocs[rand.Intn(len(b.EnemyExpLocs))]
					reaper.CommandPos(ability.Move, pos)
				}
			}
		}
	}

	// Damaged reapers
	reapers = b.Groups.Get(Retreat).Units
	for _, reaper := range reapers {
		if reaper.Health > 30 {
			b.Groups.Add(Reapers, reaper)
			continue
		}
		// Use attack if enemy is close but can't attack reaper
		if scl.AttackDelay.IsCool(reaper.UnitType, reaper.WeaponCooldown, reaper.Bot.FramesPerOrder) &&
			(goodTargets.InRangeOf(reaper, 0).Exists() || okTargets.InRangeOf(reaper, 0).Exists()) &&
			allEnemies.CanAttack(reaper, 1).Empty() {
			reaper.Attack(goodTargets, okTargets)
			continue
		}
		// Throw mine
		if b.ThrowMine(reaper, goodTargets) {
			continue
		}
		b.ReaperFallback(reaper, allEnemies, b.StartLoc)
	}
}

func (b *bot) Logic() {
	// time.Sleep(time.Millisecond * 5)
	b.BuildingsCheck()
	b.Builders()
	b.Build()
	b.Repair()
	b.Scout()
	b.WorkerRushDefence()
	b.Miners()
	b.Reapers()
}
