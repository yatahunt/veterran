package main

import (
	"bitbucket.org/aisee/sc2lib"
	"github.com/chippydip/go-sc2ai/api"
)

// todo: check scans for mines
// todo: if banshee build turrets
// todo: no units around & taking damage => cloaked banshee
// todo: рабочие пытаются поставить все здания на одной точке
// todo: орбиталки без минералов не кидают мулов
// todo: use dead units events
// todo: wall closed flag -> no worker defence
// todo: анализировать неуспешные попытки строительства

var buildPos = map[scl.BuildingSize]scl.Points{}
var firstBarrackBuildPos = scl.Points{}
var buildTurrets = false
var turretsPos = scl.Points{}

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
	Reapers
	ReapersRetreat
	Cyclones
	Mines
	MinesRetreat
	MechRetreat
	MechHealing
	UnderConstruction
	Buildings
	MaxGroup
)
const safeBuildRange = 7

func (b *bot) FindBuildingsPositions() {
	// Positions for first 2 supplies and barrack
	rp2x2 := b.FindRamp2x2Positions(b.MainRamp)
	firstBarrackBuildPos = b.FindRampBarracksPositions(b.MainRamp)
	for _, p := range b.GetBuildingPoints(firstBarrackBuildPos[1], scl.S5x3) { // Take second position, with addon
		b.SetBuildable(p, false)
		b.SetPathable(p, false)
	}

	// Positions for main base buildings
	var pf2x2, pf3x3, pf5x3 scl.Points
	slh := b.HeightAt(b.StartLoc)
	start := b.StartLoc + 9
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
				if dist <= 64 || dist >= 144 {
					continue
				}
				pos := b.StartLoc + vec
				if mfs.ClosestTo(pos).Point().Dist2(pos) != 9 {
					continue
				}
				if !b.IsPosOk(pos, scl.S3x3, 0, scl.IsBuildable) { // scl.IsPathable?
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
		pf2x2 = append(pf2x2[:1], append(rp2x2, pf2x2...)...)
		// Use closest 5x3 position for first barracks
		firstBarrackBuildPos[0], pf5x3 = pf5x3[0], pf5x3[1:]
		buildPos[scl.S2x2] = pf2x2
	} else {
		buildPos[scl.S2x2] = append(rp2x2, pf2x2...)
	}

	buildPos[scl.S3x3] = pf3x3
	buildPos[scl.S5x3] = pf5x3
	buildPos[scl.S5x5] = b.ExpLocs

	b.Debug2x2Buildings(buildPos[scl.S2x2]...)
	b.Debug3x3Buildings(buildPos[scl.S3x3]...)
	b.Debug5x3Buildings(buildPos[scl.S5x3]...)
	b.Debug3x3Buildings(buildPos[scl.S5x5]...)
	b.DebugSend()
}

func (b *bot) FindTurretPosition(cc *scl.Unit) {
	mfs := b.MineralFields.Units().CloserThan(scl.ResourceSpreadDistance, cc.Point())
	if mfs.Empty() {
		return
	}
	vec := cc.Point().Dir(mfs.Center())
	pos := (cc.Point() + vec * 3).Floor()
	if vec == scl.Pt(-1, -1) {
		pos += vec
	}
	if !turretsPos.Has(pos) {
		turretsPos.Add(pos)
	}
	b.Debug2x2Buildings(turretsPos...)
	b.DebugSend()
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
