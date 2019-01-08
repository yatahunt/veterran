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
	if buildings := b.Units.Units().Filter(scl.Structure); buildings.Exists() {
		workersRange = math.Max(workersRange, buildings.FurthestTo(b.StartLoc).Point().Dist(b.StartLoc)+6)
	}

	workers := b.Units.OfType(terran.SCV).CloserThan(scl.ResourceSpreadDistance, b.StartLoc)
	enemies := b.EnemyUnits.Units().Filter(scl.NotFlying).CloserThan(enemiesRange, b.StartLoc)
	alert := enemies.CloserThan(enemiesRange-4, b.StartLoc).Exists()
	if enemies.Empty() || enemies.Sum(scl.CmpGroundScore) > workers.Sum(scl.CmpGroundScore) {
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
		vec := scl.Pt(1, 0).Rotate(math.Pi * 2.0 / 16.0 * x)
		np := p + vec*5
		var np2 scl.Point
		for x := 1.0; x <= 10; x++ {
			np2 := p + vec.Mul(x)
			if b.IsPathable(np2) {
				break
			}
		}
		maxJump := 1.0
		if u.Hits < u.HitsMax {
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
		if pos.IsCloserThan(float64(reaper.Radius)+reaper.GroundRange(), reaper.Point()) {
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
	allEnemiesReady := allEnemies.Filter(scl.Ready)
	reapers := b.Groups.Get(Reapers).Units
	for _, enemy := range allEnemies {
		if enemy.IsFlying || enemy.Is(zerg.Larva, zerg.Egg, protoss.AdeptPhaseShift, terran.KD8Charge) {
			continue
		}
		// Check if enemies that close to this one and have big range can kill reaper in a second
		enemiesDPS := allEnemiesReady.Filter(func(unit *scl.Unit) bool {
			return unit.GroundRange() >= 4 && unit.IsCloserThan(unit.GroundRange(), enemy)
		}).Sum(func(unit *scl.Unit) float64 {
			return unit.GroundDPS()
		})
		reapersDPS := reapers.CloserThan(15, enemy.Point()).Sum(func(unit *scl.Unit) float64 { return unit.GroundDPS() })
		if enemiesDPS >= 60 {
			if (!assault && (reapersDPS < enemiesDPS*2 && reapers.Len() <= 30)) ||
				(assault && (reapersDPS < enemiesDPS && reapers.Len() <= 20)) {
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
			b.Groups.Add(ReapersRetreat, reaper)
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
					b.ReaperFallback(reaper, goodTargets, b.EnemyStartLoc)
					continue
				}
			}
		}

		// Evade dangerous zones
		ep := reaper.Point()
		attackers := allEnemiesReady.CanAttack(reaper, 2)
		if !assault && attackers.Exists() && attackers.Sum(scl.CmpGroundDamage) >= reaper.Hits {
			ep = reaper.GroundEvade(append(hazards, attackers...), 2, reaper.Point())
		} else {
			ep = reaper.GroundEvade(hazards, 2, reaper.Point())
		}
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
	reapers = b.Groups.Get(ReapersRetreat).Units
	for _, reaper := range reapers {
		if reaper.Health > 30 {
			b.Groups.Add(Reapers, reaper)
			continue
		}
		// Use attack if enemy is close but can't attack reaper
		if scl.AttackDelay.IsCool(reaper.UnitType, reaper.WeaponCooldown, reaper.Bot.FramesPerOrder) &&
			(goodTargets.InRangeOf(reaper, 0).Exists() || okTargets.InRangeOf(reaper, 0).Exists()) &&
			allEnemiesReady.CanAttack(reaper, 1).Empty() {
			reaper.Attack(goodTargets, okTargets)
			continue
		}
		// Throw mine
		if b.ThrowMine(reaper, goodTargets) {
			continue
		}
		b.ReaperFallback(reaper, allEnemiesReady, b.StartLoc)
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
		// Move closer
		u.CommandPos(ability.Move, target.Point())
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

		retreat := cyclone.HPS > 0
		// Keep range
		// Weapon is recharging
		if !retreat && !cyclone.HasAbility(ability.Effect_LockOn) {
			// There is an enemy
			if closestEnemy := goodTargets.Filter(scl.Visible).ClosestTo(cyclone.Point()); closestEnemy != nil {
				// And it is closer than sight range - 2
				if cyclone.InRange(closestEnemy, 4) {
					// Retreat a little
					retreat = true
				}
			}
		}
		if retreat && !cyclone.HasAbility(ability.Effect_LockOn) {
			// pos := cyclone.GroundEvade(goodTargets, 2, b.StartLoc)
			cyclone.CommandPos(ability.Move, b.StartLoc)
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
		pos := u.GroundEvade(enemies, 1, hp)
		u.CommandPos(ability.Move, pos)
	}
}

func (b *bot) Micro() {
	b.WorkerRushDefence()
	b.Reapers()
	b.Cyclones()
	b.MechRetreat()
}
