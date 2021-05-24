package macro

import (
	"bitbucket.org/aisee/minilog"
	"github.com/aiseeq/s2l/lib/scl"
	"github.com/aiseeq/s2l/protocol/api"
	"github.com/aiseeq/s2l/protocol/enums/ability"
	"github.com/aiseeq/s2l/protocol/enums/protoss"
	"github.com/aiseeq/s2l/protocol/enums/terran"
	"github.com/aiseeq/s2l/protocol/enums/zerg"
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
	factory := B.Units.My[id].First(func(unit *scl.Unit) bool {
		return unit.IsReady() && unit.IsUnused() && (!needTechlab || unit.HasTechlab()) && !usedFactories[unit.Tag]
	})
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

func OrderUnits() {
	usedFactories := scl.TagsMap{}
	B.MechPriority = false
	if B.EnemyRace != api.Race_Zerg {
		B.MechPriority = true
	}

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
	if cc != nil && B.Units.My[terran.SCV].Len() < scl.MinInt(21*ccs.Len(), 70-refs.Len()) &&
		B.CanBuy(ability.Train_SCV) && !B.WorkerRush {
		OrderTrain(cc, ability.Train_SCV, usedFactories)
	}

	if starport := GetFactory(terran.Starport, true, usedFactories); starport != nil {
		ravens := B.Pending(ability.Train_Raven)
		if B.Units.My[terran.FusionCore].First(scl.Ready) != nil {
			if B.CanBuy(ability.Train_Battlecruiser) {
				OrderTrain(starport, ability.Train_Battlecruiser, usedFactories)
			} else if ravens > 0 && B.Units.AllEnemy[zerg.Ultralisk].Empty() {
				B.DeductResources(ability.Train_Battlecruiser) // Gather money
			}
		}
		if ravens < 2 {
			if B.CanBuy(ability.Train_Raven) {
				OrderTrain(starport, ability.Train_Raven, usedFactories)
			} else if ravens == 0 {
				B.DeductResources(ability.Train_Raven) // Gather money
			}
		}
	}
	if starport := GetFactory(terran.Starport, false, usedFactories); starport != nil {
		medivacs := B.Pending(ability.Train_Medivac)
		infantry := B.Units.My[terran.Marine].Len() + B.Units.My[terran.Marauder].Len()*2
		if (medivacs == 0 || medivacs*8 < infantry) && B.CanBuy(ability.Train_Medivac) {
			OrderTrain(starport, ability.Train_Medivac, usedFactories)
		} else if medivacs == 0 {
			B.DeductResources(ability.Train_Medivac) // Gather money
		}
	}

	if factory := GetFactory(terran.Factory, true, usedFactories); factory != nil {
		cyclones := B.PendingAliases(ability.Train_Cyclone)
		tanks := B.PendingAliases(ability.Train_SiegeTank)

		buyCyclones := B.EnemyProduction.Len(terran.Banshee) > 0 && cyclones == 0
		buyTanks := B.PlayDefensive && tanks == 0
		if !buyCyclones && !buyTanks {
			cyclonesScore := B.EnemyProduction.Score(protoss.WarpPrism, protoss.Phoenix, protoss.VoidRay,
				protoss.Oracle, protoss.Tempest, protoss.Carrier, terran.VikingFighter, terran.Medivac,
				terran.Liberator, terran.Raven, terran.Banshee, terran.Battlecruiser, zerg.Queen, zerg.Mutalisk,
				zerg.Corruptor, zerg.Viper, zerg.Ultralisk, zerg.BroodLord) + 1
			tanksScore := B.EnemyProduction.Score(protoss.Stalker, protoss.Colossus, protoss.PhotonCannon,
				terran.Marine, terran.Reaper, terran.Marauder, terran.Bunker, terran.PlanetaryFortress,
				zerg.Roach, zerg.Ravager, zerg.Hydralisk, zerg.LurkerMP, zerg.SpineCrawler) + 1
			buyCyclones = cyclonesScore/float64(cyclones+1) >= tanksScore/float64(tanks+1)
			buyTanks = !buyCyclones
		}

		if buyCyclones {
			if B.CanBuy(ability.Train_Cyclone) {
				OrderTrain(factory, ability.Train_Cyclone, usedFactories)
			} else if cyclones == 0 || B.MechPriority {
				B.DeductResources(ability.Train_Cyclone) // Gather money
			}
		} else if buyTanks {
			if B.CanBuy(ability.Train_SiegeTank) {
				OrderTrain(factory, ability.Train_SiegeTank, usedFactories)
			} else if tanks == 0 || B.MechPriority {
				B.DeductResources(ability.Train_SiegeTank) // Gather money
			}
		}
	}
	if factory := GetFactory(terran.Factory, false, usedFactories); factory != nil {
		mines := B.PendingAliases(ability.Train_WidowMine)
		hellions := B.PendingAliases(ability.Train_Hellion)

		minesScore := B.EnemyProduction.Score(protoss.Stalker, protoss.Archon, protoss.Phoenix, protoss.VoidRay,
			protoss.Oracle, protoss.Tempest, protoss.Carrier, terran.Cyclone, terran.SiegeTank, terran.Thor,
			terran.VikingFighter, terran.Medivac, terran.Liberator, terran.Raven, terran.Banshee,
			terran.Battlecruiser, zerg.Hydralisk, zerg.Queen, zerg.Roach, zerg.Ravager, zerg.Mutalisk, zerg.Corruptor,
			zerg.Viper, zerg.Ultralisk, zerg.BroodLord) + 1
		hellionsScore := B.EnemyProduction.Score(zerg.Zergling, zerg.Baneling, zerg.SwarmHostMP) + 1
		buyMines := minesScore/float64(mines+1) >= hellionsScore/float64(hellions+1)

		if buyMines && (mines == 0 || hellions != 0) {
			if B.CanBuy(ability.Train_WidowMine) {
				OrderTrain(factory, ability.Train_WidowMine, usedFactories)
			} else if mines == 0 || B.MechPriority {
				B.DeductResources(ability.Train_WidowMine) // Gather money
			}
		} else {
			if B.CanBuy(ability.Train_Hellion) {
				OrderTrain(factory, ability.Train_Hellion, usedFactories)
			} else if hellions == 0 || B.MechPriority {
				B.DeductResources(ability.Train_Hellion) // Gather money
			}
		}
	}

	if rax := GetFactory(terran.Barracks, true, usedFactories); rax != nil {
		marines := B.PendingAliases(ability.Train_Marine)
		marauders := B.PendingAliases(ability.Train_Marauder)
		marinesScore := B.EnemyProduction.Score(protoss.Immortal, protoss.WarpPrism, protoss.Phoenix, protoss.VoidRay,
			protoss.Oracle, protoss.Tempest, protoss.Carrier, terran.VikingFighter, terran.Medivac, terran.Liberator,
			terran.Raven, terran.Banshee, terran.Battlecruiser, zerg.Mutalisk, zerg.Corruptor, zerg.Viper,
			zerg.BroodLord) + 1 //  zerg.Zergling,
		maraudersScore := B.EnemyProduction.Score(protoss.Zealot, protoss.Stalker, protoss.Adept, terran.Reaper,
			terran.Hellion, terran.WidowMine, terran.Cyclone, terran.Thor, zerg.Baneling, zerg.Roach, zerg.Ravager,
			zerg.Ultralisk) + 1
		buyMarauders := marinesScore/float64(marines+1) < maraudersScore/float64(marauders+1)

		if buyMarauders {
			if B.CanBuy(ability.Train_Marauder) {
				OrderTrain(rax, ability.Train_Marauder, usedFactories)
			} else {
				B.DeductResources(ability.Train_Marauder) // Gather money
			}
		}
	}
	if rax := GetFactory(terran.Barracks, false, usedFactories); rax != nil {
		// Until 4:00
		// B.Loop < 5376 && (B.Pending(ability.Train_Reaper) < 2 || B.EnemyRace == api.Race_Zerg) &&
		// before 2:40 or if they are not dying until 4:00
		if !B.LingRush && (B.Loop < 3584 || (B.Loop < 5376 && B.Pending(ability.Train_Reaper) > B.Loop/1344)) &&
			B.CanBuy(ability.Train_Reaper) {
			OrderTrain(rax, ability.Train_Reaper, usedFactories)
		} else if B.CanBuy(ability.Train_Marine) {
			OrderTrain(rax, ability.Train_Marine, usedFactories)
		}
	}
}
