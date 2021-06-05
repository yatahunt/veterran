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
	}
	myId := B.Obs.PlayerCommon.PlayerId
	seconds := int(float64(B.Loop) / scl.FPS)
	log.Infof("Test: %v, Versus: %v, Result: %v, Time: %ds",
		B.TestVal, B.EnemyRace, B.Result[myId-1].Result, seconds)
	if B.Result[myId-1].Result != api.Result_Victory {
		seconds = 10000
	}
	return seconds
}

func main() {
	log.SetConsoleLevel(log.L_info) // L_info L_debug
	rand.Seed(time.Now().UnixNano())
	stats := map[api.Race]map[float64][]int{}
	for _, race := range []api.Race{api.Race_Terran, api.Race_Zerg, api.Race_Protoss} {
		stats[race] = map[float64][]int{}
	}

	minVal := 1.0
	maxVal := 1.5
	valStep := 0.05
	testVal := minVal

	var cfg *client.GameConfig
	myBot := client.NewParticipant(api.Race_Terran, "VeTerran")
	cpu := client.NewComputer(api.Race_Random, api.Difficulty_CheatInsane, api.AIBuild_RandomBuild)
	for round := 1;; round++ {
		for _, mapName := range client.MapsProBotsSeason2 {
			for _, race := range []api.Race{api.Race_Terran, api.Race_Zerg, api.Race_Protoss} {
				cpu.Race = race
				if cfg == nil {
					client.SetMap(mapName + ".SC2Map")
					cfg = client.LaunchAndJoin(myBot, cpu)
				} else {
					cfg.StartGame(mapName + ".SC2Map")
				}

				seconds := RunAgent(cfg.Client, testVal)
				stats[race][testVal] = append(stats[race][testVal], seconds)

				fmt.Printf("Round %2d", round)
				for x := minVal; x < maxVal + valStep; x += valStep {
					fmt.Printf("    %.2f", x)
				}
				fmt.Println()
				for _, r := range []api.Race{api.Race_Terran, api.Race_Zerg, api.Race_Protoss} {
					fmt.Printf("%8s", r)
					for x := minVal; x < maxVal + valStep; x += valStep {
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
				for x := minVal; x < maxVal + valStep; x += valStep {
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
			}
		}
	}
}
