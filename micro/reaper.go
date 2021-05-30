package micro

import (
	"bitbucket.org/aisee/veterran/bot"
	"github.com/aiseeq/s2l/lib/point"
	"github.com/aiseeq/s2l/lib/scl"
	"github.com/aiseeq/s2l/protocol/enums/ability"
	"github.com/aiseeq/s2l/protocol/enums/buff"
)

func ThrowMine(u *scl.Unit, targets scl.Units) bool {
	closestEnemy := targets.ClosestTo(u)
	if closestEnemy != nil && u.HasAbility(ability.Effect_KD8Charge) {
		// pos := closestEnemy.EstimatePositionAfter(50)
		pos := closestEnemy.Point()
		if pos.IsCloserThan(float64(u.Radius)+u.GroundRange(), u) {
			u.CommandPos(ability.Effect_KD8Charge, pos)
			return true
		}
	}
	return false
}

func ReaperRetreat(u *scl.Unit) bool {
	if u.Hits < u.HitsMax/2 || u.HasBuff(buff.LockOn) {
		B.Groups.Add(bot.ReapersRetreat, u)
		return true
	}
	return false
}

func ReaperManeuver(u *scl.Unit) bool {
	if !u.IsHalfCool() {
		if ThrowMine(u, Targets.ReaperGood) {
			return true
		}

		// There is an enemy
		if closestEnemy := Targets.ReaperGood.Filter(scl.Visible).ClosestTo(u); closestEnemy != nil {
			// And it is closer than shooting distance -0.5
			if u.InRange(closestEnemy, -0.5) {
				// Retreat a little
				u.GroundFallback(B.Enemies.AllReady, 0.5, B.Locs.MyStart-B.Locs.MyStartMinVec*3)
				return true
			}
		}
	} else if !u.IsCool() { // iscool vs lings
		if closestEnemy := Targets.ReaperGood.Filter(scl.Visible).ClosestTo(u); closestEnemy != nil {
			// And it is closer than shooting distance -0.5
			if u.InRange(closestEnemy, -0.5) {
				// Retreat
				u.GroundFallback(B.Enemies.AllReady, 0, B.Locs.MyStart-B.Locs.MyStartMinVec*3)
				return true
			}
		}
	}
	return false
}

func ReaperAttack(u *scl.Unit, mfsPos, basePos point.Point) bool {
	closeTargets := Targets.ReaperGood.InRangeOf(u, 2)
	if closeTargets.Empty() {
		if mfsPos != 0 && !B.Grid.IsExplored(mfsPos) {
			u.CommandPos(ability.Move, mfsPos)
			return true
		}
		if basePos != 0 && !B.Grid.IsExplored(basePos) {
			u.CommandPos(ability.Move, basePos)
			return true
		}
	}
	if Targets.ReaperOk.Exists() {
		u.Attack(Targets.ReaperGood, Targets.ReaperOk)
		return true
	}
	return false
}

func ReapersLogic(us scl.Units) {
	if us.Empty() {
		return
	}

	var mfsPos, basePos point.Point
	// For exp recon before 4:00
	if B.Loop < 5376 && Targets.ReaperOk.CloserThan(B.DefensiveRange, B.Locs.MyStart).Empty() {
		mfsPos = B.Units.Minerals.All().CloserThan(scl.ResourceSpreadDistance, B.Locs.EnemyExps[0]).Center()
		basePos = B.Locs.EnemyStart
	}

	for _, u := range us {
		_ = ReaperRetreat(u) || ReaperManeuver(u) || ReaperAttack(u, mfsPos, basePos) || DefaultExplore(u)
	}
}

func ReapersRetreatLogic(us scl.Units) {
	for _, u := range us {
		if u.Hits > u.HitsMax/2+10 {
			B.Groups.Add(bot.Reapers, u)
			continue
		}

		attackers := B.Enemies.AllReady.CanAttack(u, 2)
		closeOkTargets := Targets.ReaperOk.InRangeOf(u, 0)
		closeGoodTargets := Targets.ReaperGood.InRangeOf(u, 0)
		// Use attack if enemy is close but can't attack reaper
		if u.IsCool() && (closeGoodTargets.Exists() || closeOkTargets.Exists()) && attackers.Empty() {
			u.Attack(closeGoodTargets, closeOkTargets)
			continue
		}
		// Throw mine
		if ThrowMine(u, Targets.ReaperGood) {
			continue
		}

		u.GroundFallback(attackers, 2, B.Locs.MyStart-B.Locs.MyStartMinVec*3)
	}
}
