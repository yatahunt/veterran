package macro

import (
	"bitbucket.org/aisee/sc2lib"
	"github.com/chippydip/go-sc2ai/api"
	"github.com/chippydip/go-sc2ai/enums/ability"
	"github.com/chippydip/go-sc2ai/enums/terran"
)

func OrderUpgrades() {
	lab := B.Units.My[terran.BarracksTechLab].First(scl.Ready, scl.Idle)
	if lab != nil {
		B.RequestAvailableAbilities(true, lab) // todo: request true each frame -> HasTrueAbility?
		if !B.Upgrades[ability.Research_ConcussiveShells] && B.PendingAliases(ability.Train_Marauder) >= 2 &&
			lab.HasAbility(ability.Research_ConcussiveShells) && B.CanBuy(ability.Research_ConcussiveShells) {
			lab.Command(ability.Research_ConcussiveShells)
			return
		}
		if !B.Upgrades[ability.Research_CombatShield] && B.Units.My[terran.Marine].Len() >= 4 &&
			lab.HasAbility(ability.Research_CombatShield) && B.CanBuy(ability.Research_CombatShield) {
			lab.Command(ability.Research_CombatShield)
			return
		}
		if (B.Upgrades[ability.Research_ConcussiveShells] || B.PendingAliases(ability.Research_ConcussiveShells) > 0 ||
			B.Upgrades[ability.Research_CombatShield] || B.PendingAliases(ability.Research_CombatShield) > 0) &&
			!B.Upgrades[ability.Research_Stimpack] && lab.HasAbility(ability.Research_Stimpack) &&
			B.CanBuy(ability.Research_Stimpack) {
			lab.Command(ability.Research_Stimpack)
			return
		}
	}

	eng := B.Units.My[terran.EngineeringBay].First(scl.Ready, scl.Idle)
	if eng != nil {
		B.RequestAvailableAbilities(true, eng) // request abilities again because we want to ignore resource reqs
		if B.Units.My[terran.Marine].Len()+B.Units.My[terran.Marauder].Len()*2+B.Units.My[terran.Reaper].Len()*2 >= 8 {
			for _, a := range []api.AbilityID{
				ability.Research_TerranInfantryWeaponsLevel1,
				ability.Research_TerranInfantryArmorLevel1,
				ability.Research_TerranInfantryWeaponsLevel2,
				ability.Research_TerranInfantryArmorLevel2,
				ability.Research_TerranInfantryWeaponsLevel3,
				ability.Research_TerranInfantryArmorLevel3,
			} {
				if B.Upgrades[a] {
					continue
				}
				if eng.HasAbility(a) {
					if B.CanBuy(a) {
						eng.Command(a)
						return
					} else {
						// reserve money for upgrade
						B.DeductResources(a)
					}
					break
				}
			}
		}
		if !B.Upgrades[ability.Research_HiSecAutoTracking] && B.Units.AllEnemy[terran.Banshee].Exists() &&
			eng.HasAbility(ability.Research_HiSecAutoTracking) && B.CanBuy(ability.Research_HiSecAutoTracking) {
			eng.Command(ability.Research_HiSecAutoTracking)
			return
		}
	}

	// todo: aliases
	if arm := B.Units.My[terran.Armory].First(scl.Ready, scl.Idle); arm != nil && B.Units.My.OfType(terran.WidowMine,
		terran.Hellion, terran.Cyclone, terran.SiegeTank, terran.Raven, terran.Battlecruiser).Len() > 4 {
		B.RequestAvailableAbilities(true, arm) // request abilities again because we want to ignore resource reqs
		upgrades := []api.AbilityID{
			ability.Research_TerranVehicleAndShipPlatingLevel1,
			ability.Research_TerranVehicleAndShipPlatingLevel2,
			ability.Research_TerranVehicleAndShipPlatingLevel3,
			ability.Research_TerranVehicleWeaponsLevel1,
			ability.Research_TerranVehicleWeaponsLevel2,
			ability.Research_TerranVehicleWeaponsLevel3,
		}
		if B.Units.My[terran.Battlecruiser].Exists() {
			upgrades = append([]api.AbilityID{
				ability.Research_TerranShipWeaponsLevel1,
				ability.Research_TerranShipWeaponsLevel2,
				ability.Research_TerranShipWeaponsLevel3,
			}, upgrades...)
		}
		for _, a := range upgrades {
			if B.Upgrades[a] {
				continue
			}
			if arm.HasAbility(a) {
				if B.CanBuy(a) {
					arm.Command(a)
					return
				} else {
					// reserve money for upgrade
					B.DeductResources(a)
				}
				break
			}
		}
	}

	lab = B.Units.My[terran.FactoryTechLab].First(scl.Ready, scl.Idle)
	if lab != nil && (B.Units.My[terran.Cyclone].Exists() || B.Units.My[terran.WidowMine].Exists()) {
		B.RequestAvailableAbilities(true, lab)
		if B.PendingAliases(ability.Train_Cyclone) >= 2 &&
			lab.HasAbility(ability.Research_CycloneResearchLockOnDamageUpgrade) &&
			B.CanBuy(ability.Research_CycloneResearchLockOnDamageUpgrade) {
			lab.Command(ability.Research_CycloneResearchLockOnDamageUpgrade)
			return
		}
		if B.PendingAliases(ability.Train_WidowMine) >= 2 && lab.HasAbility(ability.Research_DrillingClaws) &&
			B.CanBuy(ability.Research_DrillingClaws) {
			lab.Command(ability.Research_DrillingClaws)
			return
		}
		if B.PendingAliases(ability.Train_Hellion) >= 4 && lab.HasAbility(ability.Research_InfernalPreigniter) &&
			B.CanBuy(ability.Research_InfernalPreigniter) {
			lab.Command(ability.Research_InfernalPreigniter)
			return
		}
	}

	fc := B.Units.My[terran.FusionCore].First(scl.Ready, scl.Idle)
	if fc != nil && B.Pending(ability.Train_Battlecruiser) > 0 &&
		!B.Upgrades[ability.Research_BattlecruiserWeaponRefit] {
		B.RequestAvailableAbilities(true, fc)
		if fc.HasAbility(ability.Research_BattlecruiserWeaponRefit) &&
			B.CanBuy(ability.Research_BattlecruiserWeaponRefit) {
			fc.Command(ability.Research_BattlecruiserWeaponRefit)
			return
		}
	}
}
