package main

import (
	"bitbucket.org/aisee/minilog"
	"bitbucket.org/aisee/sc2lib"
	"github.com/chippydip/go-sc2ai/api"
	"github.com/chippydip/go-sc2ai/client"
	"github.com/chippydip/go-sc2ai/runner"
	"math/rand"
	"strings"
	"time"
)

type bot struct {
	scl.Bot
}

func runAgent(info client.AgentInfo) {
	b := bot{}
	b.Info = info
	b.FramesPerOrder = 3
	b.MaxGroup = MaxGroup
	if b.Info.IsRealtime() {
		b.FramesPerOrder = 6
		log.Info("Realtime mode")
	}
	b.UnitCreatedCallback = b.OnUnitCreated

	for b.Info.IsInGame() {
		if b.Step() {
			break
		}

		if err := b.Info.Step(1); err != nil {
			log.Error(err)
			if strings.Contains(err.Error(), "An existing connection was forcibly closed by the remote host.") {
				return
			}
		}
	}
}

func main() {
	log.SetConsoleLevel(log.L_info) // L_info L_debug
	maps := []string{"BlueshiftLE", "CeruleanFallLE", "ParaSiteLE",
		"AutomatonLE", "KairosJunctionLE", "PortAleksanderLE", "StasisLE", "DarknessSanctuaryLE"}

	rand.Seed(time.Now().UnixNano())
	runner.Set("map", maps[rand.Intn(len(maps))]+".SC2Map")
	// runner.Set("map", "ParaSiteLE.SC2Map")
	runner.Set("ComputerOpponent", "true")
	runner.Set("ComputerRace", "random")           // terran zerg protoss random
	runner.Set("ComputerDifficulty", "CheatMoney") // CheatInsane CheatMoney VeryHard
	// runner.Set("realtime", "true")

	// Create the agent and then start the game
	runner.RunAgent(client.NewParticipant(api.Race_Terran, client.AgentFunc(runAgent), ""))
}
