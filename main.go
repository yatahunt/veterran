package main

import (
	"bitbucket.org/AiSee/sc2lib"
	"github.com/chippydip/go-sc2ai/api"
	"github.com/chippydip/go-sc2ai/client"
	"github.com/chippydip/go-sc2ai/runner"
	"log"
	"math/rand"
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

	for b.Info.IsInGame() {
		b.Step()

		if err := b.Info.Step(1); err != nil {
			log.Println(err)
		}
	}
}

func main() {
	maps := []string{"AcidPlantLE", "BlueshiftLE", "CeruleanFallLE", "DreamcatcherLE",
		"FractureLE", "LostAndFoundLE", "ParaSiteLE"}

	rand.Seed(time.Now().UnixNano())
	runner.Set("map", maps[rand.Intn(len(maps))]+".SC2Map")
	// runner.Set("map", "CeruleanFallLE.SC2Map")
	runner.Set("ComputerOpponent", "true")
	runner.Set("ComputerRace", "random")
	runner.Set("ComputerDifficulty", "VeryHard")

	// Create the agent and then start the game
	runner.RunAgent(client.NewParticipant(api.Race_Terran, client.AgentFunc(runAgent), ""))
}
