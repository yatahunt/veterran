package micro

import (
	"bitbucket.org/aisee/sc2lib"
	"bitbucket.org/aisee/veterran/bot"
	"github.com/chippydip/go-sc2ai/enums/ability"
)

type Reaper struct {
	*Unit

	MfsPos, BasePos scl.Point
}

func NewReaper(u *scl.Unit) *Reaper {
	return &Reaper{Unit: NewUnit(u)}
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
		r := NewReaper(u)
		r.MfsPos = mfsPos
		r.BasePos = basePos
		r.Logic()
	}
}

func ReapersRetreatLogic(us scl.Units) {
	for _, u := range us {
		r := NewReaper(u)
		if r.Hits > r.HitsMax/2+10 {
			B.Groups.Add(bot.Reapers, r.Unit.Unit)
			continue
		}

		attackers := B.Enemies.AllReady.CanAttack(r.Unit.Unit, 2)
		closeOkTargets := Targets.ReaperOk.InRangeOf(r.Unit.Unit, 0)
		closeGoodTargets := Targets.ReaperGood.InRangeOf(r.Unit.Unit, 0)
		// Use attack if enemy is close but can't attack reaper
		if r.IsCool() && (closeGoodTargets.Exists() || closeOkTargets.Exists()) && attackers.Empty() {
			r.AttackCustom(scl.DefaultAttackFunc, ReaperMoveFunc, closeGoodTargets, closeOkTargets)
			continue
		}
		// Throw mine
		if r.ThrowMine(Targets.ReaperGood) {
			continue
		}

		r.GroundFallback(attackers, 2, B.HomeReaperPaths)
	}
}

func ReaperMoveFunc(u *scl.Unit, target *scl.Unit) {
	// Unit need to be closer to the target to shoot?
	if !u.InRange(target, -0.1) || !target.IsVisible() {
		u.AttackMove(target, bot.B.HomeReaperPaths)
	}
}

func (u *Reaper) ThrowMine(targets scl.Units) bool {
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

func (u *Reaper) Retreat() bool {
	if u.Hits < u.HitsMax/2 {
		B.Groups.Add(bot.ReapersRetreat, u.Unit.Unit)
		return true
	}
	return false
}

func (u *Reaper) Maneuver() bool {
	if !u.IsCool() {
		if u.ThrowMine(Targets.ReaperGood) {
			return true
		}

		// There is an enemy
		if closestEnemy := Targets.ReaperGood.Filter(scl.Visible).ClosestTo(u); closestEnemy != nil {
			// And it is closer than shooting distance - 0.5
			if u.InRange(closestEnemy, -0.5) {
				// Retreat a little
				attackers := B.Enemies.AllReady.CanAttack(u.Unit.Unit, 2)
				u.GroundFallback(attackers, -0.5, B.HomeReaperPaths)
				return true
			}
		}
	}
	return false
}

func (u *Reaper) Attack() bool {
	closeTargets := Targets.ReaperGood.InRangeOf(u.Unit.Unit, 0)
	if u.MfsPos != 0 && !B.IsExplored(u.MfsPos) && closeTargets.Empty() {
		u.CommandPos(ability.Move, u.MfsPos)
		return true
	}
	if u.BasePos != 0 && !B.IsExplored(u.BasePos) && closeTargets.Empty() {
		u.CommandPos(ability.Move, u.BasePos)
		return true
	}
	if Targets.ReaperGood.Exists() || Targets.ReaperOk.Exists() {
		u.AttackCustom(scl.DefaultAttackFunc, ReaperMoveFunc, Targets.ReaperGood, Targets.ReaperOk)
		return true
	}
	return false
}
