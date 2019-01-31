package micro

import (
	"bitbucket.org/aisee/sc2lib"
	"bitbucket.org/aisee/veterran/bot"
	"github.com/chippydip/go-sc2ai/enums/ability"
)

func ReaperMoveFunc(u *scl.Unit, target *scl.Unit) {
	// Unit need to be closer to the target to shoot?
	if !u.InRange(target, -0.1) || !target.IsVisible() {
		u.AttackMove(target, bot.B.HomeReaperPaths)
	}
}

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
	if u.Hits < u.HitsMax/2 {
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
				attackers := B.Enemies.AllReady.CanAttack(u, 2)
				u.GroundFallback(attackers, -0.5, B.HomeReaperPaths)
				return true
			}
		}
	}
	// iscool vs lings
	if !u.IsCool() {
		if closestEnemy := Targets.ReaperGood.Filter(scl.Visible).ClosestTo(u); closestEnemy != nil {
			// And it is closer than shooting distance -2
			if u.InRange(closestEnemy, -2) {
				// Retreat
				attackers := B.Enemies.AllReady.CanAttack(u, 2)
				u.GroundFallback(attackers, -2, B.HomeReaperPaths)
				return true
			}
		}
	}
	return false
}

func ReaperAttack(u *scl.Unit, mfsPos, basePos scl.Point) bool {
	closeTargets := Targets.ReaperGood.InRangeOf(u, 2)
	if mfsPos != 0 && !B.IsExplored(mfsPos) && closeTargets.Empty() {
		u.CommandPos(ability.Move, mfsPos)
		return true
	}
	if basePos != 0 && !B.IsExplored(basePos) && closeTargets.Empty() {
		u.CommandPos(ability.Move, basePos)
		return true
	}
	if Targets.ReaperOk.Exists() {
		u.AttackCustom(scl.DefaultAttackFunc, ReaperMoveFunc, Targets.ReaperGood, Targets.ReaperOk)
		return true
	}
	return false
}

func ReapersLogic(us scl.Units) {
	if us.Empty() {
		return
	}

	var mfsPos, basePos scl.Point
	// For exp recon before 4:00
	if B.Loop < 5376 && B.Enemies.All.CloserThan(B.DefensiveRange, B.Locs.MyStart).Empty() {
		mfsPos = B.Units.Minerals.All().CloserThan(scl.ResourceSpreadDistance, B.Locs.EnemyExps[0]).Center()
		basePos = B.Locs.EnemyStart
	}

	for _, u := range us {
		_ = ReaperRetreat(u) || ReaperManeuver(u) || MarauderStim(u) || ReaperAttack(u, mfsPos, basePos) ||
			DefaultExplore(u)
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
			u.AttackCustom(scl.DefaultAttackFunc, ReaperMoveFunc, closeGoodTargets, closeOkTargets)
			continue
		}
		// Throw mine
		if ThrowMine(u, Targets.ReaperGood) {
			continue
		}

		u.GroundFallback(attackers, 2, B.HomeReaperPaths)
	}
}
