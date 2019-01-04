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
		unit.Attack(enemies)
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
	reapers = b.Groups.Get(Retreat).Units
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

func (b *bot) Micro() {
	b.WorkerRushDefence()
	b.Reapers()
}
