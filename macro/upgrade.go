package macro

import (
	"github.com/aiseeq/s2l/lib/scl"
	"github.com/aiseeq/s2l/protocol/api"
	"github.com/aiseeq/s2l/protocol/enums/ability"
	"github.com/aiseeq/s2l/protocol/enums/terran"
)

func OrderUpgrades() {
	lab := B.Units.My[terran.BarracksTechLab].First(scl.Ready, scl.Idle)
	if lab != nil {
		if !B.Upgrades[ability.Research_ConcussiveShells] && B.PendingAliases(ability.Train_Marauder) >= 3 &&
			lab.HasIrrAbility(ability.Research_ConcussiveShells) && B.CanBuy(ability.Research_ConcussiveShells) {
			lab.Command(ability.Research_ConcussiveShells)
			return
		}
		if !B.Upgrades[ability.Research_CombatShield] && B.Units.My[terran.Marine].Len() >= 6 &&
			lab.HasIrrAbility(ability.Research_CombatShield) && B.CanBuy(ability.Research_CombatShield) {
			lab.Command(ability.Research_CombatShield)
			return
		}
		if (B.Upgrades[ability.Research_ConcussiveShells] || B.PendingAliases(ability.Research_ConcussiveShells) > 0 ||
			B.Upgrades[ability.Research_CombatShield] || B.PendingAliases(ability.Research_CombatShield) > 0) &&
			!B.Upgrades[ability.Research_Stimpack] && lab.HasIrrAbility(ability.Research_Stimpack) &&
			B.CanBuy(ability.Research_Stimpack) {
			lab.Command(ability.Research_Stimpack)
			return
		}
	}

	eng := B.Units.My[terran.EngineeringBay].First(scl.Ready, scl.Idle)
	if eng != nil {
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
				if eng.HasIrrAbility(a) {
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
		/*if !B.Upgrades[ability.Research_HiSecAutoTracking] && B.Units.AllEnemy[terran.Banshee].Exists() &&
			eng.HasIrrAbility(ability.Research_HiSecAutoTracking) && B.CanBuy(ability.Research_HiSecAutoTracking) {
			eng.Command(ability.Research_HiSecAutoTracking)
			return
		}*/
	}

	if arm := B.Units.My[terran.Armory].First(scl.Ready, scl.Idle); arm != nil && B.Units.My.OfType(terran.WidowMine,
		terran.WidowMineBurrowed, terran.Hellion, terran.HellionTank, terran.Cyclone, terran.SiegeTank,
		terran.SiegeTankSieged, terran.Raven, terran.Battlecruiser, terran.Thor, terran.Banshee,
		terran.VikingFighter).Len() > 4 {
		upgrades := []api.AbilityID{
			ability.Research_TerranVehicleAndShipPlatingLevel1,
			ability.Research_TerranVehicleWeaponsLevel1,
			ability.Research_TerranShipWeaponsLevel1,
			ability.Research_TerranVehicleAndShipPlatingLevel2,
			ability.Research_TerranVehicleWeaponsLevel2,
			ability.Research_TerranShipWeaponsLevel2,
			ability.Research_TerranVehicleAndShipPlatingLevel3,
			ability.Research_TerranVehicleWeaponsLevel3,
			ability.Research_TerranShipWeaponsLevel3,
		}
		for _, a := range upgrades {
			if B.Upgrades[a] {
				continue
			}
			if arm.HasIrrAbility(a) {
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
		if B.PendingAliases(ability.Train_Cyclone) >= 3 &&
			lab.HasIrrAbility(ability.Research_CycloneLockOnDamage) &&
			B.CanBuy(ability.Research_CycloneLockOnDamage) {
			lab.Command(ability.Research_CycloneLockOnDamage)
			return
		}
		if B.PendingAliases(ability.Train_WidowMine) >= 4 && lab.HasIrrAbility(ability.Research_DrillingClaws) &&
			B.CanBuy(ability.Research_DrillingClaws) {
			lab.Command(ability.Research_DrillingClaws)
			return
		}
		if B.PendingAliases(ability.Train_Hellion) >= 4 && lab.HasIrrAbility(ability.Research_InfernalPreigniter) &&
			B.CanBuy(ability.Research_InfernalPreigniter) {
			lab.Command(ability.Research_InfernalPreigniter)
			return
		}
	}

	lab = B.Units.My[terran.StarportTechLab].First(scl.Ready, scl.Idle)
	if lab != nil && B.Units.My[terran.Banshee].Exists() && B.PendingAliases(ability.Train_Banshee) >= 3 {
		if lab.HasIrrAbility(ability.Research_BansheeHyperflightRotors) &&
			B.CanBuy(ability.Research_BansheeHyperflightRotors) {
			lab.Command(ability.Research_BansheeHyperflightRotors)
			return
		}
		if lab.HasIrrAbility(ability.Research_BansheeCloakingField) &&
			B.CanBuy(ability.Research_BansheeCloakingField) {
			lab.Command(ability.Research_BansheeCloakingField)
			return
		}
	}

	fc := B.Units.My[terran.FusionCore].First(scl.Ready, scl.Idle)
	if fc != nil && B.Pending(ability.Train_Battlecruiser) > 1 &&
		!B.Upgrades[ability.Research_BattlecruiserWeaponRefit] {
		if fc.HasIrrAbility(ability.Research_BattlecruiserWeaponRefit) &&
			B.CanBuy(ability.Research_BattlecruiserWeaponRefit) {
			fc.Command(ability.Research_BattlecruiserWeaponRefit)
			return
		}
	}
}
