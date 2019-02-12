package tests

import (
	"bitbucket.org/aisee/sc2lib/grid"
	"bitbucket.org/aisee/sc2lib/point"
	"bitbucket.org/aisee/sc2lib/scl"
	"bitbucket.org/aisee/veterran/bot"
	"testing"
)

// go test -v -run=^$ -bench=^Benchmark_Pathing$ -benchtime=2s -cpuprofile=cpu.prof
// go test -v -run=^$ -bench=^Benchmark_Pathing$ -benchtime=2s -memprofile=mem.prof
// go tool pprof tests.test.exe cpu.prof
// go tool pprof tests.test.exe mem.prof

func Benchmark_Pathing(b *testing.B) {
	B := bot.B
	B.LoadState()
	B.Locs.MyStart = point.Pt3(B.Info.Observation().Observation.RawData.Player.Camera)
	B.Locs.EnemyStart = point.Pt2(B.Info.GameInfo().StartRaw.StartLocations[0])

	B.Grid = grid.New(B.Info.GameInfo().StartRaw, B.Info.Observation().Observation.RawData.MapState)
	B.WayMap = B.FindWaypointsMap(B.Grid)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		scl.NavPath(B.Grid, B.WayMap, B.Locs.MyStart-3, B.Locs.EnemyStart-3)
	}
}
