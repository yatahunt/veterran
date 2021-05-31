package tests

import (
	"bitbucket.org/aisee/veterran/bot"
	"github.com/aiseeq/s2l/protocol/api"
	"github.com/aiseeq/s2l/protocol/enums/protoss"
	"github.com/aiseeq/s2l/protocol/enums/terran"
	"github.com/aiseeq/s2l/protocol/enums/zerg"
)

func VikingVsCollosus(myId, enemyId api.PlayerID, b *bot.Bot) {
	b.DebugAddUnits(protoss.Colossus, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 4), 1)
	// b.DebugAddUnits(protoss.WarpPrism, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 4), 1)
	// b.DebugAddUnits(terran.MissileTurret, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 8), 1)
	b.DebugAddUnits(terran.VikingFighter, myId, b.Locs.MyStart.Towards(b.Locs.MapCenter, -10), 1)
	// b.DebugAddUnits(terran.Banshee, myId, b.Locs.MyStart.Towards(b.Locs.MapCenter, -10), 1)
	// b.DebugAddUnits(terran.SiegeTank, myId, b.Locs.MyStart.Towards(b.Locs.MapCenter, -10), 1)
	b.DebugSend()
}

func ReapersVsDarks(myId, enemyId api.PlayerID, b *bot.Bot) {
	b.DebugAddUnits(protoss.DarkTemplar, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 4), 1)
	b.DebugAddUnits(terran.Marine, myId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 8), 5)
	b.DebugSend()
}

func BansheeTest(myId, enemyId api.PlayerID, b *bot.Bot) {
	// b.DebugAddUnits(terran.Marine, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 4), 3)
	// b.DebugAddUnits(terran.MissileTurret, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, -6), 1)
	b.DebugAddUnits(protoss.Stalker, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 6), 1)
	b.DebugAddUnits(terran.Banshee, myId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 10), 1)
	b.DebugSend()
	b.Actions.MoveCamera(b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 8))
}

func BattleCruiserTest(myId, enemyId api.PlayerID, b *bot.Bot) { // todo: evasion?
	b.DebugAddUnits(terran.Marine, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 4), 10)
	b.DebugAddUnits(terran.Battlecruiser, myId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 10), 1)
	b.DebugAddUnits(terran.VikingFighter, myId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 10), 2)
	b.DebugSend()
	b.Actions.MoveCamera(b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 8))
}

func CycloneTest(myId, enemyId api.PlayerID, b *bot.Bot) {
	// b.DebugAddUnits(protoss.Tempest, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 4), 1)
	b.DebugAddUnits(zerg.Roach, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 4), 1)
	b.DebugAddUnits(terran.Cyclone, myId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 6), 1)
	b.DebugSend()
	b.Actions.MoveCamera(b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 8))
}

func HellionTest(myId, enemyId api.PlayerID, b *bot.Bot) {
	// b.DebugAddUnits(protoss.Zealot, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 6), 1)
	// b.DebugAddUnits(zerg.Zergling, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 4), 4)
	b.DebugAddUnits(terran.Marine, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 6), 2)
	// b.DebugAddUnits(terran.Hellion, myId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 8), 1)
	b.DebugAddUnits(terran.HellionTank, myId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 8), 1)
	b.DebugSend()
	b.Actions.MoveCamera(b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 8))
}

func MarauderTest(myId, enemyId api.PlayerID, b *bot.Bot) {
	// b.DebugAddUnits(protoss.Zealot, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 6), 1)
	// b.DebugAddUnits(zerg.Zergling, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 4), 4)
	// b.DebugAddUnits(terran.Marine, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 4), 2)
	b.DebugAddUnits(zerg.Roach, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 4), 1)
	b.DebugAddUnits(terran.Marauder, myId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 8), 1)
	b.DebugSend()
	b.Actions.MoveCamera(b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 8))
}

func MarineTest(myId, enemyId api.PlayerID, b *bot.Bot) {
	// b.DebugAddUnits(protoss.Zealot, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 6), 1)
	// b.DebugAddUnits(zerg.Zergling, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 4), 6)
	// b.DebugAddUnits(zerg.Mutalisk, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 4), 1)
	// b.DebugAddUnits(zerg.Roach, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 3), 1)
	// b.DebugAddUnits(protoss.Stalker, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 6), 1)
	b.DebugAddUnits(zerg.Queen, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 6), 1)
	b.DebugAddUnits(terran.Marine, myId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 12), 4)
	b.DebugSend()
	b.Actions.MoveCamera(b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 8))
}

func ReaperTest(myId, enemyId api.PlayerID, b *bot.Bot) {
	// b.DebugAddUnits(protoss.Zealot, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 4), 1)
	// b.DebugAddUnits(zerg.Zergling, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 4), 4)
	// b.DebugAddUnits(terran.Marine, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 4), 2)
	b.DebugAddUnits(protoss.Stalker, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 4), 1)
	b.DebugAddUnits(terran.Reaper, myId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 8), 3)
	b.DebugSend()
	b.Actions.MoveCamera(b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 8))
}

func TankTest(myId, enemyId api.PlayerID, b *bot.Bot) {
	// b.DebugAddUnits(protoss.Zealot, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 4), 1)
	// b.DebugAddUnits(zerg.Zergling, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 4), 4)
	// b.DebugAddUnits(terran.Marine, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 4), 3)
	b.DebugAddUnits(protoss.Stalker, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 4), 1)
	b.DebugAddUnits(terran.SiegeTank, myId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 8), 1)
	// b.DebugAddUnits(terran.SiegeTankSieged, myId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, -9), 1)
	b.DebugSend()
	b.Actions.MoveCamera(b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 8))
}

func ThorTest(myId, enemyId api.PlayerID, b *bot.Bot) {
	b.DebugAddUnits(zerg.Zergling, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 4), 2)
	b.DebugAddUnits(zerg.Mutalisk, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 4), 1)
	// b.DebugAddUnits(terran.Marine, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 4), 3)
	// b.DebugAddUnits(zerg.Overlord, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 12), 1)
	// b.DebugAddUnits(zerg.Roach, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 3), 1)
	b.DebugAddUnits(terran.Thor, myId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 8), 1)
	b.DebugSend()
	b.Actions.MoveCamera(b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 8))
}

func VikingTest(myId, enemyId api.PlayerID, b *bot.Bot) {
	// b.DebugAddUnits(terran.MissileTurret, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 6), 1)
	// b.DebugAddUnits(zerg.Overlord, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 0), 1)
	// b.DebugAddUnits(zerg.Mutalisk, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 4), 1)
	b.DebugAddUnits(protoss.Tempest, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 4), 1)
	b.DebugAddUnits(terran.VikingFighter, myId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 28), 2)
	b.DebugSend()
	b.Actions.MoveCamera(b.Locs.MyStart.Towards(b.Locs.MapCenter, 8))
}

func Init(b *bot.Bot) {
	myId := b.Obs.PlayerCommon.PlayerId
	enemyId := 3 - myId
	b.PlayDefensive = false

	MarineTest(myId, enemyId, b)
}
