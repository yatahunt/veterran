package bot

import (
	"github.com/aiseeq/s2l/lib/point"
	"github.com/aiseeq/s2l/lib/scl"
	"github.com/aiseeq/s2l/protocol/enums/terran"
)

const (
	Miners scl.GroupID = iota + 1
	// MinersRetreat
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
	HellionScout
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

func OnUnitCreated(unit *scl.Unit) {
	if unit.UnitType == terran.SCV {
		B.Groups.Add(Miners, unit)
		return
	}
	if unit.UnitType == terran.Marine {
		B.Groups.Add(Marines, unit)
		return
	}
	if unit.UnitType == terran.Marauder {
		B.Groups.Add(Marauders, unit)
		return
	}
	if unit.UnitType == terran.Reaper {
		B.Groups.Add(Reapers, unit)
		return
	}
	if unit.UnitType == terran.WidowMine {
		B.Groups.Add(WidowMines, unit)
		return
	}
	if unit.UnitType == terran.Hellion || unit.UnitType == terran.HellionTank {
		B.Groups.Add(Hellions, unit)
		return
	}
	if unit.UnitType == terran.Cyclone {
		B.Groups.Add(Cyclones, unit)
		return
	}
	if unit.UnitType == terran.SiegeTank || unit.UnitType == terran.SiegeTankSieged {
		B.Groups.Add(Tanks, unit)
		return
	}
	if unit.UnitType == terran.Medivac {
		B.Groups.Add(Medivacs, unit)
		return
	}
	if unit.UnitType == terran.Raven {
		B.Groups.Add(Ravens, unit)
		return
	}
	if unit.UnitType == terran.Battlecruiser {
		B.Groups.Add(Battlecruisers, unit)
		return
	}
	if unit.UnitType == terran.CommandCenter {
		// Ignore first CC, turrets poses will be found for him separately
		if B.Loop >= 12 {
			FindTurretPosition(point.Pt3(unit.Pos))
		}
		// No return! Add it to UnderConstruction group if needed
	}
	if unit.IsStructure() && unit.BuildProgress < 1 {
		B.Groups.Add(UnderConstruction, unit)
		return
	}
}
