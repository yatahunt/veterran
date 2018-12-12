package main

import (
	"bitbucket.org/AiSee/minilog"
	"bitbucket.org/AiSee/sc2lib"
	"github.com/chippydip/go-sc2ai/api"
	"github.com/chippydip/go-sc2ai/client"
	"github.com/chippydip/go-sc2ai/runner"
	"math/rand"
	"strings"
	"time"
)

type bot struct {
	scl.Bot

	PositionsForSupplies scl.Points
	PositionsForBarracks scl.Points

	Builder1    api.UnitTag
	Builder2    api.UnitTag
	Retreat     map[api.UnitTag]bool
}

func runAgent(info client.AgentInfo) {
	b := bot{}
	b.Info = info
	b.UnitCreatedCallback = b.OnUnitCreated

	for b.Info.IsInGame() {
		b.Step()

		if err := b.Info.Step(1); err != nil {
			log.Error(err)
			if strings.Contains(err.Error(), "An existing connection was forcibly closed by the remote host.") {
				return
			}
		}
	}
}

func main() {
	maps := []string{"BlueshiftLE", "CeruleanFallLE", "ParaSiteLE",
		"AutomatonLE", "KairosJunctionLE", "PortAleksanderLE", "StasisLE", "DarknessSanctuaryLE"}

	rand.Seed(time.Now().UnixNano())
	runner.Set("map", maps[rand.Intn(len(maps))]+".SC2Map")
	// runner.Set("map", "AcidPlantLE.SC2Map") // DarknessSanctuaryLE
	runner.Set("ComputerOpponent", "true")
	runner.Set("ComputerRace", "random")
	runner.Set("ComputerDifficulty", "VeryHard") // CheatInsane CheatMoney VeryHard
	// runner.Set("realtime", "true")

	// Create the agent and then start the game
	runner.RunAgent(client.NewParticipant(api.Race_Terran, client.AgentFunc(runAgent), ""))
}
