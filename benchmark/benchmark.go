package main

import (
	log "bitbucket.org/aisee/minilog"
	"bitbucket.org/aisee/veterran/bot"
	"bitbucket.org/aisee/veterran/macro"
	"bitbucket.org/aisee/veterran/micro"
	"bitbucket.org/aisee/veterran/roles"
	"fmt"
	"github.com/aiseeq/s2l/lib/point"
	"github.com/aiseeq/s2l/lib/scl"
	"github.com/aiseeq/s2l/protocol/api"
	"github.com/aiseeq/s2l/protocol/client"
	"github.com/aiseeq/s2l/protocol/enums/protoss"
	"github.com/aiseeq/s2l/protocol/enums/terran"
	"github.com/aiseeq/s2l/protocol/enums/zerg"
	"github.com/google/gxui/math"
	"math/rand"
	"time"
)

func RunAgent(c *client.Client, testVal float64) int {
	B := &bot.Bot{
		Bot:           scl.New(c, bot.OnUnitCreated),
		PlayDefensive: true,
		BuildPos:      map[scl.BuildingSize]point.Points{},
		CycloneLocks:  map[api.UnitTag]api.UnitTag{},
	}
	bot.B = B
	B.FramesPerOrder = 3
	B.LastLoop = -math.MaxInt
	B.MaxGroup = bot.MaxGroup
	B.Logic = func() {
		bot.DefensivePlayCheck()
		roles.Roles(B)
		macro.Macro(B)
		micro.Micro(B)
	}
	stop := make(chan struct{})
	B.Init(stop) // Called later because in Init() we need to use *B in callback
	B.TestVal = testVal

	for B.Client.Status == api.Status_in_game {
		bot.Step()
		if _, err := c.Step(api.RequestStep{Count: uint32(B.FramesPerOrder)}); err != nil {
			if err.Error() == "Not in a game" {
				break
			}
			log.Fatal(err)
		}

		B.UpdateObservation()
	}

	stop <- struct{}{}
	if len(B.Result) == 0 {
		B.UpdateObservation()
		if len(B.Result) == 0 {
			log.Error("Failed to get game result")
			return 0
		}
	}
	myId := B.Obs.PlayerCommon.PlayerId
	seconds := int(float64(B.Loop) / scl.FPS)
	log.Infof("Test: %v, Versus: %v, Result: %v, Time: %ds",
		B.TestVal, B.EnemyRace, B.Result[myId-1].Result, seconds)
	if B.Result[myId-1].Result != api.Result_Victory {
		// seconds = 10000
		return 0
	}
	return seconds
}

func testWholeGame() {
	log.SetConsoleLevel(log.L_info) // L_info L_debug
	rand.Seed(time.Now().UnixNano())
	stats := map[api.Race]map[float64][]int{}
	for _, race := range []api.Race{api.Race_Terran, api.Race_Zerg, api.Race_Protoss} {
		stats[race] = map[float64][]int{}
	}

	minVal := 1.0
	maxVal := 3.0
	valStep := 0.25
	testVal := minVal

	var cfg *client.GameConfig
	myBot := client.NewParticipant(api.Race_Terran, "VeTerran")
	cpu := client.NewComputer(api.Race_Random, api.Difficulty_CheatInsane, api.AIBuild_RandomBuild)
	for round := 1; ; round++ {
		/*for _, mapName := range client.MapsProBotsSeason2*/ {
			mapName := client.MapsProBotsSeason2[3]
			for _, race := range []api.Race{api.Race_Terran, api.Race_Zerg, api.Race_Protoss} {
				cpu.Race = race

				var seconds int
				for seconds == 0 {
					if cfg == nil {
						client.SetMap(mapName + ".SC2Map")
						cfg = client.LaunchAndJoin(myBot, cpu)
					} else {
						cfg.StartGame(mapName + ".SC2Map")
					}
					seconds = RunAgent(cfg.Client, testVal)
				}
				stats[race][testVal] = append(stats[race][testVal], seconds)

				fmt.Printf("Round %2d", round)
				for x := minVal; x < maxVal+valStep; x += valStep {
					fmt.Printf("    %.2f", x)
				}
				fmt.Println()
				for _, r := range []api.Race{api.Race_Terran, api.Race_Zerg, api.Race_Protoss} {
					fmt.Printf("%8s", r)
					for x := minVal; x < maxVal+valStep; x += valStep {
						times := stats[r][x]
						avg := 0
						if len(times) > 0 {
							for _, secs := range times {
								avg += secs
							}
							avg /= len(times)
						}
						fmt.Printf(" %7d", avg)
					}
					fmt.Println()
				}
				fmt.Print(" Average")
				for x := minVal; x < maxVal+valStep; x += valStep {
					avg := 0
					results := 0
					for _, r := range []api.Race{api.Race_Terran, api.Race_Zerg, api.Race_Protoss} {
						times := stats[r][x]
						if len(times) > 0 {
							for _, secs := range times {
								avg += secs
							}
							results += len(times)
						}
					}
					if results > 0 {
						avg /= results
					}
					fmt.Printf(" %7d", avg)
				}
				fmt.Println()
			}
			testVal += valStep
			if testVal > maxVal { // + valStep { // floats...
				testVal = minVal
			}
		}
	}
}

/*
Strength multiplier for AssessStrength
2021-06-06 10:03:55 [Info] benchmark.go:62 - Test: 1.7000000000000006, Versus: Terran, Result: Victory, Time: 875s
Round 14    0.80    0.90    1.00    1.10    1.20    1.30    1.40    1.50    1.60    1.70
  Terran     894    1074    1030    1007     929     955     819     847     873     943
    Zerg     756     840     850     683     707     882     877     816     606     913
 Protoss     873     808     915     969     830     875     846     894     962     876
 Average     841     907     932     886     822     904     848     852     813     912

Radius for AssessStrength
2021-06-07 10:05:41 [Info] benchmark.go:62 - Test: 2, Versus: Protoss, Result: Victory, Time: 1041s
Round 24    1.00    2.00    3.00    4.00    5.00    6.00    7.00    8.00    9.00    10.00    11.00    12.00    13.00    14.00
  Terran    1022     967    1123     934     976    1020     903     857    1058     893    1048     674     972     919
    Zerg     889     683     683     854     594     774     899     595     645     829     724     622     804     759
 Protoss     880     904     827     848    1084     912     888     851     803     841     920     897     945     855
 Average     930     851     878     879     885     902     897     768     835     854     897     731     907     844

Radius for AssessStrength using only LightshadeAIE
2021-06-07 12:34:10 [Info] benchmark.go:62 - Test: 14, Versus: Protoss, Result: Victory, Time: 968s
Round 14    1.00    2.00    3.00    4.00    5.00    6.00    7.00    8.00    9.00    10.00    11.00    12.00    13.00    14.00
  Terran     980     767     810     864     923     767     696     775    1087    1030     820     891     730    1048
    Zerg    1362     557     798     926     578     817    1087     518     621    1234     523     530     803     739
 Protoss    1184     781     870     820     854    1120    1102    1521     803     815    1131     763     797     968
 Average    1175     701     826     870     785     901     961     938     837    1026     824     728     776     918

Radius for AssessStrength using only LightshadeAIE
2021-06-07 20:12:09 [Info] benchmark.go:62 - Test: 13, Versus: Terran, Result: Victory, Time: 938s
Round 41    1.00    2.00    3.00    4.00    5.00    6.00    7.00    8.00    9.00    10.00    11.00    12.00    13.00    14.00
  Terran     725     995    1155     991     833     962     898    1197    1487     802    1009     908    1009     929
    Zerg     562     623     836     956    1433     833     663     550     785     677     675     642     925    1084
 Protoss     840     933    1049    1043     807     781    1144     910    1028     813     881     834     809     803
 Average     709     850    1013     996    1024     859     901     886    1100     764     855     794     928     939

Похоже, симуляция такого типа если и может принести результаты, то разве что при очень длительном тестировании
потому что влияние различных случайных факторов черезвычайно велико. Надо придумать какой-то боевой сценарий и
опираться не на секунды, а на количество и здоровье оставшихся юнитов
*/

func VsInit(b *bot.Bot) (api.PlayerID, api.PlayerID) {
	myId := b.Obs.PlayerCommon.PlayerId
	enemyId := 3 - myId
	b.PlayDefensive = false

	for _, miner := range b.Units.My[terran.SCV] {
		b.DebugKillUnits(miner.Tag)
	}
	cc := b.Units.My[terran.CommandCenter].First()
	b.DebugKillUnits(cc.Tag)
	b.DebugAddUnits(terran.SupplyDepot, myId, b.Locs.MapCenter, 1)
	b.DebugAddUnits(terran.WidowMineBurrowed, myId, b.Locs.MapCenter, 20)

	b.DebugAddUnits(terran.Marine, myId, b.Locs.MyStart.Towards(b.Locs.MapCenter, -2), 8)
	b.DebugAddUnits(terran.Reaper, myId, b.Locs.MyStart.Towards(b.Locs.MapCenter, -2), 2)
	b.DebugAddUnits(terran.Marauder, myId, b.Locs.MyStart.Towards(b.Locs.MapCenter, -2), 4)
	b.DebugAddUnits(terran.HellionTank, myId, b.Locs.MyStart.Towards(b.Locs.MapCenter, -2), 4)
	b.DebugAddUnits(terran.WidowMine, myId, b.Locs.MyStart.Towards(b.Locs.MapCenter, -2), 2)
	b.DebugAddUnits(terran.Cyclone, myId, b.Locs.MyStart.Towards(b.Locs.MapCenter, -2), 2)
	b.DebugAddUnits(terran.SiegeTank, myId, b.Locs.MyStart.Towards(b.Locs.MapCenter, -2), 2)
	b.DebugAddUnits(terran.Thor, myId, b.Locs.MyStart.Towards(b.Locs.MapCenter, -2), 1)
	b.DebugAddUnits(terran.VikingFighter, myId, b.Locs.MyStart.Towards(b.Locs.MapCenter, -2), 4)
	b.DebugAddUnits(terran.Medivac, myId, b.Locs.MyStart.Towards(b.Locs.MapCenter, -2), 2)
	b.DebugAddUnits(terran.Raven, myId, b.Locs.MyStart.Towards(b.Locs.MapCenter, -2), 1)
	b.DebugAddUnits(terran.Banshee, myId, b.Locs.MyStart.Towards(b.Locs.MapCenter, -2), 2)
	b.DebugAddUnits(terran.Battlecruiser, myId, b.Locs.MyStart.Towards(b.Locs.MapCenter, -2), 1)

	b.Actions.MoveCamera(b.Locs.MyStart.Towards(b.Locs.MapCenter, 6))

	return myId, enemyId
}

func VsTerran(b *bot.Bot) {
	_, enemyId := VsInit(b)

	b.DebugAddUnits(terran.Marine, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 8), 1)
	b.DebugAddUnits(terran.Marine, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 7)
	b.DebugAddUnits(terran.Reaper, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 2)
	b.DebugAddUnits(terran.Marauder, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 4)
	b.DebugAddUnits(terran.HellionTank, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 4)
	b.DebugAddUnits(terran.WidowMine, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 2)
	b.DebugAddUnits(terran.Cyclone, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 2)
	b.DebugAddUnits(terran.SiegeTank, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 2)
	b.DebugAddUnits(terran.Thor, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 1)
	b.DebugAddUnits(terran.VikingFighter, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 4)
	b.DebugAddUnits(terran.Medivac, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 2)
	b.DebugAddUnits(terran.Raven, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 1)
	b.DebugAddUnits(terran.Banshee, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 2)
	b.DebugAddUnits(terran.Battlecruiser, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 1)

	b.DebugSend()
}

func VsZerg(b *bot.Bot) {
	_, enemyId := VsInit(b)

	b.DebugAddUnits(zerg.Queen, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 8), 1)
	b.DebugAddUnits(zerg.Zergling, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 16)
	b.DebugAddUnits(zerg.Baneling, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 4)
	b.DebugAddUnits(zerg.Queen, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 2)
	b.DebugAddUnits(zerg.Roach, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 4)
	b.DebugAddUnits(zerg.Ravager, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 2)
	b.DebugAddUnits(zerg.Hydralisk, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 4)
	b.DebugAddUnits(zerg.LurkerMP, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 2)
	b.DebugAddUnits(zerg.Mutalisk, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 4)
	b.DebugAddUnits(zerg.Corruptor, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 2)
	b.DebugAddUnits(zerg.SwarmHostMP, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 2)
	b.DebugAddUnits(zerg.Infestor, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 1)
	b.DebugAddUnits(zerg.Viper, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 1)
	b.DebugAddUnits(zerg.Ultralisk, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 1)
	b.DebugAddUnits(zerg.BroodLord, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 2)

	b.DebugSend()
}

func VsProtoss(b *bot.Bot) {
	_, enemyId := VsInit(b)

	b.DebugAddUnits(protoss.Zealot, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 8), 1)
	b.DebugAddUnits(protoss.Zealot, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 3)
	b.DebugAddUnits(protoss.Stalker, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 4)
	b.DebugAddUnits(protoss.Sentry, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 2)
	b.DebugAddUnits(protoss.Adept, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 2)
	b.DebugAddUnits(protoss.HighTemplar, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 2)
	b.DebugAddUnits(protoss.DarkTemplar, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 2)
	b.DebugAddUnits(protoss.Archon, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 1)
	b.DebugAddUnits(protoss.Observer, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 2)
	b.DebugAddUnits(protoss.WarpPrism, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 1)
	b.DebugAddUnits(protoss.Immortal, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 1)
	b.DebugAddUnits(protoss.Colossus, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 1)
	// b.DebugAddUnits(protoss.Disruptor, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 1)
	b.DebugAddUnits(protoss.Phoenix, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 4)
	b.DebugAddUnits(protoss.VoidRay, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 1)
	b.DebugAddUnits(protoss.Oracle, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 1)
	b.DebugAddUnits(protoss.Tempest, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 1)
	b.DebugAddUnits(protoss.Carrier, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 1)

	b.DebugSend()
}

func RunAgent2(c *client.Client, testVal float64, battleInit func(b *bot.Bot)) int {
	B := &bot.Bot{
		Bot:           scl.New(c, bot.OnUnitCreated),
		PlayDefensive: true,
		BuildPos:      map[scl.BuildingSize]point.Points{},
		CycloneLocks:  map[api.UnitTag]api.UnitTag{},
	}
	bot.B = B
	B.FramesPerOrder = 3
	B.LastLoop = -math.MaxInt
	B.MaxGroup = bot.MaxGroup
	B.Logic = func() {
		bot.DefensivePlayCheck()
		roles.Roles(B)
		macro.Macro(B)
		micro.Micro(B)
	}
	stop := make(chan struct{})
	B.Init(stop) // Called later because in Init() we need to use *B in callback
	B.TestVal = testVal
	battleInit(B)

	for B.Client.Status == api.Status_in_game {
		bot.Step()
		if _, err := c.Step(api.RequestStep{Count: uint32(B.FramesPerOrder)}); err != nil {
			if err.Error() == "Not in a game" {
				break
			}
			log.Fatal(err)
		}

		B.UpdateObservation()
		if B.Loop > scl.TimeToLoop(0, 30) {
			stop <- struct{}{}
			/*return int(B.Obs.Score.ScoreDetails.TotalDamageDealt.Life+B.Obs.Score.ScoreDetails.TotalDamageDealt.Shields-
				-B.Obs.Score.ScoreDetails.TotalDamageTaken.Life-B.Obs.Score.ScoreDetails.TotalDamageTaken.Shields)*/
			return int(B.Obs.Score.ScoreDetails.KilledMinerals.Army+B.Obs.Score.ScoreDetails.KilledVespene.Army-
				-B.Obs.Score.ScoreDetails.LostMinerals.Army-B.Obs.Score.ScoreDetails.LostVespene.Army)
		}
	}
	stop <- struct{}{}
	log.Error("Something went wrong")
	return 0
}

func testBattle() {
	log.SetConsoleLevel(log.L_info) // L_info L_debug
	rand.Seed(time.Now().UnixNano())
	stats := map[api.Race]map[float64][]int{}
	for _, race := range []api.Race{api.Race_Terran, api.Race_Zerg, api.Race_Protoss} {
		stats[race] = map[float64][]int{}
	}

	minVal := 1.2
	maxVal := 1.4
	valStep := 0.05
	testVal := minVal

	var cfg *client.GameConfig
	myBot := client.NewParticipant(api.Race_Terran, "VeTerran")
	cpu := client.NewComputer(api.Race_Terran, api.Difficulty_CheatInsane, api.AIBuild_RandomBuild)
	mapName := client.MapsProBotsSeason2[3]
	round := 1
	for {
		for _, race := range []api.Race{api.Race_Terran, api.Race_Zerg, api.Race_Protoss} {
			var score int
			for score == 0 {
				if cfg == nil {
					// client.SetRealtime()
					client.LaunchPortStart = 8368
					client.SetMap(mapName + ".SC2Map")
					cfg = client.LaunchAndJoin(myBot, cpu)
					if err := cfg.Client.QuickSave(); err != nil {
						log.Error(err)
					}
				} else {
					if err := cfg.Client.QuickLoad(); err != nil {
						log.Error(err)
					}
				}
				switch race {
				case api.Race_Terran:
					score = RunAgent2(cfg.Client, testVal, VsTerran)
				case api.Race_Zerg:
					score = RunAgent2(cfg.Client, testVal, VsZerg)
				case api.Race_Protoss:
					score = RunAgent2(cfg.Client, testVal, VsProtoss)
				}
				log.Infof("Test: %v, Versus: %v, Score: %v", testVal, race, score)
			}
			stats[race][testVal] = append(stats[race][testVal], score)

			fmt.Printf("Round %2d", round)
			for x := minVal; x < maxVal+valStep; x += valStep {
				fmt.Printf("    %.2f", x)
			}
			fmt.Println()
			for _, r := range []api.Race{api.Race_Terran, api.Race_Zerg, api.Race_Protoss} {
				fmt.Printf("%8s", r)
				for x := minVal; x < maxVal+valStep; x += valStep {
					times := stats[r][x]
					avg := 0
					if len(times) > 0 {
						for _, secs := range times {
							avg += secs
						}
						avg /= len(times)
					}
					fmt.Printf(" %7d", avg)
				}
				fmt.Println()
			}
			fmt.Print(" Average")
			for x := minVal; x < maxVal+valStep; x += valStep {
				avg := 0
				results := 0
				for _, r := range []api.Race{api.Race_Terran, api.Race_Zerg, api.Race_Protoss} {
					times := stats[r][x]
					if len(times) > 0 {
						for _, secs := range times {
							avg += secs
						}
						results += len(times)
					}
				}
				if results > 0 {
					avg /= results
				}
				fmt.Printf(" %7d", avg)
			}
			fmt.Println()
		}
		testVal += valStep
		if testVal > maxVal + valStep { // floats...
			testVal = minVal
			round++
		}
	}
}

/*
Radius for AssessStrength
2021-06-08 00:44:03 [Info] benchmark.go:381 - Test: 9, Versus: Zerg, Score: 7525
Round 36    1.00    2.00    3.00    4.00    5.00    6.00    7.00    8.00    9.00   10.00   11.00   12.00   13.00   14.00
  Terran    4339    5676    6211    6430    6509    6545    6425    6492    6618    6606    6603    6580    6620    6589
    Zerg    6020    6698    6654    6404    6561    7015    6892    7316    7122    7237    7257    7297    7317    7296
 Protoss    5068    6669    5877    7376    6813    7550    7297    7515    7344    7304    7412    7382    7236    7340
 Average    5142    6348    6247    6737    6628    7037    6871    7108    7025    7049    7091    7086    7058    7075

Strength multiplier for AssessStrength
2021-06-08 13:13:40 [Info] benchmark.go:381 - Test: 1, Versus: Zerg, Score: 7625
Round 82    1.00    1.10    1.20    1.30    1.40    1.50    1.60    1.70    1.80    1.90    2.00
  Terran    6365    6583    6592    6958    6940    6964    6950    6913    6883    6872    6870
    Zerg    6772    6786    6752    6912    6885    6786    6908    6853    6826    6922    6878
 Protoss    6769    7014    6460    6867    6873    6703    6767    6656    6705    6833    6864
 Average    6635    6794    6601    6912    6899    6818    6875    6807    6805    6876    6871

But same thing:
2021-06-08 13:14:49 [Info] benchmark.go:382 - Test: 1.35, Versus: Terran, Score: 6350
Round 66    1.20    1.25    1.30    1.35    1.40
  Terran    6729    6470    6470    6462    6467
    Zerg    7218    7123    7229    7175    7233
 Protoss    7798    7764    7786    7792    7783
 Average    7248    7119    7162    7139    7161
 */

func main() {
	testBattle()
}