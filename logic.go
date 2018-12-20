package main

import (
	"bitbucket.org/aisee/sc2lib"
	"github.com/chippydip/go-sc2ai/api"
	"github.com/chippydip/go-sc2ai/enums/ability"
	"github.com/chippydip/go-sc2ai/enums/protoss"
	"github.com/chippydip/go-sc2ai/enums/terran"
	"github.com/chippydip/go-sc2ai/enums/zerg"
)

// todo: enemies types -> global
// todo: better building
// todo: fix morph abilities cost
// todo: groups

var workerRush = false
var pos2x2, pos3x3, pos5x3 scl.Points

const (
	Miners scl.GroupID = iota + 1
	Builders
	WorkerRushDefenders
	ProxyBuilders
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
	// todo: filter too close to those
	pos2x2.Add(b.FindRamp2x2Positions(b.MainRamp)...)
	pos5x3.Add(b.FindRampBarracksPositions(b.MainRamp))
	rbpts := b.GetBuildingPoints(pos5x3[0], scl.S5x3)

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

	pos2x2.Add(pf2x2...)
	pos3x3.Add(pf3x3...)
	pos5x3.Add(pf5x3...)

	b.Debug2x2Buildings(pos2x2...)
	b.Debug3x3Buildings(pos3x3...)
	b.Debug5x3Buildings(pf5x3...)
	b.DebugSend()
}

func (b *bot) Build() {
	// Buildings
	suppliesCount := b.Units.OfType(terran.SupplyDepot, terran.SupplyDepotLowered).Len()
	if b.CanBuy(ability.Build_SupplyDepot) && b.Orders[ability.Build_SupplyDepot] == 0 &&
		suppliesCount < pos2x2.Len() && b.FoodLeft < 6 {
		pos := pos2x2[suppliesCount]
		if scv := b.GetSCV(pos); scv != nil {
			scv.CommandPos(ability.Build_SupplyDepot, pos)
		}
	}
	raxPending := b.Units[terran.Barracks].Len()
	if b.CanBuy(ability.Build_Barracks) && raxPending < 3 && pos5x3.Len() > raxPending {
		pos := pos5x3[raxPending]
		scv := b.Groups.Get(ProxyBuilders).Units.Filter(func(unit *scl.Unit) bool {
			return unit.TargetAbility() != ability.Build_Barracks
		}).ClosestTo(pos)
		if scv != nil {
			scv.CommandPos(ability.Build_Barracks, pos)
		}
		return
	}
	if b.CanBuy(ability.Build_Refinery) && (raxPending == 1 && b.Units[terran.Refinery].Len() == 0 ||
		raxPending == 3 && b.Units[terran.Refinery].Len() == 1) {
		// Find first geyser that is close to my base, but it doesn't have Refinery on top of it
		geyser := b.VespeneGeysers.Units().CloserThan(10, b.StartLoc).First(func(unit *scl.Unit) bool {
			return b.Units[terran.Refinery].CloserThan(1, unit.Point()).Len() == 0
		})
		if geyser != nil {
			scv := b.GetSCV(geyser.Point())
			if scv != nil {
				scv.CommandTag(ability.Build_Refinery, geyser.Tag)
			}
		}
	}

	// Morph
	cc := b.Units[terran.CommandCenter].First(scl.Ready, scl.Idle)
	// b.CanBuy(ability.Morph_OrbitalCommand) requires 550 minerals?
	if cc != nil && b.Orders[ability.Train_Reaper] >= 2 && b.Minerals >= 150 {
		cc.Command(ability.Morph_OrbitalCommand)
	}
	if supply := b.Units[terran.SupplyDepot].First(); supply != nil {
		supply.Command(ability.Morph_SupplyDepot_Lower)
	}

	// Cast
	cc = b.Units[terran.OrbitalCommand].First(func(unit *scl.Unit) bool { return unit.Energy >= 50 })
	if cc != nil {
		homeMineral := b.MineralFields.Units().CloserThan(scl.ResourceSpreadDistance, b.StartLoc).First()
		if homeMineral != nil {
			cc.CommandTag(ability.Effect_CalldownMULE, homeMineral.Tag)
		}
	}

	// Units
	ccs := b.Units.OfType(terran.CommandCenter, terran.OrbitalCommand, terran.PlanetaryFortress)
	cc = ccs.First(scl.Ready, scl.Idle)
	if cc != nil && b.Units[terran.SCV].Len() < 21 && b.CanBuy(ability.Train_SCV) {
		cc.Command(ability.Train_SCV)
		return
	}
	rax := b.Units[terran.Barracks].First(scl.Ready, scl.Idle)
	if rax != nil && b.CanBuy(ability.Train_Reaper) {
		rax.Command(ability.Train_Reaper)
	}
}

func (b *bot) ProxyBuildres() {
	if b.Loop == 224 { // 10 sec
		scv := b.GetSCV(b.EnemyStartLoc)
		if scv != nil {
			pos := pos5x3[0]
			scv.CommandPos(ability.Move, pos)
			b.Groups.Add(ProxyBuilders, scv)
		}
	}
	if b.Loop == 672 { // 30 sec
		scv := b.GetSCV(b.EnemyStartLoc)
		if scv != nil {
			pos := pos5x3[1]
			scv.CommandPos(ability.Move, pos)
			b.Groups.Add(ProxyBuilders, scv)
		}
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

func (b *bot) Attack(targets scl.Units, us ...*scl.Unit) {
	for _, u := range us {
		closeTargets := targets.InRangeOf(u, 0)
		if closeTargets.Exists() {
			// target could be visible (not snapshot) when its map point is hidden
			// scl.PosVisible
			target := closeTargets.Filter(scl.Visible).Min(func(unit *scl.Unit) float64 {
				return unit.Hits
			})
			if target != nil {
				u.CommandTag(ability.Attack_Attack, target.Tag)
				return
			}
		}

		target := targets.ClosestTo(u.Point())
		far := u.SightRange() / 2

		// Attack as position, unit will choose best target around
		pos := b.UnitTargetPos(u)
		if u.IsIdle() || pos == 0 || target.Point().IsFurtherThan(far, pos) {
			u.CommandPos(ability.Attack_Attack, target.Point())
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
		b.Attack(enemies, unit)
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

// todo: okTargets, goodTargets, priorityTargets, treats
func (b *bot) PriorityAttack(goodTargets, okTargets scl.Units, us ...*scl.Unit) {
	for _, u := range us {
		if scl.UnitsOrders[u.Tag].Loop+u.Bot.FramesPerOrder > u.Bot.Loop {
			continue // Not more than FramesPerOrder
		}
		closeGoodTargets := goodTargets.InRangeOf(u, 0.5)
		closeOkTargets := okTargets.InRangeOf(u, 0)

		// Attack close targets
		targets := closeGoodTargets
		if targets.Empty() && u.WeaponCooldown < float32(b.FramesPerOrder) {
			targets = closeOkTargets
		}
		if targets.Exists() {
			target := targets.Min(func(unit *scl.Unit) float64 {
				return unit.Hits
			})
			if target != nil {
				u.CommandTag(ability.Attack_Attack, target.Tag)
				continue
			}
		}

		// Move closer to targets
		target := goodTargets.ClosestTo(u.Point())
		if target == nil {
			if !b.IsExplored(b.EnemyStartLoc) {
				u.CommandPos(ability.Move, b.EnemyStartLoc)
				continue
			}
			target = okTargets.ClosestTo(u.Point())
		}
		if target != nil && (!u.InRange(target, 0) || !b.IsVisible(target.Point())) {
			if u.WeaponCooldown > 0 {
				// Spamming this thing is the key. Or orders will be ignored (or postponed)
				u.SpamCmds = true
			}
			u.CommandPos(ability.Move, target.Point())
		}
	}
}

func (b *bot) Reapers() {
	okTargets := scl.Units{}
	goodTargets := scl.Units{}
	for _, unit := range b.AllEnemyUnits.Units() {
		if !unit.IsFlying && unit.IsNot(zerg.Larva, zerg.Egg, protoss.AdeptPhaseShift, terran.KD8Charge) {
			okTargets.Add(unit)
			if (!unit.IsStructure() && b.IsVisible(unit.Point())) || unit.IsDefensive() {
				goodTargets.Add(unit)
			}
		}
	}

	reapers := b.Groups.Get(Reapers).Units
	/*if reapers.Exists() {
		time.Sleep(time.Millisecond * 20)
	}*/
	for _, reaper := range reapers {
		if reaper.Hits < 21 {
			b.Groups.Add(Retreat, reaper)
			continue
		}

		// Keep range
		// Weapon is recharging
		if reaper.WeaponCooldown >= float32(b.FramesPerOrder) {
			// There is an enemy
			if closestEnemy := goodTargets.ClosestTo(reaper.Point()); closestEnemy != nil {
				// And it is closer than shooting distance - 0.5
				if reaper.InRange(closestEnemy, -0.5) {
					// Retreat a little
					reaper.CommandPos(ability.Move, b.StartLoc)
					continue
				}
			}
		}

		// Attack
		if goodTargets.Exists() || okTargets.Exists() {
			b.PriorityAttack(goodTargets, okTargets, reaper)
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
	// time.Sleep(time.Millisecond * 5)

	b.BuildingsCheck()
	b.Builders()
	b.ProxyBuildres()
	b.Build()
	b.Scout()
	b.WorkerRushDefence()
	b.Miners()
	b.Reapers()
	b.Retreat()
}
