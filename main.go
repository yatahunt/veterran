package main

import (
	"bitbucket.org/aisee/minilog"
	"bitbucket.org/aisee/veterran/bot"
	"bitbucket.org/aisee/veterran/macro"
	"bitbucket.org/aisee/veterran/micro"
	"bitbucket.org/aisee/veterran/roles"
	"github.com/aiseeq/s2l/lib/point"
	"github.com/aiseeq/s2l/lib/scl"
	"github.com/aiseeq/s2l/protocol/api"
	"github.com/aiseeq/s2l/protocol/client"
	"math/rand"
	"time"
)

// todo: быстрый майнинг (в распределении добычи ориентироваться на не фактическое количество ресурсов, а на прирост)
// todo: пофиксить позиции зданий на разных картах
// todo: рипер раш
// todo: при пересылке рабочих между цц использовать не обычную команду, а безопасную сетку
// todo: есть какие-то проблемы с путями и камнями

// todo: перед отправкой на майнинг газа проверять что рядом есть коммандник
// todo: corrosive bile игнорится
// todo: телепортить баттлов раньше
// todo: не игнорировать свармхостов
// todo: раскладывать танки если целей в радиусе будет много, а близко никого нет
// todo: алгоритм дрючит закопанные мины пытаясь ими отступить не выкапывая, походу
// todo: отступающие юниты должны оценивать скорость преследователей. Если их всё равно догоняют, то отстреливаться
// todo: не строить аддоны, если рядом враги
// todo: закрывать стенку пораньше или двигать с неё юнитов как-то, чтобы она закрывалась (от лингов)
// todo: хеллбатом микрить меньше и геллион тоже иногда глючит
// todo: проверки на крип при строительсте (и закопанные юниты тоже мешают)
// todo: риперы (и танки) крутятся под скалой без вижна, а их атакует морпех сверху (команда атаки на хайграунд)

// todo: баньши, викинги, а то что сразу баттлы?
// todo: Подбитые закопанные перезарядившиеся мины остаются на месте навсегда
// todo: Шарики дисраптеров

// todo: ? нельзя перестраивать маршруты когда меняется pathing grid, иначе образуются неправильные пути
// todo: против протосов не ставится второй барак, м.б. очень жадное развитие, это м.б. опасно
// todo: постройка цц и бункера не отменилась даже когда позиции оказались на экспе чизерга были разведаны рабочими.
// И после этого вообще всё заглючило и экспы больше не ставились (наверное, близость врага вообще отменяет строительство экспов)
// todo: Если много минералов (потому что не можешь выйти на 3-ю), переходить в био
// todo: иногда рабочих запирает между зданиями -> поднять здание, которое не может построить аддон?
// todo: уклонение от эффектов не работает, если цель движения не попадает в эффект
// todo: здания строятся рядом с керриерами и сразу уничтожаются
// todo: отступающе отстреливающиеся геллионы слишком легко дохнут
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
// go tool pprof VeTerran.exe cpu.prof

func RunAgent(c *client.Client) {
	B := &bot.Bot{
		Bot:           scl.New(c, bot.OnUnitCreated),
		PlayDefensive: true,
		BuildPos:      map[scl.BuildingSize]point.Points{},
	}
	bot.B = B
	B.FramesPerOrder = 3
	B.MaxGroup = bot.MaxGroup
	/*if B.Client.Realtime {
		B.FramesPerOrder = 6
		log.Info("Realtime mode")
	}*/
	B.Logic = func() {
		// time.Sleep(time.Millisecond * 10)
		bot.DefensivePlayCheck()
		roles.Roles(B)
		macro.Macro(B)
		micro.Micro(B)
	}
	// todo: Move init into lib
	B.Init() // Called later because in Init() we need to use *B in callback

	for B.Client.Status == api.Status_in_game {
		bot.Step()

		if _, err := c.Step(api.RequestStep{Count: uint32(B.FramesPerOrder)}); err != nil {
			if err.Error() == "Not in a game" {
				log.Info("Game over")
				return
			}
			log.Fatal(err)
		}

		B.UpdateObservation()
	}
}

func run() {
	log.SetConsoleLevel(log.L_info) // L_info L_debug
	rand.Seed(time.Now().UnixNano())

	// Create the agent and then start the game
	myBot := client.NewParticipant(api.Race_Terran, "VeTerran") // todo: rename
	cpu := client.NewComputer(api.Race_Protoss, api.Difficulty_Medium, api.AIBuild_RandomBuild)
	c := client.LaunchAndJoin(myBot, cpu)

	RunAgent(c)
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
