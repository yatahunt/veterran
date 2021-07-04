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

	for y := -3.0; y <= 3; y++ {
		for x := -8.0; x <= 8; x++ {
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
	for _, exp := range B.Locs.MyExps {
		FindTurretPosition(exp)
		FindBunkerPosition(exp)
	}

	// Make positions for mining targets calculations
	B.TurretsMiningPos = make(point.Points, len(B.TurretsPos))
	copy(B.TurretsMiningPos, B.TurretsPos)
	for n := range B.TurretsMiningPos {
		B.TurretsMiningPos[n] += 0.5+0.5i
	}

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
		pos := B.FindClosestPos(closeSupply, scl.S2x2, ability.Build_MissileTurret,
			0, 2, 2, scl.IsBuildable, scl.IsPathable)
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
	if mfs.Len() != 8 {
		return // Unorthodox expansion, its better to skip it
	}
	mfsCenter := mfs.Center()

	var corners point.Points
	ccVec := (mfsCenter - ptr.Point()).Norm()
	minSide := ccVec.Compas()
	ccDir := mfsCenter.Dir(ptr)
	geysers := B.Units.Geysers.All().CloserThan(10, ptr)
	if minSide.IsDiagonal() {
		// Minerals are in quarter-circle
		// We need to find minerals on the edge of field
		mfs.OrderByDistanceTo(mfsCenter, true)           // furthest are in corners
		corners = append(corners, mfs[0].Point()-1-0.5i) // Bottom left corner of the mineral field
		for _, mf := range mfs[1:4] {
			if mf.IsCloserThan(4, mfs[0]) {
				continue
			}
			corners = append(corners, mf.Point()-1-0.5i)
			break
		}
		// Move corners so they become turrets positions
		for n, corner := range corners {
			side := (corner - ptr.Point()).Compas()
			corners[n] += ccDir
			switch side {
			case point.N:
				corners[n] -= 1i
			case point.S:
				// nothing
			case point.E:
				corners[n] -= 1
				if imag(ccDir) == -1 {
					corners[n] -= 1i
				}
			case point.W:
				corners[n] += 1
				if imag(ccDir) == -1 {
					corners[n] -= 1i
				}
			}
		}
		// Check if geysers are close
		if len(geysers) == 2 && geysers[0].Dist2(geysers[1]) < 8*8 {
			geysersCenter := geysers.Center()
			n := 0
			if geysersCenter.Dist2(corners[1]) < geysersCenter.Dist2(corners[0]) {
				n = 1
			}
			// Here we need to move position horizontally or vertically to touch both geysers
			furthestGeyser := geysers.FurthestTo(corners[n])
			side := (corners[n] - furthestGeyser.Point()).Compas()
			switch side {
			case point.N:
				corners[n].SetY(furthestGeyser.Point().Y() + 1.5)
			case point.S:
				corners[n].SetY(furthestGeyser.Point().Y() - 3.5)
			case point.E:
				corners[n].SetX(furthestGeyser.Point().X() + 1.5)
			case point.W:
				corners[n].SetX(furthestGeyser.Point().X() - 3.5)
			}
		}
	} else {
		// Minerals are in line or half-circle
		for _, geyser := range geysers {
			corner := geyser.Point() - 1.5 - 1.5i
			side := (geyser.Point() - geysers.Center()).Compas()
			switch side {
			case point.N:
				corner -= 2i
			case point.S:
				corner += 3i
			case point.E:
				corner -= 2
			case point.W:
				corner += 3
			}
			switch minSide {
			case point.N:
				corner += 1i
			case point.E:
				corner += 1
			}
			corners = append(corners, corner)
		}
	}
	// B.DebugPoints(corners...)

	var pos point.Point
	for _, corner := range corners {
		if !B.IsPosOk(corner, scl.S2x2, 0, scl.IsBuildable, scl.IsPathable, scl.IsNoCreep)  {
			continue
		}
		pos = corner.CellCenter()
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
	minVec := (mfs.Center() - ccPos).Norm()
	centerVec := (ccPos - B.Locs.MapCenter).Norm()
	for x := 4.0; x < 8; x++ {
		pos = (ccPos - minVec.Mul(x)).Floor().CellCenter()
		if B.IsPosOk(pos, scl.S3x3, 0, scl.IsBuildable, scl.IsPathable, scl.IsNoCreep) {
			B.BunkersPos.Add(pos)
			for _, p := range B.GetBuildingPoints(pos, scl.S3x3) {
				B.Grid.SetBuildable(p, false)
				B.Grid.SetPathable(p, false)
			}
			break
		}
		pos = (ccPos - centerVec.Mul(x)).Floor().CellCenter()
		if B.IsPosOk(pos, scl.S3x3, 0, scl.IsBuildable, scl.IsPathable, scl.IsNoCreep) {
			B.BunkersPos.Add(pos)
			for _, p := range B.GetBuildingPoints(pos, scl.S3x3) {
				B.Grid.SetBuildable(p, false)
				B.Grid.SetPathable(p, false)
			}
			break
		}
	}
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
}

func DisableDefensivePlay() {
	if !B.PlayDefensive {
		return
	}
	B.PlayDefensive = false
	if bunkers := B.Units.My[terran.Bunker]; bunkers.Exists() {
		bunkers.Command(ability.UnloadAll_Bunker)
		// bunkers.CommandQueue(ability.Effect_Salvage)
	}
}

func DefensivePlayCheck() {
	armyScore := B.Units.MyAll.Filter(scl.NotWorker).Sum(scl.CmpFood)
	enemyScore := B.Enemies.All.Filter(scl.NotWorker).Sum(scl.CmpFood)
	balance := B.Obs.Score.ScoreDetails.KilledMinerals.Army + B.Obs.Score.ScoreDetails.KilledVespene.Army -
		B.Obs.Score.ScoreDetails.LostMinerals.Army - B.Obs.Score.ScoreDetails.LostVespene.Army
	if B.ProxyReapers || B.ProxyMarines || B.FoodUsed > 180 ||
		armyScore > enemyScore*2 && balance >= 0 &&
			(B.Obs.Score.ScoreDetails.FoodUsed.Army >= 25 ||
				B.BruteForce && B.Obs.Score.ScoreDetails.FoodUsed.Army >= 16 && B.Loop < scl.TimeToLoop(4, 0)) {
		DisableDefensivePlay()
	} else if armyScore < enemyScore {
		EnableDefensivePlay()
	}

	if B.PlayDefensive {
		buildings := append(B.Groups.Get(Buildings).Units, B.Groups.Get(UnderConstruction).Units...)
		farBuilding := buildings.FurthestTo(B.Locs.MyStart)
		if farBuilding != nil {
			B.DefensiveRange = farBuilding.Dist(B.Locs.MyStart) + 20
		}
	}

	// Disable greed if enemies are close (but one scout is ok)
	if (B.CcAfterRax || B.CcBeforeRax) &&
		B.Enemies.All.Filter(scl.NotWorker).CloserThan(50, B.Locs.MyStart).Len() >= 2 {
		B.CcAfterRax = false
		B.CcBeforeRax = false
	}
}
