package main

import (
	"bitbucket.org/aisee/sc2lib"
	"github.com/chippydip/go-sc2ai/enums/ability"
	"github.com/chippydip/go-sc2ai/enums/protoss"
	"github.com/chippydip/go-sc2ai/enums/terran"
	"github.com/chippydip/go-sc2ai/enums/zerg"
)

// todo: протестировать, может MinersRetreat вообще не нужно
// todo: торы против зергов вместо батлов?
// todo: викинги против баньши? Или просто добавить туррелей?
// todo: убрать лишний скан после того как снаряды от убитой баньши долетают до цели
// todo: юниты в углах карты могут отвлекать минки
// todo: ? циклоны перестали стрелять отступая от лингов
// todo: ? рабы всё ещё творят херню (когда их больше, чем нужно?) - видимо, связано с починкой и недостатком ревурсов
// todo: ? рабочие пытаются поставить все здания на одной точке -> возможно, нужно строить на %3 кадрах (ошибки отсутствия ресурсов?)
// todo: нужно что-то придумать с SCV и miners под атакой. Сейчас они реагируют слишком сильно и пугливо
// todo: минки боятся рабочих, забегают в угол и тупят -> отслеживать время взрыва и закапывать если по пути к лечению
// todo: хрень с хайграундом на автоматоне, юниты идут не туда и дохнут
// todo: надо как-то определять какие здания не стоит чинить, т.к. рабочий будет убит (по числу ranged?)
// todo: танку надо перераскладываться, если на границе его радиуса только здания
// todo: строить первый CC на хайграунде если опасно?
// todo: детектить однобазовый оллин и переходить на вторую только после определённого лимита
// todo: поднимать и спасать CC, но забить на починку рефов, если рядом враги
// todo: если есть апгрейд для минок, закапывать их, если за ними гонится кто-то быстрее их
// todo: детект спидлингов + крип
// todo: use dead units events
// todo: анализировать неуспешные попытки строительства, зарытые линги мешают поставить СС -> ставить башню рядом?

var isRealtime = false
var workerRush = false
var buildTurrets = false
var playDefensive = true
var defensiveRange = 0.0
var buildPos = map[scl.BuildingSize]scl.Points{}
var firstBarrackBuildPos = scl.Points{}
var turretsPos = scl.Points{}
var bunkersPos = scl.Points{}
var findTurretPositionFor *scl.Unit
var lastBuildLoop = 0
var doubleHealers []scl.GroupID

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
	ScvReserve
	Marines
	Marauders
	Reapers
	ReapersRetreat
	Cyclones
	WidowMines
	WidowMinesRetreat
	Hellions
	Tanks
	TanksOnExps
	Medivacs
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
				if mfs.ClosestTo(pos).Dist2(pos) >= 9 {
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
	// I'll try to defend it other way
	/*if b.EnemyRace == api.Race_Protoss {
		// Insert supplies for wall after pos that is closest to base
		pos := pf2x2.FurthestTo(b.MainRamp.Top)
		pf2x2 = append(scl.Points{pos}, append(rp2x2, pf2x2...)...)
		// Use closest 5x3 position for first barracks
		firstBarrackBuildPos[0] = pf5x3.FurthestTo(b.MainRamp.Top)
	} else {*/
	pf2x2 = append(rp2x2, pf2x2...)
	//}

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
	mfs := b.MineralFields.Units().CloserThan(scl.ResourceSpreadDistance, cc)
	vesps := b.VespeneGeysers.Units().CloserThan(scl.ResourceSpreadDistance, cc)
	mfs.Add(vesps...)
	if mfs.Empty() {
		return
	}

	for _, p := range b.GetBuildingPoints(cc, scl.S5x5) {
		b.SetBuildable(p, false)
	}

	var pos scl.Point
	vec := (mfs.Center() - cc.Point()).Norm()
	for x := 3.0; x < 8; x++ {
		pos = (cc.Point() + vec.Mul(x)).Floor()
		if b.IsPosOk(pos, scl.S2x2, 0, scl.IsBuildable, scl.IsPathable, scl.IsNoCreep) {
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

func (b *bot) FindBunkerPosition(ptr scl.Pointer) {
	ccPos := ptr.Point().Floor()
	mfs := b.MineralFields.Units().CloserThan(scl.ResourceSpreadDistance, ccPos)
	vesps := b.VespeneGeysers.Units().CloserThan(scl.ResourceSpreadDistance, ccPos)
	mfs.Add(vesps...)
	if mfs.Empty() {
		return
	}

	for _, p := range b.GetBuildingPoints(ccPos, scl.S5x5) {
		b.SetBuildable(p, false)
	}

	var pos scl.Point
	vec := (mfs.Center() - ccPos).Norm()
	for x := 3.0; x < 8; x++ {
		pos = (ccPos - vec.Mul(x)).Floor()
		if bunkersPos.Has(pos) {
			return // There is already position for one bunker
		}
		if b.IsPosOk(pos, scl.S3x3, 0, scl.IsBuildable, scl.IsPathable, scl.IsNoCreep) {
			bunkersPos.Add(pos)
			break
		}
	}
	/*b.Debug3x3Buildings(bunkersPos...)
	b.DebugSend()*/
}

func (b *bot) getEmptyBunker(ptr scl.Pointer) *scl.Unit {
	bunkers := b.Units[terran.Bunker].Filter(func(unit *scl.Unit) bool {
		return unit.CargoSpaceTaken < unit.CargoSpaceMax
	})
	return bunkers.Min(func(unit *scl.Unit) float64 { return unit.Dist2(ptr) })
}

func (b *bot) RecalcEnemyStartLoc(np scl.Point) {
	b.EnemyStartLoc = np
	b.EnemyMainCenter = b.FindBaseCenter(b.EnemyStartLoc)
	b.FindExpansions()
	b.InitRamps()
}

func (b *bot) EnableDefensivePlay() {
	playDefensive = true
	for _, cc := range b.Units.OfType(terran.CommandCenter, terran.OrbitalCommand, terran.PlanetaryFortress) {
		if cc.IsCloserThan(1, b.StartLoc) {
			continue
		}
		b.FindBunkerPosition(cc)
	}
}

func (b *bot) DefensivePlayCheck() {
	armyScore := b.Units.Units().Filter(scl.DpsGt5).Sum(scl.CmpGroundScore)
	enemyScore := b.AllEnemyUnits.Units().Filter(scl.DpsGt5).Sum(scl.CmpGroundScore)
	if armyScore > enemyScore * 1.5 && b.Obs.Score.ScoreDetails.FoodUsed.Army >= 75 {
		playDefensive = false
		bunkersPos = nil
		if bunkers := b.Units[terran.Bunker]; bunkers.Exists() {
			bunkers.Command(ability.UnloadAll_Bunker)
			bunkers.CommandQueue(ability.Effect_Salvage)
		}
		if tanks := b.Groups.Get(TanksOnExps).Units; tanks.Exists() {
			b.Groups.Add(Tanks, tanks...)
		}
	}
	if armyScore > enemyScore * 1.5 && b.Obs.Score.ScoreDetails.FoodUsed.Army >= 50 {
		playDefensive = false
	}
	if armyScore * 1.5 < enemyScore {
		b.EnableDefensivePlay()
	}
	if b.Loop >= 3584 && b.Loop < 3594 { // 2:40
		townHalls := append(scl.UnitAliases.For(terran.CommandCenter), scl.UnitAliases.For(zerg.Hatchery)...)
		townHalls = append(townHalls, protoss.Nexus)
		if b.AllEnemyUnits.OfType(townHalls...).Len() < 2 {
			b.EnableDefensivePlay()
		}
	}
	/*if b.Loop < 4480 && b.AllEnemyUnits[protoss.Stalker].Len() > 2 { // 3:20
		b.EnableDefensivePlay()
	}*/
	if playDefensive {
		buildings := append(b.Groups.Get(Buildings).Units, b.Groups.Get(UnderConstruction).Units...)
		farBuilding := buildings.FurthestTo(b.StartLoc)
		if farBuilding != nil {
			defensiveRange = farBuilding.Dist(b.StartLoc) + 20
		}
	}
}

func (b *bot) Logic() {
	// time.Sleep(time.Millisecond * 10)
	b.DefensivePlayCheck()
	b.Roles()
	b.Macro()
	b.Micro()
}
