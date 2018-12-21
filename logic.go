package main

import (
	"bitbucket.org/aisee/sc2lib"
	"github.com/chippydip/go-sc2ai/api"
	"github.com/chippydip/go-sc2ai/enums/ability"
	"github.com/chippydip/go-sc2ai/enums/protoss"
	"github.com/chippydip/go-sc2ai/enums/terran"
	"github.com/chippydip/go-sc2ai/enums/zerg"
	"time"
	"bitbucket.org/aisee/minilog"
	"math"
)

// todo: enemies types -> global
// todo: better building
// todo: restart buildings construction
// todo: fix morph abilities cost
// todo: groups

var workerRush = false
var buildPos = map[scl.BuildingSize]scl.Points{}

const (
	Miners scl.GroupID = iota + 1
	Builders
	WorkerRushDefenders
	Scout
	Reapers
	Retreat
	UnderConstruction
	Buildings
	MaxGroup
)

func (b *bot) GetSCV(pos scl.Point) *scl.Unit {
	csv := b.Groups.Get(Miners).Units.Filter(scl.Gathering).ClosestTo(pos)
	if csv != nil {
		b.Groups.Add(Builders, csv)
	}
	return csv
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
	buildings := b.Groups.Get(UnderConstruction).Units
	for _, building := range buildings {
		if building.BuildProgress == 1 {
			switch building.UnitType {
			default:
				b.Groups.Add(Buildings, building) // And remove from current group
			}
			continue
		}
		// Cancel building if it will be destroyed soon
		if building.HPS*2.5 > building.Hits {
			building.Command(ability.Cancel)
		}
	}
}

func (b *bot) Builders() {
	builders := b.Groups.Get(Builders).Units
	for _, u := range builders {
		enemy := b.EnemyUnits.Units().First(func(unit *scl.Unit) bool {
			return unit.InRange(u, 0.5)
		})
		if enemy != nil {
			b.Groups.Add(Miners, u)
		}
	}

	// Move idle builders into miners
	idleBuilder := b.Groups.Get(Builders).Units.First(scl.Idle)
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

	buildPos[scl.S2x2].Add(b.FindRamp2x2Positions(b.MainRamp)...)
	buildPos[scl.S5x3].Add(b.FindRampBarracksPositions(b.MainRamp))
	rbpts := b.GetBuildingPoints(buildPos[scl.S5x3][0], scl.S5x3)

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

	buildPos[scl.S2x2].Add(pf2x2...)
	buildPos[scl.S3x3].Add(pf3x3...)
	buildPos[scl.S5x3].Add(pf5x3...)

	b.Debug2x2Buildings(buildPos[scl.S2x2]...)
	b.Debug3x3Buildings(buildPos[scl.S3x3]...)
	b.Debug5x3Buildings(buildPos[scl.S5x3]...)
	b.DebugSend()
}

func (b *bot) BuildIfCan(aid api.AbilityID, size scl.BuildingSize, limit, active int) bool {
	// todo: buildings -> bot obj?
	buildings := b.Units.Units().Filter(scl.Structure)
	if b.CanBuy(aid) && b.Units[scl.AbilityUnit[aid]].Len() < limit && b.Orders[aid] < active {
		for _, pos := range buildPos[size] {
			if buildings.CloserThan(math.Sqrt2, pos).Exists() {
				continue
			}

			bps := b.GetBuildingPoints(pos, size)
			if !b.CheckPoints(bps, scl.IsNoCreep) {
				continue
			}

			scv := b.GetSCV(pos)
			if scv != nil {
				scv.CommandPos(aid, pos)
				log.Debugf("Building %v", b.Units[scl.AbilityUnit[aid]])
				return true
			}
			log.Debug("Failed to find SCV")
			return false
		}
		log.Debugf("Can't find position for %v", b.Units[scl.AbilityUnit[aid]])
		if size == scl.S3x3 {
			log.Debug("Trying bigger size for 3x3")
			return b.BuildIfCan(aid, scl.S5x3, limit, active)
		}
	}
	return false
}

func (b *bot) Build() {
	ccs := b.Units.OfType(terran.CommandCenter, terran.OrbitalCommand, terran.PlanetaryFortress)

	// Buildings
	if b.FoodLeft < 6 && b.BuildIfCan(ability.Build_SupplyDepot, scl.S2x2, 20, 1) {
		return
	}
	if b.BuildIfCan(ability.Build_Barracks, scl.S5x3, 3 * ccs.Len(), 3) {
		return
	}
	raxPending := b.Units[terran.Barracks].Len()
	if b.CanBuy(ability.Build_Refinery) && (raxPending == 1 && b.Units[terran.Refinery].Len() == 0 ||
		raxPending == 3 && b.Units[terran.Refinery].Len() >= 1) && b.Orders[ability.Build_Refinery] < 2 {
		cc := ccs.First(scl.Ready)
		// Find first geyser that is close to my base, but it doesn't have Refinery on top of it
		geyser := b.VespeneGeysers.Units().CloserThan(10, cc.Point()).First(func(unit *scl.Unit) bool {
			return b.Units[terran.Refinery].CloserThan(1, unit.Point()).Len() == 0
		})
		if geyser != nil {
			scv := b.GetSCV(geyser.Point())
			if scv != nil {
				scv.CommandTag(ability.Build_Refinery, geyser.Tag)
				return
			}
		}
	}
	// todo: BuildIfCan
	if b.CanBuy(ability.Build_CommandCenter) && b.Orders[ability.Build_CommandCenter] == 0 {
		for _, pos := range b.ExpLocs {
			if b.Units.Units().Filter(scl.Structure).CloserThan(3, pos).Exists() {
				continue // todo: better check
			}
			if scv := b.GetSCV(pos); scv != nil {
				scv.CommandPos(ability.Build_CommandCenter, pos)
				return
			}
		}
	}

	// Morph
	cc := b.Units[terran.CommandCenter].First(scl.Ready, scl.Idle)
	// b.CanBuy(ability.Morph_OrbitalCommand) requires 550 minerals?
	if cc != nil && b.Orders[ability.Train_Reaper] >= 2 && b.Minerals >= 150 {
		cc.Command(ability.Morph_OrbitalCommand)
		return
	}
	if supply := b.Units[terran.SupplyDepot].First(); supply != nil {
		supply.Command(ability.Morph_SupplyDepot_Lower)
	}

	// Cast
	cc = b.Units[terran.OrbitalCommand].First(func(unit *scl.Unit) bool { return unit.Energy >= 50 })
	if cc != nil {
		homeMineral := b.MineralFields.Units().
			CloserThan(scl.ResourceSpreadDistance, cc.Point()).
			Max(func(unit *scl.Unit) float64 {
			return float64(unit.MineralContents)
		})
		if homeMineral != nil {
			cc.CommandTag(ability.Effect_CalldownMULE, homeMineral.Tag)
		}
	}

	// Units
	cc = ccs.First(scl.Ready, scl.Idle)
	if cc != nil && b.Units[terran.SCV].Len() < scl.MaxInt(21 * ccs.Len(), 70) && b.CanBuy(ability.Train_SCV) {
		cc.Command(ability.Train_SCV)
		return
	}
	rax := b.Units[terran.Barracks].First(scl.Ready, scl.Idle)
	if rax != nil && b.CanBuy(ability.Train_Reaper) {
		rax.Command(ability.Train_Reaper)
		return
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
	enemies := b.EnemyUnits.OfType(terran.SCV, zerg.Drone, protoss.Probe).CloserThan(10, b.StartLoc)
	alert := enemies.CloserThan(6, b.StartLoc).Exists()

	army := b.Groups.Get(WorkerRushDefenders).Units
	if army.Exists() && enemies.Empty() {
		b.Groups.Add(Miners, army...)
		return
	}

	if enemies.Len() >= 10 {
		workerRush = true
	}

	balance := 1.0 * float64(army.Len()) / float64(enemies.Len())
	if alert && balance < 1 {
		worker := b.Groups.Get(Miners).Units.First(scl.Gathering, func(unit *scl.Unit) bool {
			return unit.Hits >= 20
		})
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
	if b.Loop%6 != 0 {
		// try to fix destribution bug. Might be caused by AssignedHarvesters lagging
		return
	}
	// Std miners handler
	b.HandleMiners(
		b.Groups.Get(Miners).Units,
		b.Units.OfType(terran.CommandCenter, terran.OrbitalCommand, terran.PlanetaryFortress).Filter(scl.Ready),
		1)

	// If there is ready unsaturated refinery and an scv gathering, send it there
	refinery := b.Units[terran.Refinery].
		First(func(unit *scl.Unit) bool { return unit.IsReady() && unit.AssignedHarvesters < 3 })
	if refinery != nil {
		scv := b.GetSCV(refinery.Point())
		if scv != nil {
			scv.CommandTag(ability.Harvest_Gather_SCV, refinery.Tag)
		}
	}
}

func (b *bot) Reapers() {
	okTargets := scl.Units{}
	goodTargets := scl.Units{}
	for _, unit := range b.AllEnemyUnits.Units() {
		if !unit.IsFlying && unit.IsNot(zerg.Larva, zerg.Egg, protoss.AdeptPhaseShift, terran.KD8Charge) {
			okTargets.Add(unit)
			if !unit.IsStructure() || unit.IsDefensive() {
				goodTargets.Add(unit)
			}
		}
	}

	reapers := b.Groups.Get(Reapers).Units
	for _, reaper := range reapers {
		if reaper.Hits < 21 {
			b.Groups.Add(Retreat, reaper)
			continue
		}

		// Keep range
		// Weapon is recharging
		if !scl.AttackDelay.IsCool(reaper.UnitType, reaper.WeaponCooldown, reaper.Bot.FramesPerOrder) {
			// There is an enemy
			if closestEnemy := goodTargets.ClosestTo(reaper.Point()); closestEnemy != nil {
				// And it is closer than shooting distance - 0.5
				if reaper.InRange(closestEnemy, -0.5) {
					// Retreat a little
					reaper.SpamCmds = true
					reaper.CommandPos(ability.Move, b.StartLoc)
					continue
				}
			}
		}

		// Attack
		if goodTargets.Exists() || okTargets.Exists() {
			time.Sleep(time.Millisecond * 20)
			reaper.Attack(goodTargets, okTargets)
		} else {
			reaper.CommandPos(ability.Attack, b.EnemyStartLoc)
		}
	}
}

func (b *bot) Retreat() {
	reapers := b.Groups.Get(Retreat).Units
	for _, reaper := range reapers {
		if reaper.Health > 30 {
			b.Groups.Add(Reapers, reaper)
			continue
		}
		reaper.CommandPos(ability.Move, b.StartLoc)
	}
}

func (b *bot) Logic() {
	b.BuildingsCheck()
	b.Builders()
	b.Build()
	b.Scout()
	b.WorkerRushDefence()
	b.Miners()
	b.Reapers()
	b.Retreat()
}
