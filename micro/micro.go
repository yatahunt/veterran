package micro

import (
	"bitbucket.org/aisee/veterran/bot"
	"github.com/aiseeq/s2l/lib/point"
	"github.com/aiseeq/s2l/lib/scl"
	"github.com/aiseeq/s2l/protocol/enums/ability"
	"github.com/aiseeq/s2l/protocol/enums/buff"
	"github.com/aiseeq/s2l/protocol/enums/protoss"
	"github.com/aiseeq/s2l/protocol/enums/terran"
	"github.com/aiseeq/s2l/protocol/enums/zerg"
	"math"
)

type TargetsTypes struct {
	All                     scl.Units
	Flying                  scl.Units
	Ground                  scl.Units
	Armed                   scl.Units
	ArmedArmored            scl.Units
	ArmedFlying             scl.Units
	ArmedFlyingArmored      scl.Units
	ArmedGround             scl.Units
	ArmedGroundArmored      scl.Units
	ArmedGroundLight        scl.Units
	ArmedGroundNotBuildings scl.Units
	AntiAir                 scl.Units
	ReaperOk                scl.Units
	ReaperGood              scl.Units
	ForMines                scl.Units
	ForYamato               scl.Units
	MyGround                scl.Units // My units that can be damaged by my tanks with splash
}

var B *bot.Bot
var Targets TargetsTypes

func InitTargets() {
	Targets = TargetsTypes{}
	for _, u := range B.Enemies.All {
		if u.Is(zerg.Larva, zerg.Egg, protoss.AdeptPhaseShift, protoss.DisruptorPhased, terran.KD8Charge) {
			continue
		}

		if !u.IsFlying {
			Targets.ReaperOk.Add(u)
			if u.IsArmed() {
				Targets.ReaperGood.Add(u)
			}
		}
		if B.PlayDefensive && u.IsFurtherThan(B.DefensiveRange, B.Locs.MyStart) {
			continue
		}

		Targets.All.Add(u)
		if u.IsArmed() {
			Targets.Armed.Add(u)
			if u.IsArmored() {
				Targets.ArmedArmored.Add(u)
			}
			if u.AirDamage() > 0 {
				Targets.AntiAir.Add(u)
			}
		}
		if u.IsFlying || u.UnitType == protoss.Colossus {
			Targets.Flying.Add(u)
			if u.IsArmed() {
				Targets.ArmedFlying.Add(u)
				if u.IsArmored() {
					Targets.ArmedFlyingArmored.Add(u)
				}
			}
		}
		if !u.IsFlying {
			Targets.Ground.Add(u)
			if u.IsArmed() {
				Targets.ArmedGround.Add(u)
				if u.IsArmored() {
					Targets.ArmedGroundArmored.Add(u)
				} else if u.IsLight() {
					Targets.ArmedGroundLight.Add(u)
				}
				if !u.IsStructure() {
					Targets.ArmedGroundNotBuildings.Add(u)
				}
			}
		}
		if !u.IsStructure() {
			Targets.ForMines.Add(u)
		}
		if u.AirDamage() > 0 && u.Hits > 120 || u.UnitType == protoss.Carrier || u.UnitType == zerg.Ultralisk ||
			u.UnitType == zerg.Viper || u.UnitType == zerg.Infestor {
			Targets.ForYamato.Add(u)
		}
	}
	for _, u := range B.Units.My.All() {
		if !u.IsFlying {
			Targets.MyGround.Add(u)
		}
	}
}

func WorkerMoveFunc(u *scl.Unit, target *scl.Unit) { // todo: move into scv.go?
	if !u.InRange(target, 0) || !target.IsVisible() {
		if u.WeaponCooldown > 0 && u.PosDelta == 0 {
			u.SpamCmds = true
		}
		u.CommandPos(ability.Move, target)
	}
}

func WorkerRushDefence() {
	enemiesRange := 12.0
	workersRange := 10.0
	enemyWorkers := B.Units.Enemy.OfType(terran.SCV, zerg.Drone, protoss.Probe)
	if B.WorkerRush {
		workersRange = 50.0
	} else if building := B.Units.My.All().Filter(scl.Structure).ClosestTo(B.Ramps.My.Top); building != nil {
		workersRange = math.Max(workersRange, building.Dist(B.Locs.MyStart)+6)
	}

	if (B.ProxyReapers || B.ProxyMarines || B.BruteForce) &&
		enemyWorkers.CloserThan(B.Locs.MyStart.Dist(B.Locs.MapCenter), B.Locs.MyStart).Len() >= 10 {
		// Worker rush, probably. Disable cheeze
		B.Groups.Add(bot.Miners, B.Groups.Get(bot.ScvReserve).Units...)
		B.Groups.Add(bot.Miners, B.Groups.Get(bot.ProxyBuilders).Units...)
		B.ProxyReapers = false
		B.ProxyMarines = false
		B.BruteForce = false
		B.CcAfterRax = false
		B.CcBeforeRax = false
	}

	workers := B.Units.My[terran.SCV].CloserThan(scl.ResourceSpreadDistance, B.Locs.MyStart)
	enemies := B.Enemies.Visible.Filter(scl.Ground).CloserThan(enemiesRange, B.Locs.MyStart)
	alert := enemies.CloserThan(enemiesRange-4, B.Locs.MyStart).Exists()
	if enemies.Empty() || enemies.Sum(scl.CmpGroundScore) > workers.Sum(scl.CmpGroundScore)*2 || B.WorkerRush {
		enemies = enemyWorkers.CloserThan(workersRange, B.Locs.MyStart)
		alert = enemies.CloserThan(workersRange-4, B.Locs.MyStart).Exists()
		if alert && enemies.Len() >= 10 {
			B.WorkerRush = true
			bot.DisableDefensivePlay()
		}
	}
	if B.WorkerRush && enemyWorkers.CloserThan(70, B.Locs.MyStart).Empty() {
		B.WorkerRush = false
	}

	army := B.Groups.Get(bot.WorkerRushDefenders).Units
	if army.Exists() && enemies.Empty() {
		B.Groups.Add(bot.Miners, army...)
		return
	}

	balance := army.Sum(scl.CmpGroundScore) / enemies.Sum(scl.CmpGroundScore)
	if alert && balance < 1 {
		worker := bot.GetSCV(B.Locs.MyStart, bot.WorkerRushDefenders, 20)
		if worker != nil {
			army.Add(worker)
		}
	}

	for _, unit := range army {
		if unit.Hits < 11 {
			B.Groups.Add(bot.Miners, unit)
			continue
		}

		if unit.IsCoolToAttack() {
			unit.AttackCustom(scl.DefaultAttackFunc, WorkerMoveFunc, enemies)
		} else if unit.IsCoolToMove() {
			friends := army.InRangeOf(unit, 0)
			friend := friends.Min(scl.CmpHits)
			if friend != nil && friend.Hits < 45 && B.Minerals > 0 {
				unit.CommandTag(ability.Effect_Repair_SCV, friend.Tag)
			}
		}
	}

	if B.WorkerRush && B.Minerals >= 75 {
		workers := B.Groups.Get(bot.Miners).Units.Filter(func(unit *scl.Unit) bool {
			return unit.Hits < 11 && unit.IsGathering() && enemies.CanAttack(unit, 2).Empty()
		})
		if workers.Len() >= 2 {
			workers[0].CommandTag(ability.Effect_Repair_SCV, workers[1].Tag)
			workers[1].CommandTag(ability.Effect_Repair_SCV, workers[0].Tag)
			newGroup := B.Groups.New(workers[0], workers[1])
			B.DoubleHealers = append(B.DoubleHealers, newGroup)
		}
	}
}

func GetHealingPoints() point.Points {
	var healingPoints point.Points
	scvs := B.Units.My[terran.SCV]
	for _, expLoc := range append(B.Locs.MyExps, B.Locs.MyStart) {
		if scvs.CloserThan(scl.ResourceSpreadDistance, expLoc).Len() < 4 {
			continue
		}
		healingPoints = append(healingPoints, expLoc)
	}
	return healingPoints
}

func MechRetreat() {
	us := B.Groups.Get(bot.MechRetreat).Units
	if us.Empty() {
		return
	}

	enemies := B.Enemies.AllReady
	healingPoints := GetHealingPoints()
	if len(healingPoints) == 0 {
		return // todo: something with units if no healing points
	}

	for _, u := range us {
		if u.Health == u.HealthMax && !u.HasBuff(buff.RavenScramblerMissile) {
			bot.OnUnitCreated(u) // Add to corresponding group
			continue
		}

		// Find closest healing point
		healingPoint := healingPoints.ClosestTo(u) - B.Locs.MyStartMinVec*3
		if u.UnitType == terran.Battlecruiser && u.HasAbility(ability.Effect_TacticalJump) {
			u.CommandPos(ability.Effect_TacticalJump, healingPoint)
			continue
		}
		if u.IsCloserThan(4, healingPoint) {
			u.CommandPos(ability.Move, healingPoint) // For battlecruisers
			B.Groups.Add(bot.MechHealing, u)
			continue
		}
		if u.UnitType == terran.Cyclone && u.HasAbility(ability.Effect_LockOn) {
			targets := enemies.Filter(scl.Visible).InRangeOf(u, 2)
			if targets.Exists() && CycloneAttackFunc(u, 0, targets) {
				continue
			}
		}
		if u.Is(terran.HellionTank, terran.SiegeTank) {
			targets := enemies.Filter(scl.Visible).InRangeOf(u, 0)
			if targets.Exists() && u.IsCoolToAttack() && !u.IsAlreadyAttackingTargetInRange() &&
				scl.DefaultAttackFunc(u, 0, targets) {
				continue
			}
		}

		if u.IsCloserThan(8, healingPoint) {
			u.CommandPos(ability.Move, healingPoint)
			continue
		}
		if u.EvadeEffects() {
			continue
		}
		if u.IsFlying {
			if u.UnitType == terran.Medivac && u.HasAbility(ability.Effect_MedivacIgniteAfterburners) {
				u.Command(ability.Effect_MedivacIgniteAfterburners)
			} else {
				pos, _ := u.AirEvade(enemies, 2, healingPoint)
				u.CommandPos(ability.Move, pos)
			}
		} else {
			u.GroundFallback(healingPoint, false)
		}
	}
}

func StaticDefense() {
	targets := B.Enemies.Visible
	buildings := B.Units.My.OfType(terran.Bunker, terran.MissileTurret, terran.AutoTurret) // terran.PlanetaryFortress
	for _, building := range buildings {
		closeTargets := targets.InRangeOf(building, 0)
		if building.UnitType == terran.Bunker && B.Upgrades[ability.Research_Stimpack] {
			// targets.InRangeOf(building, 0).Sum(scl.CmpHits) >= 200
			/*log.Info(building.Abilities)
			log.Info(building.BuffIds)*/
		}

		if closeTargets.Exists() {
			building.Attack(closeTargets)
		}
	}
}

func FlyingBuildings() {
	buildings := B.Units.My.OfType(terran.CommandCenter, terran.CommandCenterFlying,
		terran.OrbitalCommand, terran.OrbitalCommandFlying)
	enemies := B.Enemies.AllReady
	for _, building := range buildings {
		attackers := enemies.CanAttack(building, 0)
		if !building.IsFlying && building.Hits < building.HitsMax*3/4 && attackers.Exists() {
			if building.IsIdle() {
				building.Command(ability.Lift)
			} else {
				building.Command(ability.Cancel_Last)
				building.CommandQueue(ability.Lift)
			}
		} else if building.IsFlying && attackers.Empty() {
			building.CommandPos(ability.Land, B.Locs.MyExps.ClosestTo(building))
		}
	}
}

func ThorEvacs() {
	for _, med := range B.Groups.Get(bot.ThorEvacs).Units {
		if med.HasAbility(ability.Effect_MedivacIgniteAfterburners) {
			med.Command(ability.Effect_MedivacIgniteAfterburners)
			continue
		}
		if len(med.Passengers) == 0 {
			thor := B.Groups.Get(bot.MechRetreat).Units.OfType(terran.Thor, terran.ThorAP).ClosestTo(med)
			if thor == nil {
				B.Groups.Add(bot.Medivacs, med)
				continue
			}
			med.CommandTag(ability.Load_Medivac, thor.Tag)
			continue
		}
		healingPoints := GetHealingPoints()
		if healingPoints.Empty() {
			healingPoints.Add(B.Locs.MyStart - B.Locs.MyStartMinVec*3)
		}
		healingPoint := healingPoints.ClosestTo(med) - B.Locs.MyStartMinVec*3
		med.CommandPos(ability.UnloadAllAt_Medivac, healingPoint)
	}
}

func Micro(b *bot.Bot) {
	B = b // todo: better

	InitTargets()
	WorkerRushDefence()

	for group, logic := range map[scl.GroupID]func(units scl.Units){
		bot.Marines:           MarinesLogic,
		bot.Marauders:         MaraudersLogic,
		bot.Reapers:           ReapersLogic,
		bot.ReapersRetreat:    ReapersRetreatLogic,
		bot.Cyclones:          CyclonesLogic,
		bot.WidowMines:        WidowMinesLogic,
		bot.WidowMinesRetreat: WidowMinesRetreatLogic,
		bot.Hellions:          HellionsLogic,
		bot.Tanks:             TanksLogic,
		bot.Thors:             ThorsLogic,
		bot.Medivacs:          MedivacsLogic,
		bot.Vikings:           VikingsLogic,
		bot.Ravens:            RavensLogic,
		bot.Banshees:          BansheesLogic,
		bot.Battlecruisers:    BattlecruisersLogic,
	} {
		logic(B.Groups.Get(group).Units)
	}

	MechRetreat()
	StaticDefense()
	FlyingBuildings()
	ThorEvacs()
}
