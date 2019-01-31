package main

import (
	"bitbucket.org/aisee/minilog"
	"bitbucket.org/aisee/veterran/bot"
	"bitbucket.org/aisee/veterran/macro"
	"bitbucket.org/aisee/veterran/micro"
	"bitbucket.org/aisee/veterran/roles"
	"github.com/chippydip/go-sc2ai/api"
	"github.com/chippydip/go-sc2ai/client"
	"github.com/chippydip/go-sc2ai/runner"
	"math/rand"
	"strings"
	"time"
)

// todo: иногда рабочих запирает между зданиями -> поднять здание, которое не может построить аддон?
// todo: уклонение от эффектов не работает, если цель движения не попадает в эффект
// todo: здания строятся рядом с керриерами и сразу уничтожаются
// todo: отступающе отстреливающиеся геллионы слишком легко дохнут
// todo: не строить аддоны, если рядом враги
// todo: ? иногда бот не отменяет строящиеся добиваемые здания?
// todo: надо выбирать цели в соответствии с типом брони и снаряда
// todo: викинги против баньши? Туррели не помогут
// todo: минки боятся рабочих, забегают в угол и тупят -> отслеживать время взрыва и закапывать если по пути к лечению
// todo: ? хрень с хайграундом на автоматоне, юниты идут не туда и дохнут
// todo: надо как-то определять какие здания не стоит чинить, т.к. рабочий будет убит (по числу ranged?)
// todo: строить первый CC на хайграунде если опасно?
// todo: если есть апгрейд для минок, закапывать их, если за ними гонится кто-то быстрее их
// todo: детект спидлингов + крип
// todo: юниты в углах карты могут отвлекать минки
// todo: убрать лишний скан после того как снаряды от убитой баньши долетают до цели
// todo: use dead units events
// todo: анализировать неуспешные попытки строительства, зарытые линги мешают поставить СС -> ставить башню рядом?

func run() {
	log.SetConsoleLevel(log.L_info) // L_info L_debug
	maps := []string{"BlueshiftLE", "CeruleanFallLE", "ParaSiteLE",
		"AutomatonLE", "KairosJunctionLE", "PortAleksanderLE", "StasisLE", "DarknessSanctuaryLE"}

	rand.Seed(time.Now().UnixNano())
	runner.Set("map", maps[rand.Intn(len(maps))]+".SC2Map")
	// runner.Set("map", "DarknessSanctuaryLE.SC2Map")
	runner.Set("ComputerOpponent", "true")
	runner.Set("ComputerRace", "random")           // terran zerg protoss random
	runner.Set("ComputerDifficulty", "CheatMoney") // CheatInsane CheatMoney VeryHard
	// runner.Set("realtime", "true")

	// Create the agent and then start the game
	runner.RunAgent(client.NewParticipant(api.Race_Terran, client.AgentFunc(RunAgent), ""))
}

func RunAgent(info client.AgentInfo) {
	B := bot.B
	B.Info = info
	B.FramesPerOrder = 3
	B.MaxGroup = bot.MaxGroup
	if B.Info.IsRealtime() {
		B.FramesPerOrder = 6
		log.Info("Realtime mode")
	}
	B.UnitCreatedCallback = bot.OnUnitCreated
	B.Logic = func() {
		// time.Sleep(time.Millisecond * 10)
		bot.DefensivePlayCheck()
		roles.Roles()
		macro.Macro()
		micro.Micro()
	}

	for B.Info.IsInGame() {
		bot.Step()

		if err := B.Info.Step(1); err != nil {
			log.Error(err)
			if strings.Contains(err.Error(), "An existing connection was forcibly closed by the remote host.") {
				return
			}
		}
	}
}

func main() {
	/*f, err := os.Create("cpu.prof")
	if err != nil {
		log.Fatal("could not create CPU profile: ", err)
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		log.Fatal("could not start CPU profile: ", err)
	}
	defer pprof.StopCPUProfile()*/

	run()

	/*f, err = os.Create("mem.prof")
	if err != nil {
		log.Fatal("could not create memory profile: ", err)
	}
	runtime.GC() // get up-to-date statistics
	if err := pprof.WriteHeapProfile(f); err != nil {
		log.Fatal("could not write memory profile: ", err)
	}
	f.Close()*/
}
