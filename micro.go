package main

import (
	"bitbucket.org/aisee/sc2lib"
	"github.com/chippydip/go-sc2ai/api"
	"github.com/chippydip/go-sc2ai/enums/ability"
	"github.com/chippydip/go-sc2ai/enums/protoss"
	"github.com/chippydip/go-sc2ai/enums/terran"
	"github.com/chippydip/go-sc2ai/enums/zerg"
	"math"
	"math/rand"
)

func (b *bot) WorkerRushDefence() {
	enemiesRange := 12.0
	workersRange := 10.0
	if building := b.Units.Units().Filter(scl.Structure).ClosestTo(b.MainRamp.Top); building != nil {
		workersRange = math.Max(workersRange, building.Point().Dist(b.StartLoc)+6)
	}
	if workerRush {
		workersRange = 70.0
	}

	workers := b.Units.OfType(terran.SCV).CloserThan(scl.ResourceSpreadDistance, b.StartLoc)
	enemies := b.EnemyUnits.Units().Filter(scl.NotFlying).CloserThan(enemiesRange, b.StartLoc)
	alert := enemies.CloserThan(enemiesRange-4, b.StartLoc).Exists()
	if enemies.Empty() || enemies.Sum(scl.CmpGroundScore) > workers.Sum(scl.CmpGroundScore)*2 || workerRush {
		enemies = b.EnemyUnits.OfType(terran.SCV, zerg.Drone, protoss.Probe).CloserThan(workersRange, b.StartLoc)
		alert = enemies.CloserThan(workersRange-4, b.StartLoc).Exists()
		if alert && enemies.Len() >= 10 {
			workerRush = true
		}
	}
	if workerRush && b.EnemyUnits.OfType(terran.SCV, zerg.Drone, protoss.Probe).CloserThan(70, b.StartLoc).Empty() {
		workerRush = false
	}

	army := b.Groups.Get(WorkerRushDefenders).Units
	if army.Exists() && enemies.Empty() {
		b.Groups.Add(Miners, army...)
		return
	}

	balance := army.Sum(scl.CmpGroundScore) / enemies.Sum(scl.CmpGroundScore)
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
		if scl.AttackDelay.UnitIsCool(unit) {
			unit.Attack(enemies)
		} else {
			friends := army.InRangeOf(unit, 0)
			friend := friends.Min(scl.CmpHits)
			if friend != nil && friend.Hits < 45 && b.Minerals > 0 {
				unit.CommandTag(ability.Effect_Repair_SCV, friend.Tag)
			}
		}
	}
}

func (b *bot) ThrowMine(reaper *scl.Unit, targets scl.Units) bool {
	closestEnemy := targets.ClosestTo(reaper.Point())
	if closestEnemy != nil && reaper.HasAbility(ability.Effect_KD8Charge) {
		// pos := closestEnemy.EstimatePositionAfter(50)
		pos := closestEnemy.Point()
		if pos.IsCloserThan(float64(reaper.Radius)+reaper.GroundRange(), reaper.Point()) {
			reaper.CommandPos(ability.Effect_KD8Charge, pos)
			return true
		}
	}
	return false
}

func ReaperMoveFunc(u *scl.Unit, target *scl.Unit) {
	// Unit need to be closer to the target to shoot?
	if !u.InRange(target, -0.1) || !target.IsVisible() {
		u.AttackMove(target, u.Bot.HomeReaperPaths)
		/*if u.Hits == u.HitsMax {
			u.AttackMove(target, u.Bot.EnemyReaperPaths)
		} else {
			u.AttackMove(target, u.Bot.HomeReaperPaths)
		}*/
	}
}

func (b *bot) Explore(u *scl.Unit) {
	if !b.IsExplored(b.EnemyStartLoc) {
		u.CommandPos(ability.Attack, b.EnemyStartLoc)
	} else {
		// Search for other bases
		if u.IsIdle() {
			pos := b.EnemyExpLocs[rand.Intn(len(b.EnemyExpLocs))]
			u.CommandPos(ability.Move, pos)
		}
	}
}

func (b *bot) Marines() {
	okTargets := scl.Units{}
	goodTargets := scl.Units{}
	allEnemies := b.AllEnemyUnits.Units()
	allEnemiesReady := allEnemies.Filter(scl.Ready)
	marines := b.Groups.Get(Marines).Units
	for _, enemy := range allEnemies {
		if enemy.Is(zerg.Larva, zerg.Egg, protoss.AdeptPhaseShift, terran.KD8Charge) {
			continue
		}
		okTargets.Add(enemy)
		if !enemy.IsStructure() || enemy.IsDefensive() {
			goodTargets.Add(enemy)
		}
	}

	for _, marine := range marines {
		if !scl.AttackDelay.UnitIsCool(marine) {
			attackers := allEnemiesReady.CanAttack(marine, 2)
			closeTargets := goodTargets.InRangeOf(marine, -0.5)
			if attackers.Exists() || closeTargets.Exists() {
				marine.GroundFallback(attackers, 2, b.HomePaths)
				continue
			}
		}

		// Attack
		if goodTargets.Exists() || okTargets.Exists() {
			marine.Attack(goodTargets, okTargets)
		} else {
			b.Explore(marine)
		}
	}
}

func (b *bot) Reapers() {
	okTargets := scl.Units{}
	goodTargets := scl.Units{}
	allEnemies := b.AllEnemyUnits.Units()
	allEnemiesReady := allEnemies.Filter(scl.Ready)
	reapers := b.Groups.Get(Reapers).Units
	for _, enemy := range allEnemies {
		if enemy.IsFlying || enemy.Is(zerg.Larva, zerg.Egg, protoss.AdeptPhaseShift, terran.KD8Charge) {
			continue
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
			b.Groups.Add(ReapersRetreat, reaper)
			continue
		}

		// Keep range
		// Weapon is recharging
		if !scl.AttackDelay.UnitIsCool(reaper) {
			if b.ThrowMine(reaper, goodTargets) {
				continue
			}

			// There is an enemy
			/*if closestEnemy := goodTargets.Filter(scl.Visible).ClosestTo(reaper.Point()); closestEnemy != nil {
				// And it is closer than shooting distance - 0.5
				if reaper.InRange(closestEnemy, -0.5) {
					// Retreat a little
					reaper.GroundFallback(goodTargets, -0.5, b.HomeReaperPaths)
					continue
				}
			}*/

			attackers := allEnemiesReady.CanAttack(reaper, 2)
			closeTargets := goodTargets.InRangeOf(reaper, -0.5)
			if attackers.Exists() || closeTargets.Exists() {
				reaper.GroundFallback(attackers, 2, b.HomeReaperPaths)
				continue
			}
		}

		// Attack
		if goodTargets.Exists() || okTargets.Exists() {
			reaper.AttackCustom(scl.DefaultAttackFunc, ReaperMoveFunc, goodTargets, okTargets /*, hazards*/)
		} else {
			b.Explore(reaper)
		}
	}

	// Damaged reapers
	reapers = b.Groups.Get(ReapersRetreat).Units
	for _, reaper := range reapers {
		if reaper.Health > 30 {
			b.Groups.Add(Reapers, reaper)
			continue
		}
		// Use attack if enemy is close but can't attack reaper
		if scl.AttackDelay.UnitIsCool(reaper) &&
			(goodTargets.InRangeOf(reaper, 0).Exists() || okTargets.InRangeOf(reaper, 0).Exists()) &&
			allEnemiesReady.CanAttack(reaper, 1).Empty() {
			reaper.AttackCustom(scl.DefaultAttackFunc, ReaperMoveFunc, goodTargets, okTargets)
			continue
		}
		// Throw mine
		if b.ThrowMine(reaper, goodTargets) {
			continue
		}
		reaper.GroundFallback(allEnemiesReady, 2, b.HomeReaperPaths)
	}
}

func CycloneAttackFunc(u *scl.Unit, priority int, targets scl.Units) bool {
	hasLockOn := u.HasAbility(ability.Effect_LockOn)
	visibleTargets := targets.Filter(scl.Visible)
	if hasLockOn {
		closeTargets := visibleTargets.InRangeOf(u, 2) // Range = 7. Weapons + 2
		if closeTargets.Exists() {
			target := closeTargets.Max(func(unit *scl.Unit) float64 {
				if unit.IsArmored() {
					return unit.Hits * 2
				}
				return unit.Hits
			})
			u.CommandTag(ability.Effect_LockOn, target.Tag)
			return true
		}
		return false
	}
	closeTargets := visibleTargets.InRangeOf(u, 0)
	if closeTargets.Exists() {
		target := closeTargets.Min(func(unit *scl.Unit) float64 {
			return unit.Hits
		})
		u.CommandTag(ability.Attack_Attack_23, target.Tag)
		return true
	}
	return false
}

func CycloneMoveFunc(u *scl.Unit, target *scl.Unit) {
	// Unit need to be closer to the target to shoot? (lock-on range)
	if !u.InRange(target, 2-0.1) || !target.IsVisible() {
		u.AttackMove(target, u.Bot.HomePaths)
	}
}

func (b *bot) Cyclones() {
	okTargets := scl.Units{}
	goodTargets := scl.Units{}
	airTargets := scl.Units{}
	allEnemies := b.AllEnemyUnits.Units()
	allEnemiesReady := allEnemies.Filter(scl.Ready)
	cyclones := b.Groups.Get(Cyclones).Units
	for _, enemy := range allEnemies {
		if enemy.Is(zerg.Larva, zerg.Egg, protoss.AdeptPhaseShift, terran.KD8Charge) {
			continue
		}
		okTargets.Add(enemy)
		if enemy.IsStructure() && !enemy.IsDefensive() {
			continue
		}
		goodTargets.Add(enemy)
		if !enemy.IsFlying {
			continue
		}
		airTargets.Add(enemy)
	}

	for _, cyclone := range cyclones {
		if cyclone.Hits < 51 {
			b.Groups.Add(MechRetreat, cyclone)
			continue
		}

		// Keep range
		// canAttack := scl.AttackDelay.UnitIsCool(cyclone) || cyclone.HasAbility(ability.Effect_LockOn)
		/* if canAttack && ! { // And if can't lock on
			canAttack = allEnemies.ByTag(cyclone.EngagedTargetTag) == nil // There is no lock
		} */
		/*if !canAttack {
			attackers := allEnemiesReady.CanAttack(cyclone, 4)
			closeTargets := goodTargets.InRangeOf(cyclone, -0.5)
			if attackers.Exists() || closeTargets.Exists() {
				cyclone.GroundFallback(attackers, 2, b.HomePaths)
				continue
			}
		}*/
		attackers := allEnemiesReady.CanAttack(cyclone, 2)
		retreat := cyclone.HPS > 0 && attackers.Exists()
		if !retreat && !cyclone.HasAbility(ability.Effect_LockOn) && attackers.Exists() {
			target := allEnemies.ByTag(cyclone.EngagedTargetTag)
			// Someone is locked on and he is close enough
			retreat = target != nil && cyclone.InRange(target, 5)
		}
		if retreat {
			cyclone.GroundFallback(attackers, 2, b.HomePaths)
			continue
		}

		// Attack
		if airTargets.Exists() || goodTargets.Exists() || okTargets.Exists() {
			cyclone.AttackCustom(CycloneAttackFunc, CycloneMoveFunc, airTargets, goodTargets, okTargets)
		} else {
			b.Explore(cyclone)
		}
	}
}

func (b *bot) Mines() {
	targets := scl.Units{}
	// detectors := scl.Units{}
	allEnemies := b.AllEnemyUnits.Units()
	// allEnemiesReady := allEnemies.Filter(scl.Ready)
	mines := b.Groups.Get(WidowMines).Units
	for _, enemy := range allEnemies {
		/*if enemy.DetectRange > 0 {
			detectors.Add(enemy)
		}*/
		if enemy.IsStructure() || enemy.Is(zerg.Larva, zerg.Egg, protoss.AdeptPhaseShift, terran.KD8Charge) {
			continue
		}
		targets.Add(enemy)
	}

	for _, mine := range mines {
		attackers := allEnemies.CanAttack(mine, 2)
		// Something that could detect mine is close, ex: photon cannon
		/*detectorIsNear := detectors.First(func(unit *scl.Unit) bool {
			return unit.Point().IsCloserThan(float64(unit.DetectRange)+1, mine.Point())
		}) != nil*/
		// Someone is attacking mine, but it can't attack anyone
		detected := false
		if mine.HPS > 0 {
			// detected = targets.InRangeOf(mine, 0).Empty() - this is wrong? Mine has no weapon?
			detected = targets.First(func(unit *scl.Unit) bool {
				return mine.Point().Dist(unit.Point()) <= float64(mine.Radius+unit.Radius+5)
			}) == nil
		}
		/*if !detected && detectorIsNear {
			// In range of known detector
			detected = detectors.First(func(unit *scl.Unit) bool {
				return unit.Point().IsCloserThan(float64(unit.DetectRange), mine.Point())
			}) != nil
		}*/

		if mine.IsBurrowed && (detected ||
			!mine.HasAbility(ability.Smart) || // Reloading
			targets.CloserThan(10, mine.Point()).Empty() /*&& !detectorIsNear*/ && attackers.Empty()) {
			// No targets, enemies or close detectors around
			if mine.Hits < mine.HitsMax/2 {
				b.Groups.Add(MechRetreat, mine)
			} else {
				b.Groups.Add(WidowMinesRetreat, mine)
			}
			mine.Command(ability.BurrowUp_WidowMine)
			continue
		}

		targetIsClose := targets.CloserThan(4, mine.Point()).Exists() // For enemies that can't attack ground
		if !mine.IsBurrowed && !detected && (attackers.Exists() /*|| detectorIsNear*/ || targetIsClose) {
			mine.Command(ability.BurrowDown_WidowMine)
			continue
		}

		if targets.Exists() {
			mine.CommandPos(ability.Move, targets.ClosestTo(mine.Point()).Point())
		} else {
			b.Explore(mine)
		}
	}

	mines = b.Groups.Get(WidowMinesRetreat).Units
	for _, mine := range mines {
		if mine.Hits < mine.HitsMax {
			b.Groups.Add(MechRetreat, mine)
			continue
		}
		if mine.IsBurrowed && mine.HasAbility(ability.Smart) {
			b.Groups.Add(WidowMines, mine)
			continue
		}
		if mine.IsIdle() {
			vec := (b.EnemyStartLoc - mine.Point()).Norm()
			p1 := mine.Point() - vec*20
			p2 := p1
			if rand.Intn(2) == 1 {
				vec *= 1i
			} else {
				vec *= -1i
			}
			for {
				if !b.IsPathable(p2 + vec*10) {
					break
				}
				p2 += vec * 10
			}

			mine.CommandPos(ability.Move, p1)
			mine.CommandPosQueue(ability.Move, p2)
			mine.CommandQueue(ability.BurrowDown_WidowMine)
		}
	}
}

func (b *bot) Tanks() {
	okTargets := scl.Units{}
	goodTargets := scl.Units{}
	allEnemies := b.AllEnemyUnits.Units()
	allEnemiesReady := allEnemies.Filter(scl.Ready)
	tanks := b.Groups.Get(Tanks).Units
	for _, enemy := range allEnemies {
		if enemy.Is(zerg.Larva, zerg.Egg, protoss.AdeptPhaseShift, terran.KD8Charge) {
			continue
		}
		okTargets.Add(enemy)
		if enemy.IsStructure() && !enemy.IsDefensive() {
			continue
		}
		goodTargets.Add(enemy)
	}

	for _, tank := range tanks {
		if tank.UnitType == terran.SiegeTank {
			if tank.Hits < 71 {
				b.Groups.Add(MechRetreat, tank)
				continue
			}

			// Keep range
			attackers := allEnemiesReady.CanAttack(tank, 2)
			if !scl.AttackDelay.UnitIsCool(tank) {
				closeTargets := goodTargets.InRangeOf(tank, -0.5)
				if attackers.Exists() || closeTargets.Exists() {
					tank.GroundFallback(attackers, 2, b.HomePaths)
					continue
				}

				/*retreat := attackers.Exists()
				if !retreat && goodTargets.Exists() {
					closestTarget := goodTargets.ClosestTo(tank.Point())
					retreat = tank.RangeDelta(closestTarget, -0.1) <= 0
				}
				if retreat {
					tank.GroundFallback(attackers, 2, b.HomePaths)
					continue
				}*/
			}

			targets := goodTargets.InRangeOf(tank, 0)
			if targets.Empty() {
				targets = okTargets.InRangeOf(tank, 0)
			}
			farTargets := goodTargets.InRangeOf(tank, 13-7) // Sieged range - mobile range
			if farTargets.Empty() {
				farTargets = okTargets.InRangeOf(tank, 13-7)
			}

			if targets.Empty() && farTargets.Exists() && attackers.Exists() {
				tank.Command(ability.Morph_SiegeMode)
				continue
			}

			if goodTargets.Exists() || okTargets.Exists() {
				tank.Attack(goodTargets, okTargets)
			} else {
				b.Explore(tank)
			}
		}
		if tank.UnitType == terran.SiegeTankSieged {
			farTargets := goodTargets.InRangeOf(tank, 2).Filter(func(unit *scl.Unit) bool {
				return unit.IsFurtherThan(float64(tank.Radius+unit.Radius+2), tank)
			})
			targets := farTargets.InRangeOf(tank, 0)
			if targets.Empty() {
				targets = okTargets.InRangeOf(tank, 0)
			}
			if targets.Exists() {
				tank.Attack(targets)
				continue
			}

			if farTargets.Empty() {
				tank.Command(ability.Morph_Unsiege)
			}
		}
	}
}

func (b *bot) Ravens() {
	ravens := b.Groups.Get(Ravens).Units
	if ravens.Empty() {
		return
	}

	friends := append(b.Groups.Get(Tanks).Units, b.Groups.Get(Cyclones).Units...)
	if friends.Empty() {
		friends = b.Groups.Get(WidowMines).Units
	}
	if friends.Empty() {
		friends = b.Groups.Get(Battlecruisers).Units
	}
	if friends.Empty() {
		friends = b.Groups.Get(Reapers).Units
	}
	if friends.Empty() {
		return
	}

	allEnemies := b.AllEnemyUnits.Units()
	allEnemiesReady := allEnemies.Filter(scl.Ready)
	enemiesCenter := allEnemiesReady.Center()
	friends.OrderBy(func(unit *scl.Unit) float64 {
		return unit.Point().Dist2(enemiesCenter)
	}, false)

	for n, raven := range ravens {
		if raven.Hits < 71 {
			b.Groups.Add(MechRetreat, raven)
			continue
		}

		if raven.Energy >= 50 {
			if raven.TargetAbility() == ability.Effect_AutoTurret {
				continue // Let him finish placing
			}
			closeEnemies := allEnemies.CloserThan(8, raven.Point())
			if closeEnemies.Exists() && closeEnemies.Sum(scl.CmpHits) >= 300 {
				pos := raven.Point().Towards(closeEnemies.Center(), 3)
				pos = b.FindClosestPos(pos, scl.S2x2, 0, 1, 1, scl.IsBuildable, scl.IsPathable)
				if pos != 0 {
					raven.CommandPos(ability.Effect_AutoTurret, pos.S2x2Fix())
					continue
				}
			}
		}

		pos := friends[scl.MinInt(n, len(friends)-1)].Point().Towards(enemiesCenter, 2)
		pos, safe := raven.AirEvade(allEnemiesReady.CanAttack(raven, 2), 2, pos)
		if !safe || pos.IsFurtherThan(1, raven.Point()) {
			raven.CommandPos(ability.Move, pos)
			continue
		}
	}
}

func (b *bot) Battlecruisers() {
	bcs := b.Groups.Get(Battlecruisers).Units
	if bcs.Empty() {
		return
	}

	okTargets := scl.Units{}
	goodTargets := scl.Units{}
	yamaTargets := scl.Units{}
	yamaFiring := map[api.UnitTag]int{}
	allEnemies := b.AllEnemyUnits.Units()
	// allEnemiesReady := allEnemies.Filter(scl.Ready)

	for _, enemy := range allEnemies {
		if enemy.Is(zerg.Larva, zerg.Egg, protoss.AdeptPhaseShift, terran.KD8Charge) {
			continue
		}
		okTargets.Add(enemy)
		if enemy.IsStructure() && !enemy.IsDefensive() {
			continue
		}
		goodTargets.Add(enemy)
		if enemy.AirDamage() > 0 && enemy.Hits > 120 { // Carrier?
			yamaTargets.Add(enemy)
		}
	}

	for _, bc := range bcs {
		if bc.TargetAbility() == ability.Effect_YamatoGun {
			yamaFiring[bc.TargetTag()]++
		}
	}

	for _, bc := range bcs {
		/*if bc.TargetAbility() == ability.Effect_YamatoGun || bc.TargetAbility() == ability.Effect_TacticalJump {
			continue
		}*/

		if (bc.HasAbility(ability.Effect_TacticalJump) && bc.Hits < 100) ||
			(!bc.HasAbility(ability.Effect_TacticalJump) && bc.Hits < 200) {
			cc := b.Units.OfType(scl.UnitAliases.For(terran.CommandCenter)...).Max(func(unit *scl.Unit) float64 {
				return float64(unit.AssignedHarvesters)
			})
			if cc != nil {
				b.Groups.Add(MechRetreat, bc)
				continue
			}
		}

		if yamaTargets.Exists() && bc.HasAbility(ability.Effect_YamatoGun) {
			targets := yamaTargets.InRangeOf(bc, 4).Filter(func(unit *scl.Unit) bool {
				return unit.Hits-float64(yamaFiring[unit.Tag]*240) > 0
			})
			if targets.Exists() {
				target := targets.Filter(func(unit *scl.Unit) bool {
					return unit.Hits-float64(yamaFiring[unit.Tag]*240) <= 240
				}).Max(scl.CmpHits)
				if target == nil {
					target = targets.Max(scl.CmpHits)
				}
				bc.CommandTag(ability.Effect_YamatoGun, target.Tag)
				yamaFiring[target.Tag]++
				continue
			}
		}

		// retreat is needed?

		if goodTargets.Exists() || okTargets.Exists() {
			bc.Attack(goodTargets, okTargets)
		} else {
			b.Explore(bc)
		}
	}
}

func (b *bot) MechRetreat() {
	us := b.Groups.Get(MechRetreat).Units
	if us.Empty() {
		return
	}
	enemies := b.AllEnemyUnits.Units().Filter(scl.Ready)
	scvs := b.Units[terran.SCV]
	var healingPoints []int
	for expNum, expLoc := range b.ExpLocs {
		if scvs.CloserThan(scl.ResourceSpreadDistance, expLoc).Len() < 4 {
			continue
		}
		healingPoints = append(healingPoints, expNum)
	}
	if len(healingPoints) == 0 {
		return
	}
	for _, u := range us {
		if u.Health == u.HealthMax {
			b.OnUnitCreated(u) // Add to corresponding group
			continue
		}
		// Find closest healing point
		var healingExp int
		var healingPoint scl.Point
		dist := math.Inf(1)
		for _, expNum := range healingPoints {
			newDist := u.Point().Dist2(b.ExpLocs[expNum])
			if newDist < dist {
				healingExp = expNum
				healingPoint = b.ExpLocs[expNum] - b.StartMinVec*3
				dist = newDist
			}
		}
		if u.UnitType == terran.Battlecruiser && u.HasAbility(ability.Effect_TacticalJump) {
			u.CommandPos(ability.Effect_TacticalJump, healingPoint)
			continue
		}
		if u.Point().IsCloserThan(4, healingPoint) {
			u.CommandPos(ability.Move, healingPoint) // For battlecruisers
			b.Groups.Add(MechHealing, u)
			continue
		}
		if u.UnitType == terran.Cyclone && u.HasAbility(ability.Effect_LockOn) {
			targets := enemies.Filter(scl.Visible).InRangeOf(u, 2)
			if targets.Exists() {
				CycloneAttackFunc(u, 0, targets)
				continue
			}
		}
		if u.UnitType == terran.SiegeTank {
			targets := enemies.Filter(scl.Visible).InRangeOf(u, 0)
			if targets.Exists() {
				u.Attack(targets)
				continue
			}
		}

		if u.Point().IsCloserThan(8, healingPoint) {
			u.CommandPos(ability.Move, healingPoint)
			continue
		}
		u.GroundFallback(enemies, 2, b.ExpPaths[healingExp])
	}
}

func (b *bot) Micro() {
	b.WorkerRushDefence()
	b.Marines()
	b.Reapers()
	b.Cyclones()
	b.Mines()
	b.Tanks()
	b.Ravens()
	b.Battlecruisers()
	b.MechRetreat()
}
