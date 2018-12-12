package main

import (
	"bitbucket.org/AiSee/sc2lib"
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

const (
	Miners scl.GroupID = iota + 1
	Builders
	WorkerRushDefenders
	ProxyBuilder
	Scout
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
			building.Command(b.Cmds, ability.Cancel)
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
	pos := b.StartLoc + vec*3.5
	b.PositionsForSupplies.Add(pos)
	b.PositionsForSupplies.Add(pos.Neighbours4(2)...)

	pos = b.EnemyStartLoc.Towards(b.StartLoc, 25)
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
			b.PositionsForBarracks.Add(scl.Pt2(pfb[key].TargetPos))
		}
	}
}

func (b *bot) Build() {
	// Buildings
	suppliesCount := b.Units.OfType(terran.SupplyDepot, terran.SupplyDepotLowered).Len()
	if b.CanBuy(ability.Build_SupplyDepot) && b.Orders[ability.Build_SupplyDepot] == 0 &&
		suppliesCount < b.PositionsForSupplies.Len() && b.FoodLeft < 6 {
		pos := b.PositionsForSupplies[suppliesCount]
		if scv := b.GetSCV(pos); scv != nil {
			scv.CommandPos(b.Cmds, ability.Build_SupplyDepot, pos)
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
			scv = b.GetSCV(pos)
		}
		if scv != nil {
			scv.CommandPos(b.Cmds, ability.Build_Barracks, pos)
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
				scv.CommandTag(b.Cmds, ability.Build_Refinery, geyser.Tag)
			}
		}
	}

	// Morph
	cc := b.Units[terran.CommandCenter].First(scl.Ready, scl.Idle)
	// b.CanBuy(ability.Morph_OrbitalCommand) requires 550 minerals?
	if cc != nil && b.Orders[ability.Train_Reaper] >= 2 && b.Minerals >= 150 {
		cc.Command(b.Cmds, ability.Morph_OrbitalCommand)
	}
	if supply := b.Units[terran.SupplyDepot].First(); supply != nil {
		supply.Command(b.Cmds, ability.Morph_SupplyDepot_Lower)
	}

	// Cast
	cc = b.Units[terran.OrbitalCommand].First(func(unit *scl.Unit) bool { return unit.Energy >= 50 })
	if cc != nil {
		homeMineral := b.MineralFields.Units().CloserThan(scl.ResourceSpreadDistance, b.StartLoc).First()
		if homeMineral != nil {
			cc.CommandTag(b.Cmds, ability.Effect_CalldownMULE, homeMineral.Tag)
		}
	}

	// Units
	ccs := b.Units.OfType(terran.CommandCenter, terran.OrbitalCommand, terran.PlanetaryFortress)
	cc = ccs.First(scl.Ready, scl.Idle)
	if cc != nil && b.Units[terran.SCV].Len() < 21 && b.CanBuy(ability.Train_SCV) {
		cc.Command(b.Cmds, ability.Train_SCV)
		return
	}
	rax := b.Units[terran.Barracks].First(scl.Ready, scl.Idle)
	if rax != nil && b.CanBuy(ability.Train_Reaper) {
		rax.Command(b.Cmds, ability.Train_Reaper)
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
			scv.CommandPos(b.Cmds, ability.Move, b.EnemyStartLocs[0])
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
				scv.CommandPos(b.Cmds, ability.Move, p)
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
			target := closeTargets.Min(func(unit *scl.Unit) float64 {
				return unit.Hits
			})
			// target could be visible (not snapshot) when its map point is hidden
			if b.IsVisible(target.Point()) {
				u.CommandTag(b.Cmds, ability.Attack_Attack, target.Tag)
				return
			}
		}

		target := targets.ClosestTo(u.Point())
		far := u.SightRange() / 2

		// Attack as position, unit will choose best target around
		pos := b.UnitTargetPos(u)
		if u.IsIdle() || pos == 0 || target.Point().IsFurtherThan(far, pos) {
			u.CommandPos(b.Cmds, ability.Attack_Attack, target.Point())
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
			scv.CommandTag(b.Cmds, ability.Harvest_Gather_SCV, refinery.Tag)
		}
	}
}

func (b *bot) Army() {
	// time.Sleep(time.Millisecond * 20)
	if b.Loop == 224 { // 10 sec
		scv := b.GetSCV(b.EnemyStartLoc)
		if scv != nil {
			pos := b.PositionsForBarracks[0]
			scv.CommandPos(b.Cmds, ability.Move, pos)
			b.Builder1 = scv.Tag
		}
	}
	if b.Loop == 672 { // 30 sec
		scv := b.GetSCV(b.EnemyStartLoc)
		if scv != nil {
			pos := b.PositionsForBarracks[1]
			scv.CommandPos(b.Cmds, ability.Move, pos)
			b.Builder2 = scv.Tag
		}
	}

	okTargets := scl.Units{}
	goodTargets := scl.Units{}
	for _, units := range b.EnemyUnits {
		for _, unit := range units {
			if !unit.IsFlying && unit.IsNot(zerg.Larva, zerg.Egg, protoss.AdeptPhaseShift, terran.KD8Charge) {
				okTargets.Add(unit)
				if !unit.IsStructure() {
					goodTargets.Add(unit)
				}
			}
		}
	}

	reapers := b.Units[terran.Reaper]
	if okTargets.Empty() {
		reapers.CommandPos(b.Cmds, ability.Attack, b.EnemyStartLoc)
	} else {
		for _, reaper := range reapers {
			// retreat
			if b.Retreat[reaper.Tag] && reaper.Health > 50 {
				delete(b.Retreat, reaper.Tag)
			}
			if reaper.Health < 21 || b.Retreat[reaper.Tag] {
				b.Retreat[reaper.Tag] = true
				reaper.CommandPos(b.Cmds, ability.Move, b.PositionsForBarracks[0])
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
						reaper.CommandPos(b.Cmds, ability.Move, b.PositionsForBarracks[0])
						continue
					}
				}
			}

			// Attack
			// todo: use func
			if goodTargets.Exists() {
				target := goodTargets.ClosestTo(reaper.Point())
				// Snapshots couldn't be targeted using tags
				if reaper.IsCloserThan(4, target) && target.DisplayType != api.DisplayType_Snapshot {
					// If target is far, attack it as unit, ling will run ignoring everything else
					reaper.CommandTag(b.Cmds, ability.Attack, target.Tag)
				} else {
					// Attack as position, ling will choose best target around
					reaper.CommandPos(b.Cmds, ability.Attack, target.Point())
				}
			} else {
				target := okTargets.ClosestTo(reaper.Point())
				reaper.CommandPos(b.Cmds, ability.Attack, target.Point())
			}
		}
	}
}

func (b *bot) Logic() {
	// time.Sleep(time.Millisecond * 5)

	b.BuildingsCheck()
	b.Builders()
	b.Build()
	b.Scout()
	b.WorkerRushDefence()
	b.Miners()
	b.Army()
}
