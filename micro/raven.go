package micro

import (
	"bitbucket.org/aisee/sc2lib"
	"bitbucket.org/aisee/veterran/bot"
	"github.com/chippydip/go-sc2ai/enums/ability"
)

type Raven struct {
	*Unit

	Friend        *scl.Unit
	EnemiesCenter scl.Point
}

func NewRaven(u *scl.Unit) *Raven {
	return &Raven{Unit: NewUnit(u)}
}

func RavensLogic(us scl.Units) {
	if us.Empty() {
		return
	}

	friends := append(B.Groups.Get(bot.Tanks).Units, B.Groups.Get(bot.Cyclones).Units...)
	friends = append(friends, B.Groups.Get(bot.Marines).Units...)
	friends = append(friends, B.Groups.Get(bot.Marauders).Units...)
	if friends.Empty() {
		friends = B.Groups.Get(bot.WidowMines).Units
	}
	if friends.Empty() {
		friends = B.Groups.Get(bot.Battlecruisers).Units
	}
	if friends.Empty() {
		friends = B.Groups.Get(bot.Reapers).Units
	}
	if friends.Empty() {
		return
	}

	enemiesCenter := B.Enemies.AllReady.Center()
	friend := friends.ClosestTo(enemiesCenter)

	for _, u := range us {
		r := NewRaven(u)
		r.Friend = friend
		r.EnemiesCenter = enemiesCenter
		r.Logic()
	}
}

func (u *Raven) Retreat() bool {
	if u.TargetAbility() == ability.Effect_AutoTurret {
		return true // Let him finish placing
	}

	if u.Hits < u.HitsMax/2 {
		B.Groups.Add(bot.MechRetreat, u.Unit.Unit)
		return true
	}
	return false
}

// todo: придумать что-то со вторым юнитом в качестве якоря
func (u *Raven) Maneuver() bool {
	if u.Energy >= 50 {
		closeEnemies := B.Enemies.All.CloserThan(8, u)
		if closeEnemies.Exists() && closeEnemies.Sum(scl.CmpHits) >= 300 {
			pos := u.Towards(closeEnemies.Center(), 3)
			pos = B.FindClosestPos(pos, scl.S2x2, 0, 1, 1, scl.IsBuildable, scl.IsPathable)
			if pos != 0 {
				u.CommandPos(ability.Effect_AutoTurret, pos.S2x2Fix())
				return true
			}
		}
	}

	ravenPos := u.TargetPos()
	if ravenPos == 0 {
		ravenPos = u.Point()
	}
	pos := u.Friend.Towards(u.EnemiesCenter, 2)
	pos, safe := u.AirEvade(B.Enemies.AllReady.CanAttack(u.Unit.Unit, 2), 2, pos)
	if !safe || pos.IsFurtherThan(1, ravenPos) {
		u.CommandPos(ability.Move, pos)
	}
	return true
}

func (u *Raven) Attack() bool {
	return false
}
