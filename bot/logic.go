package bot

import (
	"github.com/aiseeq/s2l/lib/point"
	"github.com/aiseeq/s2l/lib/scl"
	"github.com/aiseeq/s2l/protocol/enums/ability"
	"github.com/aiseeq/s2l/protocol/enums/terran"
)

func FindMainBuildingTypesPositions(startLoc point.Point) (point.Points, point.Points, point.Points) {
	var pf2x2, pf3x3, pf5x3 point.Points
	slh := B.Grid.HeightAt(startLoc)
	start := startLoc + 9

	for y := -6.0; y <= 6; y++ {
		for x := -9.0; x <= 9; x++ {
			pos := start + point.Pt(3, 2).Mul(x) + point.Pt(-6, 8).Mul(y)
			if B.Grid.HeightAt(pos) == slh && B.IsPosOk(pos, scl.S3x3, 2, scl.IsBuildable) {
				if B.IsPosOk(pos+2-1i, scl.S2x2, 2, scl.IsBuildable) {
					pf5x3.Add(pos.CellCenter())
				} else {
					pf3x3.Add(pos.CellCenter())
				}
			}
			pos += 2 - 3i
			if B.Grid.HeightAt(pos) == slh && B.IsPosOk(pos, scl.S3x3, 2, scl.IsBuildable) {
				if B.IsPosOk(pos+2-1i, scl.S2x2, 2, scl.IsBuildable) {
					pf5x3.Add(pos.CellCenter())
				} else {
					pf3x3.Add(pos.CellCenter())
				}
			}
			pos += 1 - 3i
			if B.Grid.HeightAt(pos) == slh && B.IsPosOk(pos, scl.S2x2, 1, scl.IsBuildable) {
				pf2x2.Add(pos.CellCenter())
			}
			pos += 2
			if B.Grid.HeightAt(pos) == slh && B.IsPosOk(pos, scl.S2x2, 1, scl.IsBuildable) {
				pf2x2.Add(pos.CellCenter())
			}
		}
	}
	return pf2x2, pf3x3, pf5x3
}

// Mark geysers as unbildable because somehow undiscovered geysers become buildable
func MarkGeysersAsUnbuildable() {
	points := []point.Points{}
	for _, geyser := range B.Units.Geysers.All() {
		points = append(points, B.GetBuildingPoints(geyser, scl.S3x3))
		for _, ps := range points {
			for _, p := range ps {
				B.Grid.SetBuildable(p, false)
				B.Grid.SetPathable(p, false)
			}
		}
	}
}

func FindBuildingsPositions() {
	MarkGeysersAsUnbuildable()
	FindTurretPosition(B.Locs.MyStart)

	// Positions for first 2 supplies and barrack
	// todo: grid := grid.New(B.Grid.StartRaw, B.Grid.MapState) - держать отдельную сетку с пометками где что строить собрался
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
			B.TurretsPos.Add(pos.CellCenter())
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
				pf3x3.Add(pos.CellCenter())
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

	pf2x2 = append(rp2x2, pf2x2...)

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
}

func FindTurretPosition(ptr point.Pointer) {
	mfs := B.Units.Minerals.All().CloserThan(scl.ResourceSpreadDistance, ptr)
	if mfs.Empty() {
		return
	}

	MarkGeysersAsUnbuildable()

	var pos point.Point
	vec := (mfs.Center() - ptr.Point()).Norm()
	rekt := mfs.Rekt()
	rekt[0] -= 1+1i // Move LOWER point DOWN and LEFT because 2x2 building is placed on lower-left corner
	side := vec.Compas()
	if side.IsDiagonal() {
		// Minerals are in quarter of a circle. We need do find corners of rectangle that wraps all crystals
		if side == point.NE || side == point.SW {
			// Switch corners
			rekt[0], rekt[1] = point.Pt(rekt[0].X(), rekt[1].Y()), point.Pt(rekt[1].X(), rekt[0].Y())
		}
	} else {
		// Move closest to CC point behind the mineral line
		switch side {
		case point.E:
			rekt[0] = point.Pt(rekt[1].X(), rekt[0].Y())
		case point.W:
			rekt[1] = point.Pt(rekt[0].X(), rekt[1].Y())
		case point.N:
			rekt[0] = point.Pt(rekt[0].X(), rekt[1].Y())
		case point.S:
			rekt[1] = point.Pt(rekt[1].X(), rekt[0].Y())
		}
	}
	// B.DebugPoints(rekt...)

	for _, corner := range rekt {
		if pos = B.FindClosestPos(corner, scl.S2x2, 0, 0, 1, scl.IsBuildable, scl.IsPathable, scl.IsNoCreep); pos == 0 {
			pos = B.FindClosestPos(corner, scl.S2x2, 0, 1, 1, scl.IsBuildable, scl.IsPathable, scl.IsNoCreep)
		}
		if pos == 0 {
			continue
		}
		pos = pos.CellCenter()
		if !B.TurretsPos.Has(pos) {
			B.TurretsPos.Add(pos)
		}
		for _, p := range B.GetBuildingPoints(pos, scl.S2x2) {
			B.Grid.SetBuildable(p, false)
			B.Grid.SetPathable(p, false)
		}
	}
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
		pos = (ccPos - vec.Mul(x)).Floor().CellCenter()
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

func RecalcEnemyStartLoc(np point.Point) { // Used on maps where there could be more than 2 players
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
		townHalls := append(B.U.UnitAliases.For(terran.CommandCenter), B.U.UnitAliases.For(zerg.Hatchery)...)
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
