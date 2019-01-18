package main

import (
	"bitbucket.org/aisee/sc2lib"
	"github.com/chippydip/go-sc2ai/api"
)

// todo: не балансировать минералы если ресурсов мало
// todo: что могут батлы против викингов кроме яматы?
// todo: ? иногда CC сканят дважды
// todo: минки должны выкапываться под плохими эффектами
// todo: батлы должны отступать на лечение по прямой и рейвены тоже
// todo: + надо раньше выходить на вторую базу
// todo: + минка от одного линга будет тупо отступать в сторону противоположную лингу
// todo: ? мины закапываются, но не стреляют, а сразу выкапываются
// todo: ? юниты не очень эффективно избегают штормов
// todo: ? не бояться больших групп врагов с радиусом атаки меньше, чем у юнита
// todo: + враги рядом с экспом не стёрлись и рабочий так и не пошёл достраивать CC
// todo: ? что-то придумать с самоубийственной атакой юнитов малого радиуса, когда накопились танки и мины
// todo: рабы всё ещё творят херню когда их больше, чем нужно
// todo: рабочие пытаются поставить все здания на одной точке
// todo: орбиталки без минералов не кидают мулов
// todo: use dead units events
// todo: анализировать неуспешные попытки строительства

var workerRush = false
var buildPos = map[scl.BuildingSize]scl.Points{}
var firstBarrackBuildPos = scl.Points{}
var buildTurrets = false
var turretsPos = scl.Points{}
var findTurretPositionFor *scl.Unit

const (
	Miners scl.GroupID = iota + 1
	MinersRetreat
	Builders
	Repairers
	ScvHealer
	UnitHealers
	WorkerRushDefenders
	Scout
	ScoutBase
	Marines
	Reapers
	ReapersRetreat
	Cyclones
	WidowMines
	WidowMinesRetreat
	Tanks
	Ravens
	Battlecruisers
	MechRetreat
	MechHealing
	UnderConstruction
	Buildings
	MaxGroup
)
const safeBuildRange = 7

func (b *bot) FindMainBuildingTypesPositions(startLoc scl.Point) (scl.Points, scl.Points, scl.Points) {
	var pf2x2, pf3x3, pf5x3 scl.Points
	slh := b.HeightAt(startLoc)
	start := startLoc + 9

	for y := -3.0; y <= 3; y++ {
		for x := -9.0; x <= 9; x++ {
			pos := start + scl.Pt(3, 2).Mul(x) + scl.Pt(-6, 8).Mul(y)
			if b.HeightAt(pos) == slh && b.IsPosOk(pos, scl.S3x3, 2, scl.IsBuildable) {
				if b.IsPosOk(pos+2-1i, scl.S2x2, 2, scl.IsBuildable) {
					pf5x3.Add(pos)
				} else {
					pf3x3.Add(pos)
				}
			}
			pos += 2 - 3i
			if b.HeightAt(pos) == slh && b.IsPosOk(pos, scl.S3x3, 2, scl.IsBuildable) {
				if b.IsPosOk(pos+2-1i, scl.S2x2, 2, scl.IsBuildable) {
					pf5x3.Add(pos)
				} else {
					pf3x3.Add(pos)
				}
			}
			pos += 1 - 3i
			if b.HeightAt(pos) == slh && b.IsPosOk(pos, scl.S2x2, 2, scl.IsBuildable) {
				pf2x2.Add(pos)
			}
			pos += 2
			if b.HeightAt(pos) == slh && b.IsPosOk(pos, scl.S2x2, 2, scl.IsBuildable) {
				pf2x2.Add(pos)
			}
		}
	}
	return pf2x2, pf3x3, pf5x3
}

func (b *bot) FindBuildingsPositions() {
	// Positions for first 2 supplies and barrack
	rp2x2 := b.FindRamp2x2Positions(b.MainRamp)
	firstBarrackBuildPos = b.FindRampBarracksPositions(b.MainRamp)
	for _, p := range b.GetBuildingPoints(firstBarrackBuildPos[1], scl.S5x3) { // Take second position, with addon
		b.SetBuildable(p, false)
		b.SetPathable(p, false)
	}

	// Positions for main base buildings
	pf2x2, pf3x3, pf5x3 := b.FindMainBuildingTypesPositions(b.StartLoc)

	// Mark buildings positions as non-buildable
	for size, poses := range map[scl.BuildingSize]scl.Points{
		scl.S2x2: append(rp2x2, pf2x2...),
		scl.S3x3: pf3x3,
		scl.S5x3: pf5x3,
		scl.S5x5: b.ExpLocs,
	} {
		for _, pos := range poses {
			for _, p := range b.GetBuildingPoints(pos, size) {
				b.SetBuildable(p, false)
				b.SetPathable(p, false)
			}
		}
	}

	// Find 3x3 positions behind mineral line
	mfs := b.MineralFields.Units().CloserThan(scl.ResourceSpreadDistance, b.StartLoc)
	if mfs.Exists() {
		for y := -12.0; y <= 12; y++ {
			for x := -12.0; x <= 12; x++ {
				vec := scl.Pt(x, y)
				dist := vec.Len()
				if dist <= 8 || dist >= 12 {
					continue
				}
				pos := b.StartLoc + vec
				if mfs.ClosestTo(pos).Point().Dist2(pos) >= 9 {
					continue
				}
				if !b.IsPosOk(pos, scl.S3x3, 0, scl.IsBuildable, scl.IsPathable) {
					continue
				}
				pf3x3.Add(pos)
				for _, p := range b.GetBuildingPoints(pos, scl.S3x3) {
					b.SetBuildable(p, false)
					b.SetPathable(p, false)
				}
			}
		}
	}

	pf2x2.OrderByDistanceTo(b.StartLoc, false)
	pf3x3.OrderByDistanceTo(b.StartLoc, false)
	pf5x3.OrderByDistanceTo(b.StartLoc, false)

	// Don't build fast wall against protoss, but be ready for worker rush
	if b.EnemyRace == api.Race_Protoss {
		// Insert supplies for wall after pos that is closest to base
		pos := pf2x2.FurthestTo(b.MainRamp.Top)
		pf2x2 = append(scl.Points{pos}, append(rp2x2, pf2x2...)...)
		// Use closest 5x3 position for first barracks
		firstBarrackBuildPos[0] = pf5x3.FurthestTo(b.MainRamp.Top)
	} else {
		pf2x2 = append(rp2x2, pf2x2...)
	}

	// Positions for buildings on expansions
	pf2x2a, pf3x3a, pf5x3a := b.FindMainBuildingTypesPositions(b.ExpLocs[0])
	pf2x2a.OrderByDistanceTo(b.ExpLocs[0], false)
	pf3x3a.OrderByDistanceTo(b.ExpLocs[0], false)
	pf5x3a.OrderByDistanceTo(b.ExpLocs[0], false)
	pf2x2 = append(pf2x2, pf2x2a...)
	pf3x3 = append(pf3x3, pf3x3a...)
	pf5x3 = append(pf5x3, pf5x3a...)

	buildPos[scl.S2x2] = pf2x2
	buildPos[scl.S3x3] = pf3x3
	buildPos[scl.S5x3] = pf5x3
	buildPos[scl.S5x5] = b.ExpLocs

	/*b.Debug2x2Buildings(buildPos[scl.S2x2]...)
	b.Debug3x3Buildings(buildPos[scl.S3x3]...)
	b.Debug5x3Buildings(buildPos[scl.S5x3]...)
	b.Debug3x3Buildings(buildPos[scl.S5x5]...)
	b.DebugSend()*/
}

func (b *bot) FindTurretPosition(cc *scl.Unit) {
	mfs := b.MineralFields.Units().CloserThan(scl.ResourceSpreadDistance, cc.Point())
	vesps := b.VespeneGeysers.Units().CloserThan(scl.ResourceSpreadDistance, cc.Point())
	mfs.Add(vesps...)
	if mfs.Empty() {
		return
	}

	for _, p := range b.GetBuildingPoints(cc.Point(), scl.S5x5) {
		b.SetBuildable(p, false)
	}

	var pos scl.Point
	vec := (mfs.Center() - cc.Point()).Norm()
	for x := 3.0; x < 8; x++ {
		pos = (cc.Point() + vec.Mul(x)).Floor()
		if b.IsPosOk(pos, scl.S2x2, 0, scl.IsBuildable, scl.IsNoCreep) {
			break
		}
		pos = 0
	}
	if pos == 0 {
		return
	}
	pos = pos.S2x2Fix()
	if !turretsPos.Has(pos) {
		turretsPos.Add(pos)
	}
	/*b.Debug2x2Buildings(turretsPos...)
	b.DebugSend()*/
}

func (b *bot) RecalcEnemyStartLoc(np scl.Point) {
	b.EnemyStartLoc = np
	b.FindExpansions()
	b.InitRamps()
}

func (b *bot) Logic() {
	// time.Sleep(time.Millisecond * 5)
	b.Roles()
	b.Macro()
	b.Micro()
}
