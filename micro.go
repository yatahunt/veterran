package main

import (
	"bitbucket.org/aisee/sc2lib"
	"github.com/chippydip/go-sc2ai/enums/ability"
	"github.com/chippydip/go-sc2ai/enums/protoss"
	"github.com/chippydip/go-sc2ai/enums/terran"
	"github.com/chippydip/go-sc2ai/enums/zerg"
	"math"
	"math/rand"
)

func (b *bot) WorkerRushDefence() {
	enemiesRange := 12.0
	workersRange := 12.0
	if building := b.Units.Units().Filter(scl.Structure).ClosestTo(b.MainRamp.Top); building != nil {
		workersRange = math.Max(workersRange, building.Point().Dist(b.StartLoc)+6)
	}

	workers := b.Units.OfType(terran.SCV).CloserThan(scl.ResourceSpreadDistance, b.StartLoc)
	enemies := b.EnemyUnits.Units().Filter(scl.NotFlying).CloserThan(enemiesRange, b.StartLoc)
	alert := enemies.CloserThan(enemiesRange-4, b.StartLoc).Exists()
	if enemies.Empty() || enemies.Sum(scl.CmpGroundScore) > workers.Sum(scl.CmpGroundScore)*2 {
		enemies = b.EnemyUnits.OfType(terran.SCV, zerg.Drone, protoss.Probe).CloserThan(workersRange, b.StartLoc)
		alert = enemies.CloserThan(workersRange-4, b.StartLoc).Exists()
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
		if u.WeaponCooldown > 0 {
			// Spamming this thing is the key. Or orders will be ignored (or postponed)
			u.SpamCmds = true
		}
		enemies := u.Bot.AllEnemyUnits.Units()
		pos, safe := u.GroundFallbackPos(enemies, 2, u.Bot.EnemyReaperPaths, 2)
		if !safe {
			friendsDPS := u.Bot.Units.Units().CloserThan(safeBuildRange, u.Point()).Sum(scl.CmpGroundDPS)
			enemiesDPS := enemies.CloserThan(safeBuildRange, target.Point()).Sum(scl.CmpGroundDPS)
			if friendsDPS >= enemiesDPS {
				safe = true
			}
		}
		if safe {
			// Move closer
			u.CommandPos(ability.Move, target.Point())
		} else {
			u.CommandPos(ability.Move, pos)
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
			if closestEnemy := goodTargets.Filter(scl.Visible).ClosestTo(reaper.Point()); closestEnemy != nil {
				// And it is closer than shooting distance - 0.5
				if reaper.InRange(closestEnemy, -0.5) {
					// Retreat a little
					reaper.GroundFallback(goodTargets, -0.5, b.HomeReaperPaths)
					continue
				}
			}
		}

		// Attack
		if goodTargets.Exists() || okTargets.Exists() {
			reaper.AttackCustom(scl.DefaultAttackFunc, ReaperMoveFunc, goodTargets, okTargets /*, hazards*/)
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
		if u.WeaponCooldown > 0 {
			// Spamming this thing is the key. Or orders will be ignored (or postponed)
			u.SpamCmds = true
		}
		// copypaste here, todo: remove
		enemies := u.Bot.AllEnemyUnits.Units()
		pos, safe := u.GroundFallbackPos(enemies, 2, u.Bot.EnemyReaperPaths, 2)
		if !safe {
			friendsDPS := u.Bot.Units.Units().CloserThan(safeBuildRange, u.Point()).Sum(scl.CmpGroundDPS)
			enemiesDPS := enemies.CloserThan(safeBuildRange, target.Point()).Sum(scl.CmpGroundDPS)
			if friendsDPS >= enemiesDPS {
				safe = true
			}
		}
		if safe {
			// Move closer
			u.CommandPos(ability.Move, target.Point())
		} else {
			u.CommandPos(ability.Move, pos)
		}
	}
}

func (b *bot) Cyclones() {
	okTargets := scl.Units{}
	goodTargets := scl.Units{}
	airTargets := scl.Units{}
	allEnemies := b.AllEnemyUnits.Units()
	// allEnemiesReady := allEnemies.Filter(scl.Ready)
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
		attackers := goodTargets.CanAttack(cyclone, 2)
		retreat := cyclone.HPS > 0 && attackers.Exists()
		if !retreat && !cyclone.HasAbility(ability.Effect_LockOn) && attackers.Exists() {
			target := allEnemies.ByTag(cyclone.EngagedTargetTag)
			// Someone is locked on and he is close enough
			retreat = target != nil && cyclone.InRange(target, 5)
		}
		if retreat {
			cyclone.GroundFallback(goodTargets, 2, b.HomePaths)
			continue
		}

		// Attack
		if airTargets.Exists() || goodTargets.Exists() || okTargets.Exists() {
			cyclone.AttackCustom(CycloneAttackFunc, CycloneMoveFunc, airTargets, goodTargets, okTargets)
		} else {
			// Copypaste
			if !b.IsExplored(b.EnemyStartLoc) {
				cyclone.CommandPos(ability.Attack, b.EnemyStartLoc)
			} else {
				// Search for other bases
				if cyclone.IsIdle() {
					pos := b.EnemyExpLocs[rand.Intn(len(b.EnemyExpLocs))]
					cyclone.CommandPos(ability.Move, pos)
				}
			}
		}
	}
}

func (b *bot) Mines() {
	targets := scl.Units{}
	detectors := scl.Units{}
	allEnemies := b.AllEnemyUnits.Units()
	// allEnemiesReady := allEnemies.Filter(scl.Ready)
	mines := b.Groups.Get(Mines).Units
	for _, enemy := range allEnemies {
		if enemy.DetectRange > 0 {
			detectors.Add(enemy)
		}
		if enemy.IsStructure() || enemy.Is(zerg.Larva, zerg.Egg, protoss.AdeptPhaseShift, terran.KD8Charge) {
			continue
		}
		targets.Add(enemy)
	}

	for _, mine := range mines {
		if mine.IsBurrowed && !mine.HasAbility(ability.Smart) { // Reloading
			if mine.Hits < mine.HitsMax {
				b.Groups.Add(MechRetreat, mine)
			} else {
				b.Groups.Add(MinesRetreat, mine)
			}
			mine.Command(ability.BurrowUp_WidowMine)
			continue
		}

		attackers := allEnemies.CanAttack(mine, 2)
		if mine.IsBurrowed && targets.CloserThan(10, mine.Point()).Empty() && attackers.Empty() {
			mine.Command(ability.BurrowUp_WidowMine)
			continue
		}

		if attackers.Exists() && !mine.IsBurrowed {
			mine.Command(ability.BurrowDown_WidowMine)
			continue
		}

		if targets.Exists() {
			mine.CommandPos(ability.Move, targets.ClosestTo(mine.Point()).Point())
		} else {
			// Copypaste
			if !b.IsExplored(b.EnemyStartLoc) {
				mine.CommandPos(ability.Attack, b.EnemyStartLoc)
			} else {
				// Search for other bases
				if mine.IsIdle() {
					pos := b.EnemyExpLocs[rand.Intn(len(b.EnemyExpLocs))]
					mine.CommandPos(ability.Move, pos)
				}
			}
		}
	}

	mines = b.Groups.Get(MinesRetreat).Units
	for _, mine := range mines {
		if mine.Hits < mine.HitsMax {
			b.Groups.Add(MechRetreat, mine)
			continue
		}
		if mine.IsBurrowed && mine.HasAbility(ability.Smart) {
			b.Groups.Add(Mines, mine)
			continue
		}
		if mine.IsIdle() {
			pos := b.EnemyExpLocs[rand.Intn(len(b.EnemyExpLocs))]
			mfs := b.NeutralUnits.Units().CloserThan(scl.ResourceSpreadDistance, pos).Filter(scl.Mineral)
			if mfs.Exists() {
				pos = pos.Towards(mfs.Center(), 4)
			}
			mine.CommandPos(ability.Move, pos)
			mine.CommandQueue(ability.BurrowDown_WidowMine)
		}
	}
}

func (b *bot) MechRetreat() {
	us := b.Groups.Get(MechRetreat).Units
	if us.Empty() {
		return
	}
	enemies := b.AllEnemyUnits.Units().Filter(scl.Ready)
	ccs := b.Units.OfType(terran.CommandCenter, terran.OrbitalCommand, terran.PlanetaryFortress).Filter(scl.Ready)
	scvs := b.Units[terran.SCV]
	mfs := b.MineralFields.Units()
	var healingPoints scl.Points
	for _, cc := range ccs {
		if scvs.CloserThan(scl.ResourceSpreadDistance, cc.Point()).Len() < 4 {
			continue
		}
		healingPoints.Add(mfs.CloserThan(scl.ResourceSpreadDistance, cc.Point()).Center().Towards(cc.Point(), 2))
	}
	if healingPoints.Empty() {
		return
	}
	for _, u := range us {
		if u.Health == u.HealthMax {
			b.OnUnitCreated(u) // Add to corresponding group
			continue
		}
		hp := healingPoints.ClosestTo(u.Point())
		if u.Point().IsCloserThan(2, hp) {
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
		pos := u.GroundEvade(enemies, 1, hp)
		u.CommandPos(ability.Move, pos)
	}
}

func (b *bot) Micro() {
	b.WorkerRushDefence()
	b.Reapers()
	b.Cyclones()
	b.Mines()
	b.MechRetreat()
}
