package bot

import (
	"github.com/aiseeq/s2l/lib/scl"
	"github.com/aiseeq/s2l/protocol/enums/ability"
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
	Mules
	Scout
	ScoutBase
	ScvReserve
	ProxyBuilders
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
	Thors
	Medivacs
	ThorEvacs
	Vikings
	Ravens
	Banshees
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
	if unit.UnitType == terran.MULE {
		B.Groups.Add(Mules, unit)
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
	if unit.UnitType == terran.Thor || unit.UnitType == terran.ThorAP {
		B.Groups.Add(Thors, unit)
		return
	}
	if unit.UnitType == terran.Medivac {
		B.Groups.Add(Medivacs, unit)
		if unit.HasAbility(ability.Effect_MedivacIgniteAfterburners) {
			unit.Command(ability.Effect_MedivacIgniteAfterburners)
		}
		return
	}
	if unit.UnitType == terran.VikingAssault || unit.UnitType == terran.VikingFighter {
		B.Groups.Add(Vikings, unit)
		return
	}
	if unit.UnitType == terran.Raven {
		B.Groups.Add(Ravens, unit)
		return
	}
	if unit.UnitType == terran.Banshee {
		B.Groups.Add(Banshees, unit)
		return
	}
	if unit.UnitType == terran.Battlecruiser {
		B.Groups.Add(Battlecruisers, unit)
		return
	}
	if unit.UnitType == terran.CommandCenter {
		if B.Loop >= 12 {
			// Ignore first CC
		} else {
			B.Groups.Add(Buildings, unit)
		}
		// No return! Add it to UnderConstruction group if needed
	}
	if unit.IsStructure() && unit.BuildProgress < 1 {
		B.Groups.Add(UnderConstruction, unit)
		return
	}
}
