package main

import (
	"bitbucket.org/aisee/sc2lib"
)

// todo: строить первый барак с пересадкой и сразу после постройки саплая тем же рабочим
// todo: всё ещё есть проблемы (дёрп) с назначением на продолжение строительства если рабочего убили
// todo: use dead units events
// todo: wall closed flag -> no worker defence
// todo: fix morph abilities cost

var workerRush = false
var assault = false
var buildPos = map[scl.BuildingSize]scl.Points{}
var firstBarrackBuildPos = scl.Points{}

const (
	Miners scl.GroupID = iota + 1
	MinersRetreat
	Builders
	Repairers
	ScvHealer
	WorkerRushDefenders
	Scout
	Reapers
	Retreat
	UnderConstruction
	Buildings
	MaxGroup
)
const safeBuildRange = 7

func (b *bot) FindBuildingsPositions() {
	homeMinerals := b.MineralFields.Units().CloserThan(scl.ResourceSpreadDistance, b.StartLoc)
	if homeMinerals.Len() == 0 {
		return // This should not happen
	}
	vec := homeMinerals.Center().Dir(b.StartLoc)
	if vec.Len() == 1 {
		vec = b.StartLoc.Dir(b.MapCenter)
	}

	rp2x2 := b.FindRamp2x2Positions(b.MainRamp)
	firstBarrackBuildPos = b.FindRampBarracksPositions(b.MainRamp)
	rbpts := b.GetBuildingPoints(firstBarrackBuildPos[1], scl.S5x3) // Take second position, with addon

	/*pos = b.EnemyStartLoc.Towards(b.StartLoc, 25)
	pos = pos.Closest(b.ExpLocs).Towards(b.StartLoc, 1)

	pfb := []*api.RequestQueryBuildingPlacement{{
		AbilityId: ability.Build_Barracks,
		TargetPos: pos.To2D()}}
	for _, np := range pos.Neighbours8(4) {
		if b.IsBuildable(np) {
			pfb = append(pfb, &api.RequestQueryBuildingPlacement{
				AbilityId: ability.Build_Barracks,
				TargetPos: np.To2D()})
		}
	}
	resp := b.Info.Query(api.RequestQuery{Placements: pfb, IgnoreResourceRequirements: true})
	for key, result := range resp.Placements {
		if result.Result == api.ActionResult_Success {
			pos5x3.Add(scl.Pt2(pfb[key].TargetPos))
		}
	}*/

	var pf2x2, pf3x3, pf5x3 scl.Points
	slh := b.HeightAt(b.StartLoc)
	start := b.StartLoc + 9
	for y := -3.0; y <= 3; y++ {
		for x := -9.0; x <= 9; x++ {
			pos := start + scl.Pt(3, 2).Mul(x) + scl.Pt(-6, 8).Mul(y)
			if b.HeightAt(pos) == slh && b.IsPosOk(pos, scl.S3x3, 2, scl.IsBuildable) &&
				rbpts.Intersect(b.GetBuildingPoints(pos, scl.S3x3)).Empty() {
				if b.IsPosOk(pos+2-1i, scl.S2x2, 2, scl.IsBuildable) &&
					rbpts.Intersect(b.GetBuildingPoints(pos+2-1i, scl.S2x2)).Empty() {
					pf5x3.Add(pos)
				} else {
					pf3x3.Add(pos)
				}
			}
			pos += 2 - 3i
			if b.HeightAt(pos) == slh && b.IsPosOk(pos, scl.S3x3, 2, scl.IsBuildable) &&
				rbpts.Intersect(b.GetBuildingPoints(pos, scl.S3x3)).Empty() {
				if b.IsPosOk(pos+2-1i, scl.S2x2, 2, scl.IsBuildable) &&
					rbpts.Intersect(b.GetBuildingPoints(pos+2-1i, scl.S2x2)).Empty() {
					pf5x3.Add(pos)
				} else {
					pf3x3.Add(pos)
				}
			}
			pos += 1 - 3i
			if b.HeightAt(pos) == slh && b.IsPosOk(pos, scl.S2x2, 2, scl.IsBuildable) &&
				rbpts.Intersect(b.GetBuildingPoints(pos, scl.S2x2)).Empty() {
				pf2x2.Add(pos)
			}
			pos += 2
			if b.HeightAt(pos) == slh && b.IsPosOk(pos, scl.S2x2, 2, scl.IsBuildable) &&
				rbpts.Intersect(b.GetBuildingPoints(pos, scl.S2x2)).Empty() {
				pf2x2.Add(pos)
			}
		}
	}
	pf2x2.OrderByDistanceTo(b.StartLoc, false)
	pf3x3.OrderByDistanceTo(b.StartLoc, false)
	pf5x3.OrderByDistanceTo(b.StartLoc, false)

	buildPos[scl.S2x2] = append(rp2x2, pf2x2...)
	buildPos[scl.S3x3] = pf3x3
	buildPos[scl.S5x3] = pf5x3
	buildPos[scl.S5x5] = b.ExpLocs

	/*b.Debug2x2Buildings(buildPos[scl.S2x2]...)
	b.Debug3x3Buildings(buildPos[scl.S3x3]...)
	b.Debug5x3Buildings(buildPos[scl.S5x3]...)
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
