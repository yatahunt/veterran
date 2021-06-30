package main

import (
	"bitbucket.org/aisee/minilog"
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

// > todo: Рассчитывать сколько урона сплешем нанесёт танк своим и чужим, и т.о. выбирать лучшую цель

// todo: Логика для скрытых баньши (зоны детекции)
// todo: Линги закопанные на экспах. Теперь и минки! Но на них кидается скан. На лингов не кинуть, их вообще не видно
// todo: Как-то сделать уже так, чтобы строительство рефинери шло по общим законам со всеми проверками
// todo: На Блекбёрне битые юниты толпятся за минералами золотой базы. Проблемы с путём, BresenhamsLineDrawable может
// пройти между двумя минералами стоящими по диагогали, а юниты не могут
// todo: Юниты в углу карты игнорятся? Вражеские баттлы там прячутся

// todo: разведка первым марином
// todo: Было бы очень круто использовать хайграунд для атак без опасения ответа
// todo: А ещё в режиме защиты использовать авиацию чтобы отслеживать положение противника
// todo: Ещё круче - вставать в защиту на его пути
// todo: Починка юнитов стоящих в защитном режиме
// todo: Хватать медиваком юнит, чтобы на него сбился зацеп
// todo: не игнорировать свармхостов
// todo: отступающие юниты должны оценивать скорость преследователей. Если их всё равно догоняют, то отстреливаться
// todo: риперы (и танки) крутятся под скалой без вижна, а их атакует морпех сверху (команда атаки на хайграунд) -
// - пытаются получить вижн двигаясь у цели, а путь к ней заблокирован. Что делать с этим? Скан?
// todo: что-то надо придумать с уходом с рампы и потерей вижна
// todo: Шарики дисраптеров
// todo: алгоритм дрючит закопанные мины пытаясь ими отступить не выкапывая, походу - ну и ок
// todo: Подбитые закопанные перезарядившиеся мины остаются на месте навсегда
// todo: Ускорение мулов?

// todo: иногда рабочих запирает между зданиями -> поднять здание, которое не может построить аддон?
// todo: надо как-то определять какие здания не стоит чинить, т.к. рабочий будет убит (по числу ranged?)
// todo: строить первый CC на хайграунде если опасно?
// todo: если есть апгрейд для минок, закапывать их, если за ними гонится кто-то быстрее их
// todo: детект спидлингов + крип
// todo: анализировать неуспешные попытки строительства, зарытые линги мешают поставить СС -> ставить башню рядом?
// todo: Постановка газилок не проходит всех проверок обычных зданий

// go tool pprof VeTerran.exe cpu.prof

var PreviousTime int64

func EmulateRealtime(speed float64) {
	delta := time.Now().UnixNano() - PreviousTime
	delay := int64(1000000000.0 / scl.FPS / speed)
	toWait := delay - delta
	if toWait > 0 {
		time.Sleep(time.Duration(toWait))
	}
	PreviousTime = time.Now().UnixNano()
}

func ChooseStrategy(B *bot.Bot) {
	B.Strategy = bot.Default
	if B.Stats.LastResult == "Victory" {
		B.Strategy = B.Stats.LastStrategy
	} else {
		bestRatio := 0.0
		for s := bot.Default; s < bot.MaxStrategyId; s++ {
			if s == bot.ProxyReapers || s == bot.ProxyMarines {
				continue // Disable until the tournament
			}
			if B.Stats.LastStrategy == s && B.Stats.LastResult != "Victory" {
				continue
			}
			ratio := bot.StrategyPriority[s] // Default ratio for unused strategy
			h := B.Stats.History[s]
			if h.Victories > 0 || h.Defeats > 0 {
				ratio = float64(h.Victories+1) / (float64(h.Victories + h.Defeats + 1))
			}
			if ratio >= bestRatio {
				B.Strategy = s
				bestRatio = ratio
			}
		}
	}
	// B.Strategy = bot.Default
	B.ProxyReapers = B.Strategy == bot.ProxyReapers
	B.ProxyMarines = B.Strategy == bot.ProxyMarines
	B.BruteForce = B.Strategy == bot.BruteForce
	B.CcAfterRax = B.Strategy == bot.CcAfterRax
	B.CcBeforeRax = B.Strategy == bot.CcBeforeRax
	log.Infof("Game versus: %s, strategy: %d", client.LadderOpponentID, B.Strategy)
}

func RunAgent(c *client.Client) {
	B := &bot.Bot{
		Bot:           scl.New(c, bot.OnUnitCreated),
		PlayDefensive: true,
		BuildPos:      map[scl.BuildingSize]point.Points{},
		CycloneLocks:  map[api.UnitTag]api.UnitTag{},
	}
	bot.B = B

	B.Stats = bot.LoadGameData()
	ChooseStrategy(B)

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
	// tests.Init(B)

	for B.Client.Status == api.Status_in_game {
		bot.Step()
		/*if B.Enemies.Visible.First(scl.Visible) != nil && B.Loop > scl.TimeToLoop(4, 0) {
			EmulateRealtime(2 / float64(B.FramesPerOrder))
		}*/

		if _, err := c.Step(api.RequestStep{Count: uint32(B.FramesPerOrder)}); err != nil {
			if err.Error() == "Not in a game" {
				break
			}
			log.Error(err)
			break
		}

		B.UpdateObservation()
	}

	stop <- struct{}{}
	if len(B.Result) == 0 || B.Obs == nil {
		B.UpdateObservation()
		if len(B.Result) == 0 || B.Obs == nil {
			log.Error("Failed to get game result")
			bot.SaveGameData(B.Stats, B.Strategy, "Defeat")
			return
		}
	}
	myId := B.Obs.PlayerCommon.PlayerId
	log.Infof("Game versus: %v, Result: %v, Time: %ds",
		B.EnemyRace, B.Result[myId-1].Result, int(float64(B.Loop)/scl.FPS))

	bot.SaveGameData(B.Stats, B.Strategy, B.Result[myId-1].Result.String())
}

func run() {
	log.SetConsoleLevel(log.L_info) // L_info L_debug
	rand.Seed(time.Now().UnixNano())

	// Create the agent and then start the game
	// client.SetMap(client.MapsProBotsSeason2[3] + ".SC2Map")
	// client.SetMap(client.Maps2021season1[3] + ".SC2Map")
	// client.SetGameVersion(75689, "B89B5D6FA7CBF6452E721311BFBC6CB2")
	// client.SetRealtime()
	myBot := client.NewParticipant(api.Race_Terran, "VeTerran")
	cpu := client.NewComputer(api.Race_Random, api.Difficulty_CheatInsane, api.AIBuild_RandomBuild)
	cfg := client.LaunchAndJoin(myBot, cpu)
	if client.LadderOpponentID == "" {
		client.LadderOpponentID = fmt.Sprintf("%s-%s", cpu.Race, cpu.AiBuild)
	}

	RunAgent(cfg.Client)
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

	// bot.DebugGameData()
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
