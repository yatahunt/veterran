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

DefensivePlayCheck Attack multiplier
2021-06-08 15:04:06 [Info] benchmark.go:65 - Test: 3, Versus: Protoss, Result: Victory, Time: 1135s
Round  9    1.00    1.25    1.50    1.75    2.00    2.25    2.50    2.75    3.00
  Terran    1282    1247    1019    1149    1179     788     990     830     903
    Zerg    1382     471     514     563    1014     498    1434    1275     456
 Protoss    1013     941     921     809     768    1346     878     944    1135
 Average    1225     886     818     840     987     877    1100    1016     831

2021-06-09 09:05:05 [Info] benchmark.go:65 - Test: 1.25, Versus: Protoss, Result: Victory, Time: 895s
Round 74    1.00    1.25    1.50    1.75    2.00    2.25    2.50    2.75    3.00
  Terran    1051    1011    1036     996    1091     918     958     985     952
    Zerg     934    1030    1124     777     795     868     621     899     690
 Protoss    1039     996     908     902     923     886     872     951    1044
 Average    1008    1012    1023     891     937     891     817     945     896
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
				B.Obs.Score.ScoreDetails.TotalDamageTaken.Life-B.Obs.Score.ScoreDetails.TotalDamageTaken.Shields)*/
			return int(B.Obs.Score.ScoreDetails.KilledMinerals.Army+B.Obs.Score.ScoreDetails.KilledVespene.Army-
				B.Obs.Score.ScoreDetails.LostMinerals.Army-B.Obs.Score.ScoreDetails.LostVespene.Army)
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

	minVal := 1.0
	maxVal := 1.0
	valStep := 1.0
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
					// client.LaunchPortStart = 8268
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
		if testVal > maxVal { // + valStep { // floats...
			testVal = minVal
			round++
		}
	}
}

/*
Radius for AssessStrength
2021-06-10 00:43:17 [Info] benchmark.go:397 - Test: 13, Versus: Zerg, Score: 3975
Round 92    1.00    2.00    3.00    4.00    5.00    6.00    7.00    8.00    9.00    10.00   11.00   12.00   13.00   14.00
  Terran   -1426     533    1930    2805    3507    3679    3866    3535    3451    3663    3673    3663    3677    3688
    Zerg    3991    3505    3717    3322    4190    4267    4296    4195    4255    4181    4221    4296    4301    4220
 Protoss    1544    1141    2668    3348    3776    2944    3207    3523    3453    3265    3615    3322    3298    3308
 Average    1369    1726    2771    3158    3824    3630    3790    3751    3720    3703    3836    3760    3761    3739

Strength multiplier for AssessStrength
2021-06-10 00:39:44 [Info] benchmark.go:397 - Test: 1.7000000000000006, Versus: Zerg, Score: 4750
Round 36    1.00    1.10    1.20    1.30    1.40    1.50    1.60    1.70    1.80    1.90    2.00
  Terran    3634    3681    4125    3875    3993    3809    3686    3701    3747    3663    3706
    Zerg    4541    4115    3859    4070    4359    4133    4313    4258    4272    4313    4135
 Protoss    3360    3205    2251    2588    3048    3360    2855    2475    3393    2896    3551
 Average    3845    3667    3412    3511    3800    3767    3618    3487    3804    3624    3797
 */

func main() {
	testBattle()
}