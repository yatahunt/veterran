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
	info client.AgentInfo
	obs  *api.Observation

	actions            []*api.Action
	startLocation      scl.Point
	enemyStartLocation scl.Point
	baseLocations      scl.Points
	units              scl.UnitsByTypes
	mineralFields      scl.UnitsByTypes
	vespeneGeysers     scl.UnitsByTypes
	neutralUnits       scl.UnitsByTypes
	enemyUnits         scl.UnitsByTypes
	orders             map[api.AbilityID]int

	loop     int
	minerals int
	vespene  int
	foodCap  int
	foodUsed int
	foodLeft int

	positionsForSupplies scl.Points
	positionsForBarracks scl.Points

	okTargets   scl.Units
	goodTargets scl.Units
	builder1    api.UnitTag
	builder2    api.UnitTag
	retreat     map[api.UnitTag]bool
}

func runAgent(info client.AgentInfo) {
	b := bot{info: info}

	for b.info.IsInGame() {
		b.step()

		if err := b.info.Step(1); err != nil {
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
