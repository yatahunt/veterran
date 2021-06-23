package macro

import (
	"bitbucket.org/aisee/minilog"
	"github.com/aiseeq/s2l/lib/scl"
	"github.com/aiseeq/s2l/protocol/api"
	"github.com/aiseeq/s2l/protocol/enums/ability"
	"github.com/aiseeq/s2l/protocol/enums/protoss"
	"github.com/aiseeq/s2l/protocol/enums/terran"
	"github.com/aiseeq/s2l/protocol/enums/zerg"
	"sort"
)

/*type TrainNode struct {
	Name    string
	Ability api.AbilityID
	TechLab bool
	Premise Booler
	WaitRes Booler
	Limit   Inter
	Active  Inter
	Unlocks TrainNodes
}
type TrainNodes []TrainNode

var RootTrainOrder = TrainNodes{
	{
		Name:    "Worker Rush Marine",
		Ability: ability.Train_Marine,
		Premise: func() bool { return B.WorkerRush },
	},
}

func ProcessTrainOrder(trainOrder TrainNodes) {
	UsedFactories = map[api.UnitTag]bool{}
	for _, node := range trainOrder {
		inLimits := B.Pending(node.Ability) < node.Limit() && B.Orders[node.Ability] < node.Active()
		canBuy := B.CanBuy(node.Ability)
		waitRes := node.WaitRes != nil && node.WaitRes()
		if (node.Premise == nil || node.Premise()) && inLimits && (canBuy || waitRes) {
			techReq := B.U.Types[B.U.AbilityUnit[node.Ability]].TechRequirement
			factories := B.Units.My.OfType(B.U.UnitAliases.For(techReq)...).Filter(scl.Ready, scl.Unused)
			if node.TechLab {
				factories = factories.Filter(scl.HasTechlab)
			}
			factory := factories.First()
			if factory == nil {
				continue
			}

			if !canBuy && waitRes {
				// reserve money for building
				B.DeductResources(node.Ability)
				continue
			}

			OrderTrain(factory, node.Ability)
			UsedFactories[factory.Tag] = true
		}
		if node.Unlocks != nil && B.Units.My[B.U.AbilityUnit[node.Ability]].Exists() {
			ProcessTrainOrder(node.Unlocks)
		}
	}
}*/

func OrderTrain(factory *scl.Unit, aid api.AbilityID, usedFactories scl.TagsMap) {
	factory.Command(aid)
	B.DeductResources(aid)
	if usedFactories != nil {
		usedFactories[factory.Tag] = true
	}
	log.Debugf("%d: Training %v", B.Loop, B.U.Types[B.U.AbilityUnit[aid]].Name)
}

func GetFactory(id api.UnitTypeID, needTechlab bool, usedFactories scl.TagsMap) *scl.Unit {
	var factory *scl.Unit
	if needTechlab == false {
		// Try to find factory with reactor first
		factory = B.Units.My[id].First(func(unit *scl.Unit) bool {
			return unit.IsReady() && unit.IsUnused() && unit.HasReactor() && !usedFactories[unit.Tag]
		})
	}
	if factory == nil {
		// We need a tech lab or there is no factories with a reactor
		factory = B.Units.My[id].First(func(unit *scl.Unit) bool {
			return unit.IsReady() && unit.IsUnused() && (!needTechlab || unit.HasTechlab()) && !usedFactories[unit.Tag]
		})
	}
	if factory == nil {
		return nil
	}
	if factory.HasReactor() && B.U.UnitsOrders[factory.Tag].Loop+B.FramesPerOrder <= B.Loop {
		// I need to pass this param because else duplicate order will be ignored
		// But I need to be sure that there was no previous order recently
		factory.SpamCmds = true
	}
	return factory
}

func Order(unit api.AbilityID, factoryType api.UnitTypeID, needTechlab, gatherMoney bool, usedFactories scl.TagsMap) {
	techReq := B.U.Types[B.U.AbilityUnit[unit]].TechRequirement
	if techReq != 0 && B.Units.My.OfType(B.U.UnitAliases.For(techReq)...).First(scl.Ready) == nil {
		// log.Debugf("Tech requirement didn't met for %v", B.U.Types[B.U.AbilityUnit[unit]].Name)
		return // Not available because of tech reqs, like: supply is needed for barracks
	}
	if factory := GetFactory(factoryType, needTechlab, usedFactories); factory != nil {
		if B.CanBuy(unit) {
			OrderTrain(factory, unit, usedFactories)
			log.Debugf("Ordered %v", B.U.Types[B.U.AbilityUnit[unit]].Name)
		} else if gatherMoney {
			B.DeductResources(unit) // Gather money
			// log.Debugf("Waiting for %v", B.U.Types[B.U.AbilityUnit[unit]].Name)
		}
	}
}

func NormalizeScore(score map[api.AbilityID]int, a api.AbilityID, qty, defQty int) {
	ut := B.U.AbilityUnit[a]
	price := int(B.U.Types[ut].MineralCost + B.U.Types[ut].VespeneCost)
	score[a] = 1000 * (score[a] + defQty*price/100) / ((qty + 1) * price)
}

func OrderUnits() {
	usedFactories := scl.TagsMap{}

	if B.WorkerRush && B.CanBuy(ability.Train_Marine) {
		if rax := GetFactory(terran.Barracks, false, usedFactories); rax != nil {
			OrderTrain(rax, ability.Train_Marine, usedFactories)
		}
	}

	ccs := B.Units.My.OfType(terran.CommandCenter, terran.OrbitalCommand, terran.PlanetaryFortress)
	cc := ccs.First(scl.Ready, scl.Idle)
	refs := B.Units.My[terran.Refinery].Filter(func(unit *scl.Unit) bool {
		return unit.IsReady() && unit.VespeneContents > 0
	})
	// Build SCVs todo: check if all minerals & gas are saturated
	if cc != nil && B.Units.My[terran.SCV].Len() < scl.MinInt(21*ccs.Len(), 70-refs.Len()) &&
		B.CanBuy(ability.Train_SCV) && !B.WorkerRush {
		OrderTrain(cc, ability.Train_SCV, usedFactories)
	}

	// Tank strong: Photon Cannon, Stalker, Bunker, Planetary Fortress, Marine, Cyclone, Spine, Roach, Hydralisk,
	// Lurker
	// Tank weak: Immortal, Carrier, Banshee, Mutalisk, Ravager, Viper
	// Cyclone strong: Adept, Marauder, Roach
	// Cyclone weak: Immortal, Tank, Zergling
	// Thor strong: Archon, Disruptor, Stalker, Planetary Fortress, Marine, Battlecruiser, Mutalisk
	// Thor weak: Immortal, Marauder, Zergling
	// Mine strong: Zealot, Adept, Mutalisk, Zergling, Baneling
	// Mine weak: Stalker, Marauder, Ravager
	// Hellion strong: Zealot, Marine, Zergling, Swarm Host
	// Hellion weak: Stalker, Marauder, Roach, Queen
	// Viking strong: Colossus, Void Ray, Oracle, Tempest, Carrier, Mothership, Banshee, Liberator, Battlecruiser,
	// Brood Lord, Overseer, Corruptor, Viper
	// Viking weak: Stalker, Marine, Hydralisk, Mutalisk
	// Banshee strong: Colossus, Tank, Ultralisk, Swarm Host
	// Banshee weak: Observer, Phoenix, Viking, Raven, Hydralisk, Overseer
	// Battlecruiser strong: Phoenix, Carrier, Mutalisk
	// Battlecruiser weak: Void Ray, Viking, Thor, Corruptor
	// Raven strong: Dark Templar, Banshee, Roach
	// Raven weak: Phoenix, Ghost, Corruptor
	// Liberator strong: Phoenix, Mutalisk
	// Liberator weak: Tempest, Void Ray, Viking, Corruptor
	// Marine strong: Immortal, Viking, Marauder, Hydralisk, Queen
	// Marine weak: Adept, High Templar, Archon, Colossus, Tank, Thor, Hellion, Baneling, Lurker, Infestor, Ultralisk,
	// Brood Lord
	// Marauder strong: Stalker, Adept, Thor, Reaper, Ghost, Hellion, Mine, Roach, Ravager, Baneling
	// Marauder weak: Zealot, Disruptor, Marine, Cyclone, Zergling
	// Reaper strong: Sentry
	// Reaper weak: Stalker, Marauder, Roach
	// Ghost strong: High Templar, Raven, Infestor, Ultralisk
	// Ghost weak: Stalker, Marauder, Zergling

	score := map[api.AbilityID]int{}
	tanks := B.PendingAliases(ability.Train_SiegeTank)
	score[ability.Train_SiegeTank] = B.EnemyProduction.Score(protoss.PhotonCannon, protoss.Stalker, terran.Bunker,
		terran.PlanetaryFortress, terran.Marine, terran.Cyclone, zerg.SpineCrawler, zerg.Roach, zerg.Hydralisk,
		zerg.LurkerMP) -
		B.EnemyProduction.Score(protoss.Immortal, protoss.Carrier, terran.Banshee, zerg.Mutalisk, zerg.Ravager,
			zerg.Viper)
	NormalizeScore(score, ability.Train_SiegeTank, tanks, 1)

	cyclones := B.PendingAliases(ability.Train_Cyclone)
	score[ability.Train_Cyclone] = B.EnemyProduction.Score(protoss.Adept, terran.Marauder, zerg.Roach, // from wiki
		protoss.WarpPrism, protoss.Phoenix, protoss.VoidRay, protoss.Oracle, terran.Reaper, // mine
		terran.VikingFighter, terran.Liberator, terran.Banshee, terran.Battlecruiser, zerg.Ravager, zerg.Mutalisk,
		zerg.Corruptor, zerg.Viper, zerg.Ultralisk) -
		B.EnemyProduction.Score(protoss.Immortal, terran.SiegeTank, zerg.Zergling)
	NormalizeScore(score, ability.Train_Cyclone, cyclones, 1)

	thors := B.PendingAliases(ability.Train_Thor)
	score[ability.Train_Thor] = B.EnemyProduction.Score(protoss.Archon, protoss.Disruptor, protoss.Stalker,
		terran.PlanetaryFortress, terran.Marine, terran.Battlecruiser, zerg.Mutalisk) -
		B.EnemyProduction.Score(protoss.Immortal, terran.Marauder, zerg.Zergling)
	NormalizeScore(score, ability.Train_Thor, thors, 1)

	mines := B.PendingAliases(ability.Train_WidowMine)
	score[ability.Train_WidowMine] = B.EnemyProduction.Score(protoss.Zealot, protoss.Adept, zerg.Zergling,
		zerg.Baneling) -
		B.EnemyProduction.Score(protoss.Stalker, terran.Marauder, zerg.Ravager)
	NormalizeScore(score, ability.Train_WidowMine, mines, 1)

	hellions := B.PendingAliases(ability.Train_Hellion)
	score[ability.Train_Hellion] = B.EnemyProduction.Score(protoss.Zealot, terran.Marine, zerg.Zergling,
		zerg.SwarmHostMP) -
		B.EnemyProduction.Score(protoss.Stalker, terran.Marauder, zerg.Roach, zerg.Queen)
	NormalizeScore(score, ability.Train_Hellion, hellions, 1)

	vikings := B.PendingAliases(ability.Train_VikingFighter)
	score[ability.Train_VikingFighter] = B.EnemyProduction.Score(protoss.Colossus, protoss.VoidRay, protoss.Oracle,
		protoss.Tempest, protoss.Carrier, protoss.Mothership, terran.Banshee, terran.Liberator, terran.Battlecruiser,
		zerg.BroodLord, zerg.Overseer, zerg.Corruptor, zerg.Viper, // from wiki
		protoss.WarpPrism, protoss.Phoenix, terran.VikingFighter) - // mine
		B.EnemyProduction.Score(protoss.Stalker, terran.Marine, zerg.Hydralisk, zerg.Mutalisk)
	NormalizeScore(score, ability.Train_VikingFighter, vikings, 1)

	banshees := B.PendingAliases(ability.Train_Banshee)
	score[ability.Train_Banshee] = B.EnemyProduction.Score(protoss.Colossus, terran.SiegeTank, zerg.Ultralisk,
		zerg.SwarmHostMP) -
		B.EnemyProduction.Score(protoss.Observer, protoss.Phoenix, terran.VikingFighter, terran.Raven, zerg.Hydralisk,
			zerg.Overseer)
	NormalizeScore(score, ability.Train_Banshee, banshees, 1)

	cruisers := B.PendingAliases(ability.Train_Battlecruiser)
	score[ability.Train_Battlecruiser] = B.EnemyProduction.Score(protoss.Phoenix, protoss.Carrier, zerg.Mutalisk) -
		B.EnemyProduction.Score(protoss.VoidRay, terran.VikingFighter, terran.Thor, zerg.Corruptor) + 4000 // BCS++
	NormalizeScore(score, ability.Train_Battlecruiser, cruisers, 1)

	medivacs := B.PendingAliases(ability.Train_Medivac)
	score[ability.Train_Medivac] = 10 * (B.Units.My[terran.Marine].Len()*50 + B.Units.My[terran.Marauder].Len()*125)
	score[ability.Train_Medivac] /= medivacs + 1

	ravens := B.PendingAliases(ability.Train_Raven)
	score[ability.Train_Raven] = 1000

	marines := B.PendingAliases(ability.Train_Marine)
	score[ability.Train_Marine] = B.EnemyProduction.Score(protoss.Immortal, terran.VikingFighter, terran.Marauder,
		zerg.Hydralisk, zerg.Queen, // from wiki
		protoss.VoidRay, protoss.Carrier, zerg.Zergling) - // mine
		B.EnemyProduction.Score(protoss.Adept, protoss.HighTemplar, protoss.Archon, protoss.Colossus, terran.SiegeTank,
			terran.Thor, terran.Hellion, zerg.Baneling, zerg.LurkerMP, zerg.Infestor, zerg.Ultralisk, zerg.BroodLord)
	NormalizeScore(score, ability.Train_Marine, marines, 2)

	marauders := B.PendingAliases(ability.Train_Marauder)
	score[ability.Train_Marauder] = B.EnemyProduction.Score(protoss.Stalker, protoss.Adept, terran.Thor,
		terran.Hellion, terran.WidowMine, zerg.Roach, zerg.Ravager, zerg.Baneling, // from wiki
		terran.Reaper) - // mine
		B.EnemyProduction.Score(protoss.Zealot, protoss.Disruptor, terran.Marine, terran.Cyclone, zerg.Zergling)
	NormalizeScore(score, ability.Train_Marauder, marauders, 1)

	reapers := B.PendingAliases(ability.Train_Reaper)
	score[ability.Train_Reaper] = 0

	// Priority train score
	if B.BruteForce && tanks == 0 && B.Loop < scl.TimeToLoop(2, 45) {
		score[ability.Train_SiegeTank] += 10000
	}
	if tanks == 0 && score[ability.Train_SiegeTank] > 0 && cyclones > 0 {
		score[ability.Train_Cyclone] = -1
	}
	if cyclones == 0 && score[ability.Train_Cyclone] > 0 && tanks > 0 {
		score[ability.Train_SiegeTank] = -1
	}
	if hellions >= 4 {
		score[ability.Train_Hellion] = -1
	}
	if mines >= 8 {
		score[ability.Train_WidowMine] = -1
	}
	if vikings >= 8 {
		score[ability.Train_VikingFighter] = -1
	}
	if (B.Units.AllEnemy[terran.Banshee].Exists() ||
		B.Units.AllEnemy.OfType(B.U.UnitAliases.For(terran.Starport)...).Exists()) && vikings == 0 {
		score[ability.Train_VikingFighter] += 10000
	}
	if medivacs >= 4 || medivacs > (marines+marauders*2)/8 {
		score[ability.Train_Medivac] = -1
	}
	if B.BruteForce && medivacs == 0 && B.Loop < scl.TimeToLoop(3, 15) {
		score[ability.Train_Medivac] += 10000
	}
	if ravens == 0 {
		score[ability.Train_Raven] += 10000
	}
	if ravens >= 2 {
		score[ability.Train_Raven] = -1
	}
	if B.Loop < scl.TimeToLoop(2, 40) && reapers < 4 && !B.ProxyMarines && !B.BruteForce {
		score[ability.Train_Reaper] += 10000
	}
	/*if B.Loop < scl.TimeToLoop(3, 0) && (B.ProxyMarines || B.BruteForce) {
		score[ability.Train_Marine] += 5000
	}*/
	if B.Loop < scl.TimeToLoop(1, 30) && !B.ProxyMarines {
		// Don't build marine first if we almost have gas for the reaper
		score[ability.Train_Marine] = -1
	}

	abils := []api.AbilityID{ability.Train_SiegeTank, ability.Train_Cyclone, ability.Train_Thor,
		ability.Train_WidowMine, ability.Train_Hellion, ability.Train_VikingFighter, ability.Train_Banshee,
		ability.Train_Battlecruiser, ability.Train_Medivac, ability.Train_Raven, ability.Train_Marine,
		ability.Train_Marauder, ability.Train_Reaper}
	sort.Slice(abils, func(i, j int) bool {
		return score[abils[i]] > score[abils[j]]
	})

	for _, abil := range abils {
		if score[abil] <= 0 || B.Minerals < 50 && B.Vespene < 0 {
			break
		}
		// fmt.Printf("%s: %d, ", B.U.Types[B.U.AbilityUnit[abil]].Name, score[abil])
		switch abil {
		case ability.Train_SiegeTank:
			Order(abil, terran.Factory, true, true, usedFactories)
		case ability.Train_Cyclone:
			Order(abil, terran.Factory, true, true, usedFactories)
		case ability.Train_Thor:
			if B.Units.My[terran.Armory].First(scl.Ready) != nil {
				Order(abil, terran.Factory, true, true, usedFactories)
			}
		case ability.Train_WidowMine:
			Order(abil, terran.Factory, false, true, usedFactories)
		case ability.Train_Hellion:
			Order(abil, terran.Factory, false, true, usedFactories)
		case ability.Train_VikingFighter:
			Order(abil, terran.Starport, false, true, usedFactories)
		case ability.Train_Banshee:
			Order(abil, terran.Starport, true, true, usedFactories)
		case ability.Train_Battlecruiser:
			if B.Units.My[terran.FusionCore].First(scl.Ready) != nil {
				Order(abil, terran.Starport, true, true, usedFactories)
			}
		case ability.Train_Medivac:
			Order(abil, terran.Starport, false, true, usedFactories)
		case ability.Train_Raven:
			Order(abil, terran.Starport, true, true, usedFactories)
		case ability.Train_Marine:
			Order(abil, terran.Barracks, false, true, usedFactories)
		case ability.Train_Marauder:
			Order(abil, terran.Barracks, true, true, usedFactories)
		case ability.Train_Reaper:
			Order(abil, terran.Barracks, false, true, usedFactories)
		default:
			log.Error("Unknown ability %v", abil)
		}
	}
	// fmt.Println()
}
