package bot

import (
	"bitbucket.org/aisee/sc2lib/point"
	"bitbucket.org/aisee/sc2lib/scl"
	"github.com/chippydip/go-sc2ai/enums/ability"
	"github.com/chippydip/go-sc2ai/enums/terran"
)

func FindMainBuildingTypesPositions(startLoc point.Point) (point.Points, point.Points, point.Points) {
	var pf2x2, pf3x3, pf5x3 point.Points
	slh := B.Grid.HeightAt(startLoc)
	start := startLoc + 9

	for y := -3.0; y <= 3; y++ {
		for x := -9.0; x <= 9; x++ {
			pos := start + point.Pt(3, 2).Mul(x) + point.Pt(-6, 8).Mul(y)
			if B.Grid.HeightAt(pos) == slh && B.IsPosOk(pos, scl.S3x3, 2, scl.IsBuildable) {
				if B.IsPosOk(pos+2-1i, scl.S2x2, 2, scl.IsBuildable) {
					pf5x3.Add(pos)
				} else {
					pf3x3.Add(pos)
				}
			}
			pos += 2 - 3i
			if B.Grid.HeightAt(pos) == slh && B.IsPosOk(pos, scl.S3x3, 2, scl.IsBuildable) {
				if B.IsPosOk(pos+2-1i, scl.S2x2, 2, scl.IsBuildable) {
					pf5x3.Add(pos)
				} else {
					pf3x3.Add(pos)
				}
			}
			pos += 1 - 3i
			if B.Grid.HeightAt(pos) == slh && B.IsPosOk(pos, scl.S2x2, 2, scl.IsBuildable) {
				pf2x2.Add(pos)
			}
			pos += 2
			if B.Grid.HeightAt(pos) == slh && B.IsPosOk(pos, scl.S2x2, 2, scl.IsBuildable) {
				pf2x2.Add(pos)
			}
		}
	}
	return pf2x2, pf3x3, pf5x3
}

func FindBuildingsPositions() {
	// Positions for first 2 supplies and barrack
	// todo: grid := grid.New(B.Grid.StartRaw, B.Grid.MapState)
	rp2x2 := B.FindRamp2x2Positions(B.Ramps.My)
	B.FirstBarrack = B.FindRampBarracksPositions(B.Ramps.My)
	if B.FirstBarrack.Len() > 1 && rp2x2.Len() > 1 {
		points := []point.Points{
			B.GetBuildingPoints(B.FirstBarrack[0], scl.S3x3),
			B.GetBuildingPoints(B.FirstBarrack[1], scl.S5x3), // Take second position with addon
			B.GetBuildingPoints(rp2x2[0], scl.S2x2),
			B.GetBuildingPoints(rp2x2[1], scl.S2x2),
		}
		for _, ps := range points {
			for _, p := range ps {
				B.Grid.SetBuildable(p, false)
				B.Grid.SetPathable(p, false)
			}
		}
		// Position for turret
		closeSupply := rp2x2.ClosestTo(B.Locs.MyStart)
		pos := B.FindClosestPos(closeSupply, scl.S2x2, 0, 2, 2, scl.IsBuildable, scl.IsPathable)
		if pos != 0 {
			B.TurretsPos.Add(pos.S2x2Fix())
		}
	}

	// Positions for main base buildings
	pf2x2, pf3x3, pf5x3 := FindMainBuildingTypesPositions(B.Locs.MyStart)

	// Mark buildings positions as non-buildable
	for size, poses := range map[scl.BuildingSize]point.Points{
		scl.S2x2: append(rp2x2, pf2x2...),
		scl.S3x3: pf3x3,
		scl.S5x3: pf5x3,
		scl.S5x5: B.Locs.MyExps,
	} {
		for _, pos := range poses {
			for _, p := range B.GetBuildingPoints(pos, size) {
				B.Grid.SetBuildable(p, false)
				B.Grid.SetPathable(p, false)
			}
		}
	}

	// Find 3x3 positions behind mineral line
	mfs := B.Units.Minerals.All().CloserThan(scl.ResourceSpreadDistance, B.Locs.MyStart)
	if mfs.Exists() {
		for y := -12.0; y <= 12; y++ {
			for x := -12.0; x <= 12; x++ {
				vec := point.Pt(x, y)
				dist := vec.Len()
				if dist <= 8 || dist >= 12 {
					continue
				}
				pos := B.Locs.MyStart + vec
				if mfs.ClosestTo(pos).Dist2(pos) >= 9 {
					continue
				}
				if !B.IsPosOk(pos, scl.S3x3, 0, scl.IsBuildable, scl.IsPathable) {
					continue
				}
				pf3x3.Add(pos)
				for _, p := range B.GetBuildingPoints(pos, scl.S3x3) {
					B.Grid.SetBuildable(p, false)
					B.Grid.SetPathable(p, false)
				}
			}
		}
	}

	pf2x2.OrderByDistanceTo(B.Locs.MyStart, false)
	pf3x3.OrderByDistanceTo(B.Locs.MyStart, false)
	pf5x3.OrderByDistanceTo(B.Locs.MyStart, false)

	// Don't build fast wall against protoss, but be ready for worker rush
	// I'll try to defend it other way
	/*if B.EnemyRace == api.Race_Protoss {
		// Insert supplies for wall after pos that is closest to base
		pos := pf2x2.FurthestTo(B.Ramps.My.Top)
		pf2x2 = append(point.Points{pos}, append(rp2x2, pf2x2...)...)
		// Use closest 5x3 position for first barracks
		FirstBarrack[0] = pf5x3.FurthestTo(B.Ramps.My.Top)
	} else {*/
	pf2x2 = append(rp2x2, pf2x2...)
	//}

	// Positions for buildings on expansions
	pf2x2a, pf3x3a, pf5x3a := FindMainBuildingTypesPositions(B.Locs.MyExps[0])
	pf2x2a.OrderByDistanceTo(B.Locs.MyExps[0], false)
	pf3x3a.OrderByDistanceTo(B.Locs.MyExps[0], false)
	pf5x3a.OrderByDistanceTo(B.Locs.MyExps[0], false)
	pf2x2 = append(pf2x2, pf2x2a...)
	pf3x3 = append(pf3x3, pf3x3a...)
	pf5x3 = append(pf5x3, pf5x3a...)

	B.BuildPos[scl.S2x2] = pf2x2
	B.BuildPos[scl.S3x3] = pf3x3
	B.BuildPos[scl.S5x3] = pf5x3
	B.BuildPos[scl.S5x5] = B.Locs.MyExps

	/*B.Debug2x2Buildings(B.BuildPos[scl.S2x2]...)
	B.Debug3x3Buildings(B.BuildPos[scl.S3x3]...)
	B.Debug5x3Buildings(B.BuildPos[scl.S5x3]...)
	B.Debug3x3Buildings(B.BuildPos[scl.S5x5]...)
	B.DebugSend()*/
}

func FindTurretPosition(cc *scl.Unit) {
	mfs := B.Units.Minerals.All().CloserThan(scl.ResourceSpreadDistance, cc)
	vesps := B.Units.Geysers.All().CloserThan(scl.ResourceSpreadDistance, cc)
	mfs.Add(vesps...)
	if mfs.Empty() {
		return
	}

	for _, p := range B.GetBuildingPoints(cc, scl.S5x5) {
		B.Grid.SetBuildable(p, false)
	}

	var pos point.Point
	vec := (mfs.Center() - cc.Point()).Norm()
	for x := 3.0; x < 8; x++ {
		pos = (cc.Point() + vec.Mul(x)).Floor()
		if B.IsPosOk(pos, scl.S2x2, 0, scl.IsBuildable, scl.IsPathable, scl.IsNoCreep) {
			break
		}
		pos = 0
	}
	if pos == 0 {
		return
	}
	pos = pos.S2x2Fix()
	if !B.TurretsPos.Has(pos) {
		B.TurretsPos.Add(pos)
	}
	/*B.Debug2x2Buildings(TurretsPos...)
	B.DebugSend()*/
}

func FindBunkerPosition(ptr point.Pointer) {
	ccPos := ptr.Point().Floor()
	mfs := B.Units.Minerals.All().CloserThan(scl.ResourceSpreadDistance, ccPos)
	vesps := B.Units.Geysers.All().CloserThan(scl.ResourceSpreadDistance, ccPos)
	mfs.Add(vesps...)
	if mfs.Empty() {
		return
	}

	for _, p := range B.GetBuildingPoints(ccPos, scl.S5x5) {
		B.Grid.SetBuildable(p, false)
	}

	var pos point.Point
	vec := (mfs.Center() - ccPos).Norm()
	for x := 3.0; x < 8; x++ {
		pos = (ccPos - vec.Mul(x)).Floor()
		if B.BunkersPos.Has(pos) {
			return // There is already position for one bunker
		}
		if B.IsPosOk(pos, scl.S3x3, 0, scl.IsBuildable, scl.IsPathable, scl.IsNoCreep) {
			B.BunkersPos.Add(pos)
			break
		}
	}
	/*B.Debug3x3Buildings(B.BunkersPos...)
	B.DebugSend()*/
}

func GetEmptyBunker(ptr point.Pointer) *scl.Unit {
	bunkers := B.Units.My[terran.Bunker].Filter(func(unit *scl.Unit) bool {
		return unit.CargoSpaceTaken < unit.CargoSpaceMax
	})
	return bunkers.Min(func(unit *scl.Unit) float64 { return unit.Dist2(ptr) })
}

func RecalcEnemyStartLoc(np point.Point) {
	B.Locs.EnemyStart = np
	B.Locs.EnemyMainCenter = B.FindBaseCenter(B.Locs.EnemyStart)
	B.FindExpansions()
	B.InitRamps()
}

func EnableDefensivePlay() {
	if B.PlayDefensive {
		return
	}
	B.PlayDefensive = true
	for _, cc := range B.Units.My.OfType(terran.CommandCenter, terran.OrbitalCommand, terran.PlanetaryFortress) {
		if cc.IsCloserThan(1, B.Locs.MyStart) {
			continue
		}
		FindBunkerPosition(cc)
	}
}

func DisableDefensivePlay() {
	if !B.PlayDefensive {
		return
	}
	B.PlayDefensive = false
	B.BunkersPos = nil
	if bunkers := B.Units.My[terran.Bunker]; bunkers.Exists() {
		bunkers.Command(ability.UnloadAll_Bunker)
		// bunkers.CommandQueue(ability.Effect_Salvage)
	}
	if tanks := B.Groups.Get(TanksOnExps).Units; tanks.Exists() {
		B.Groups.Add(Tanks, tanks...)
	}
}

func DefensivePlayCheck() {
	armyScore := B.Units.My.All().Filter(scl.NotWorker).Sum(scl.CmpFood)
	enemyScore := B.Enemies.All.Filter(scl.NotWorker).Sum(scl.CmpFood)
	if armyScore > enemyScore*1.5 && B.Obs.Score.ScoreDetails.FoodUsed.Army >= 25 || B.FoodUsed > 180 {
		DisableDefensivePlay()
	} else if armyScore < enemyScore {
		EnableDefensivePlay()
	}
	/*if B.Loop >= 3584 && B.Loop < 3594 { // 2:40
		townHalls := append(scl.UnitAliases.For(terran.CommandCenter), scl.UnitAliases.For(zerg.Hatchery)...)
		townHalls = append(townHalls, protoss.Nexus)
		if B.Units.AllEnemy.OfType(townHalls...).Len() < 2 {
			B.EnableDefensivePlay()
		}
	}*/
	/*if B.Loop < 4480 && B.Units.AllEnemy[protoss.Stalker].Len() > 2 { // 3:20
		B.EnableDefensivePlay()
	}*/
	if B.PlayDefensive {
		buildings := append(B.Groups.Get(Buildings).Units, B.Groups.Get(UnderConstruction).Units...)
		farBuilding := buildings.FurthestTo(B.Locs.MyStart)
		if farBuilding != nil {
			B.DefensiveRange = farBuilding.Dist(B.Locs.MyStart) + 20
		}
	}
}
