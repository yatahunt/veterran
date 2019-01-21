package main

import (
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

func WorkerMoveFunc(u *scl.Unit, target *scl.Unit) {
	if !u.InRange(target, 0) || !target.IsVisible() {
		if u.WeaponCooldown > 0 {
			u.SpamCmds = true
		}
		u.CommandPos(ability.Move, target.Point())
	}
}

func (b *bot) WorkerRushDefence() {
	enemiesRange := 12.0
	workersRange := 10.0
	enemyWorkers := b.EnemyUnits.OfType(terran.SCV, zerg.Drone, protoss.Probe)
	if workerRush {
		workersRange = 50.0
	} else if building := b.Units.Units().Filter(scl.Structure).ClosestTo(b.MainRamp.Top); building != nil {
		workersRange = math.Max(workersRange, building.Point().Dist(b.StartLoc)+6)
	}

	workers := b.Units.OfType(terran.SCV).CloserThan(scl.ResourceSpreadDistance, b.StartLoc)
	enemies := b.EnemyUnits.Units().Filter(scl.NotFlying).CloserThan(enemiesRange, b.StartLoc)
	alert := enemies.CloserThan(enemiesRange-4, b.StartLoc).Exists()
	if enemies.Empty() || enemies.Sum(scl.CmpGroundScore) > workers.Sum(scl.CmpGroundScore)*2 || workerRush {
		enemies = enemyWorkers.CloserThan(workersRange, b.StartLoc)
		alert = enemies.CloserThan(workersRange-4, b.StartLoc).Exists()
		if alert && enemies.Len() >= 10 {
			workerRush = true
		}
	}
	if workerRush && enemyWorkers.CloserThan(70, b.StartLoc).Empty() {
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
			unit.AttackCustom(scl.DefaultAttackFunc, WorkerMoveFunc, enemies)
		} else {
			friends := army.InRangeOf(unit, 0)
			friend := friends.Min(scl.CmpHits)
			if friend != nil && friend.Hits < 45 && b.Minerals > 0 {
				unit.CommandTag(ability.Effect_Repair_SCV, friend.Tag)
			}
		}
	}

	if workerRush && b.Minerals >= 75 {
		workers := b.Groups.Get(Miners).Units.Filter(func(unit *scl.Unit) bool {
			return unit.Hits < 11 && unit.IsGathering()
		})
		if workers.Len() >= 2 {
			workers[0].CommandTag(ability.Effect_Repair_SCV, workers[1].Tag)
			workers[1].CommandTag(ability.Effect_Repair_SCV, workers[0].Tag)
			newGroup := b.Groups.New(workers[0], workers[1])
			doubleHealers = append(doubleHealers, newGroup)
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
	if playDefensive {
		pos := b.MainRamp.Top
		bunker := b.Units[terran.Bunker].ClosestTo(b.ExpLocs[0])
		if bunker != nil {
			pos = bunker.Point()
		}
		if u.IsFarFrom(pos) {
			u.CommandPos(ability.Move, pos)
		}
		return
	}
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
		if playDefensive && enemy.Point().IsFurtherThan(defensiveRange, b.StartLoc) {
			continue
		}
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

		// Load into a bunker
		if goodTargets.InRangeOf(marine, 0).Empty() {
			bunker := b.getEmptyBunker(marine.Point())
			if bunker != nil {
				if bunker.IsReady() {
					marine.CommandTag(ability.Smart, bunker.Tag)
				} else if marine.IsFarFrom(bunker.Point()) {
					marine.CommandPos(ability.Move, bunker.Point())
				}
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
	var mfsPos scl.Point
	if b.Loop < 3360 { // For exp recon before 2:30
		mfsPos = b.MineralFields.Units().CloserThan(scl.ResourceSpreadDistance, b.EnemyExpLocs[0]).Center()
	}
	okTargets := scl.Units{}
	goodTargets := scl.Units{}
	allEnemies := b.AllEnemyUnits.Units()
	allEnemiesReady := allEnemies.Filter(scl.Ready)

	reapers := b.Groups.Get(Reapers).Units
	for _, enemy := range allEnemies {
		if playDefensive && enemy.Point().IsFurtherThan(defensiveRange, b.StartLoc) {
			continue
		}
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
		if reaper.Hits < 27 {
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
			if closestEnemy := goodTargets.Filter(scl.Visible).ClosestTo(reaper.Point()); closestEnemy != nil {
				// And it is closer than shooting distance - 0.5
				if reaper.InRange(closestEnemy, -0.5) {
					// Retreat a little
					reaper.GroundFallback(goodTargets, -0.5, b.HomeReaperPaths)
					continue
				}
			}

			/*attackers := allEnemiesReady.CanAttack(reaper, 2)
			closeTargets := goodTargets.InRangeOf(reaper, -0.5)
			if attackers.Exists() || closeTargets.Exists() {
				reaper.GroundFallback(attackers, 2, b.HomeReaperPaths)
				continue
			}*/
		}

		if mfsPos != 0 && !b.IsExplored(mfsPos) && goodTargets.InRangeOf(reaper, 0).Empty() {
			reaper.CommandPos(ability.Move, mfsPos)
		} else if goodTargets.Exists() || okTargets.Exists() {
			reaper.AttackCustom(scl.DefaultAttackFunc, ReaperMoveFunc, goodTargets, okTargets)
		} else {
			b.Explore(reaper)
		}
	}

	// Damaged reapers
	reapers = b.Groups.Get(ReapersRetreat).Units
	for _, reaper := range reapers {
		if reaper.Health > 36 {
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
		if playDefensive && enemy.Point().IsFurtherThan(defensiveRange, b.StartLoc) {
			continue
		}
		if enemy.Is(zerg.Larva, zerg.Egg, protoss.AdeptPhaseShift, terran.KD8Charge) {
			continue
		}
		okTargets.Add(enemy)
		if enemy.IsStructure() && !enemy.IsDefensive() {
			continue
		}
		goodTargets.Add(enemy)
		if !enemy.IsFlying || enemy.UnitType == zerg.Overlord || enemy.UnitType == zerg.LocustMP {
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
		if playDefensive && enemy.Point().IsFurtherThan(defensiveRange, b.StartLoc) {
			continue
		}
		if enemy.IsStructure() || enemy.Is(zerg.Larva, zerg.Egg, protoss.AdeptPhaseShift, terran.KD8Charge) {
			continue
		}
		targets.Add(enemy)
	}

	for _, mine := range mines {
		attackers := allEnemies.CanAttack(mine, 2)
		// Someone is attacking mine, but it can't attack anyone
		detected := false
		if mine.HPS > 0 {
			// detected = targets.InRangeOf(mine, 0).Empty() - this is wrong? Mine has no weapon?
			detected = targets.First(func(unit *scl.Unit) bool {
				return mine.Point().Dist(unit.Point()) <= float64(mine.Radius+unit.Radius+5)
			}) == nil
		}
		safePos, safe := mine.EvadeEffectsPos(mine.Point(), false, effect.PsiStorm, effect.CorrosiveBile)
		if !safe {
			detected = true
		}

		if mine.IsBurrowed && (detected ||
			!mine.HasAbility(ability.Smart) || // Reloading
			targets.CloserThan(10, mine.Point()).Empty() && attackers.Empty()) {
			// No targets or enemies around
			if mine.Hits < mine.HitsMax/2 {
				b.Groups.Add(MechRetreat, mine)
			} else {
				b.Groups.Add(WidowMinesRetreat, mine)
			}
			mine.Command(ability.BurrowUp_WidowMine)
			continue
		}

		if !safe {
			mine.CommandPos(ability.Move, safePos)
			continue
		}

		targetIsClose := targets.CloserThan(4, mine.Point()).Exists() // For enemies that can't attack ground
		if !mine.IsBurrowed && !detected && (attackers.Exists() || targetIsClose) {
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

func (b *bot) Hellions() {
	okTargets := scl.Units{}
	goodTargets := scl.Units{}
	allEnemies := b.AllEnemyUnits.Units()
	allEnemiesReady := allEnemies.Filter(scl.Ready)
	hellions := b.Groups.Get(Hellions).Units
	for _, enemy := range allEnemies {
		if playDefensive && enemy.Point().IsFurtherThan(defensiveRange, b.StartLoc) {
			continue
		}
		if enemy.IsFlying || enemy.Is(zerg.Larva, zerg.Egg, protoss.AdeptPhaseShift, terran.KD8Charge) {
			continue
		}
		okTargets.Add(enemy)
		if !enemy.IsStructure() {
			goodTargets.Add(enemy)
		}
	}

	for _, hellion := range hellions {
		if hellion.Hits < 31 {
			b.Groups.Add(MechRetreat, hellion)
			continue
		}

		if !scl.AttackDelay.UnitIsCool(hellion) {
			attackers := allEnemiesReady.CanAttack(hellion, 2)
			closeTargets := goodTargets.InRangeOf(hellion, -0.5)
			if attackers.Exists() || closeTargets.Exists() {
				hellion.GroundFallback(attackers, 2, b.HomePaths)
				continue
			}
		}

		// Attack
		if goodTargets.Exists() || okTargets.Exists() {
			hellion.Attack(goodTargets, okTargets)
		} else {
			b.Explore(hellion)
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
		if playDefensive && enemy.Point().IsFurtherThan(defensiveRange, b.StartLoc) {
			continue
		}
		if enemy.IsFlying || enemy.Is(zerg.Larva, zerg.Egg, protoss.AdeptPhaseShift, terran.KD8Charge) {
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
		if playDefensive && enemy.Point().IsFurtherThan(defensiveRange, b.StartLoc) {
			continue
		}
		if enemy.Is(zerg.Larva, zerg.Egg, protoss.AdeptPhaseShift, terran.KD8Charge) {
			continue
		}
		okTargets.Add(enemy)
		if enemy.IsStructure() && !enemy.IsDefensive() {
			continue
		}
		goodTargets.Add(enemy)
		if enemy.AirDamage() > 0 && enemy.Hits > 120 || enemy.UnitType == protoss.Carrier ||
			enemy.UnitType == zerg.Ultralisk || enemy.UnitType == zerg.Viper || enemy.UnitType == zerg.Infestor {
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
			(!bc.HasAbility(ability.Effect_TacticalJump) && bc.Hits < 250) {
			b.Groups.Add(MechRetreat, bc)
			continue
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
			/*cc := b.Units.OfType(scl.UnitAliases.For(terran.CommandCenter)...).Max(func(unit *scl.Unit) float64 {
				return float64(unit.AssignedHarvesters)
			})
			if cc != nil {
				u.CommandPos(ability.Effect_TacticalJump, cc.Point() - b.StartMinVec * 3)
			} else {
				u.CommandPos(ability.Effect_TacticalJump, healingPoint)
			}*/
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
		if u.IsFlying {
			pos, _ := u.AirEvade(enemies, 2, healingPoint)
			u.CommandPos(ability.Move, pos)
		} else {
			u.GroundFallback(enemies, 2, b.ExpPaths[healingExp])
		}
	}
}

func (b *bot) Micro() {
	b.WorkerRushDefence()
	b.Marines()
	b.Reapers()
	b.Cyclones()
	b.Mines()
	b.Hellions()
	b.Tanks()
	b.Ravens()
	b.Battlecruisers()
	b.MechRetreat()
}
