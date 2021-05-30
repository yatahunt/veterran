package tests

import (
	"bitbucket.org/aisee/veterran/bot"
	"github.com/aiseeq/s2l/protocol/enums/protoss"
	"github.com/aiseeq/s2l/protocol/enums/terran"
	"github.com/aiseeq/s2l/protocol/enums/zerg"
)

func VikingVsCollosus(b *bot.Bot) {
	myId := b.Obs.PlayerCommon.PlayerId
	enemyId := 3 - myId
	b.DebugAddUnits(protoss.Colossus, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 4), 1)
	// b.DebugAddUnits(protoss.WarpPrism, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 4), 1)
	// b.DebugAddUnits(terran.MissileTurret, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 8), 1)
	b.DebugAddUnits(terran.VikingFighter, myId, b.Locs.MyStart.Towards(b.Locs.MapCenter, -10), 1)
	// b.DebugAddUnits(terran.Banshee, myId, b.Locs.MyStart.Towards(b.Locs.MapCenter, -10), 1)
	// b.DebugAddUnits(terran.SiegeTank, myId, b.Locs.MyStart.Towards(b.Locs.MapCenter, -10), 1)
	b.DebugSend()
}

func ReapersVsDarks(b *bot.Bot) {
	myId := b.Obs.PlayerCommon.PlayerId
	enemyId := 3 - myId
	b.DebugAddUnits(protoss.DarkTemplar, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 4), 1)
	b.DebugAddUnits(terran.Reaper, myId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 8), 1)
	b.DebugSend()
}

func KitingTest(b *bot.Bot) {
	myId := b.Obs.PlayerCommon.PlayerId
	enemyId := 3 - myId
	b.PlayDefensive = false
	b.DebugAddUnits(protoss.Zealot, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 6), 1)
	// b.DebugAddUnits(terran.MissileTurret, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 4), 1)
	// b.DebugAddUnits(zerg.Zergling, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 4), 2)
	// b.DebugAddUnits(zerg.Mutalisk, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 4), 2)
	// b.DebugAddUnits(terran.Marine, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 4), 2)
	// b.DebugAddUnits(zerg.Overlord, enemyId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 12), 8)
	b.DebugAddUnits(terran.Reaper, myId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 4), 1)
	// b.DebugAddUnits(terran.Thor, myId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 8), 1)
	// b.DebugAddUnits(terran.SiegeTank, myId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 8), 1)
	// b.DebugAddUnits(terran.SiegeTankSieged, myId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, -9), 1)
	// b.DebugAddUnits(terran.Banshee, myId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 8), 1)
	// b.DebugAddUnits(terran.Cyclone, myId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 12), 1)
	// b.DebugAddUnits(terran.Hellion, myId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 8), 1)
	// b.DebugAddUnits(terran.HellionTank, myId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 4), 1)
	// b.DebugAddUnits(terran.Marauder, myId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 8), 1)
	// b.DebugAddUnits(terran.VikingFighter, myId, b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 8), 1)
	b.DebugSend()
	b.Actions.MoveCamera(b.Locs.EnemyStart.Towards(b.Locs.MapCenter, 8))
}

func AirEvade(b *bot.Bot) {
	myId := b.Obs.PlayerCommon.PlayerId
	enemyId := 3 - myId
	b.PlayDefensive = false
	b.DebugAddUnits(terran.MissileTurret, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 8), 1)
	b.DebugAddUnits(zerg.Overlord, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 4), 1)
	// b.DebugAddUnits(zerg.Mutalisk, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 4), 2)
	// b.DebugAddUnits(protoss.Tempest, enemyId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 4), 1)
	b.DebugAddUnits(terran.VikingFighter, myId, b.Locs.MyStart.Towards(b.Locs.MapCenter, 16), 1)
	b.DebugSend()
	b.Actions.MoveCamera(b.Locs.MyStart.Towards(b.Locs.MapCenter, 8))
}

func Init(b *bot.Bot) {
	// VikingVsCollosus(B)
	// ReapersVsDarks(B)
	KitingTest(b)
	// AirEvade(b)
}
