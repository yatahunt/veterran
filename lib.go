package main

import (
	"bitbucket.org/AiSee/sc2lib"
	"fmt"
	"github.com/chippydip/go-sc2ai/api"
	"os"
)

func (b *bot) chatSend(msg string) {
	b.info.SendActions([]*api.Action{{
		ActionChat: &api.ActionChat{
			Channel: api.ActionChat_Broadcast,
			Message: msg,
		},
	}})
}

func (b *bot) canBuy(ability api.AbilityID) bool {
	cost := scl.AbilityCost[ability]
	return b.minerals >= cost.Minerals && b.vespene >= cost.Vespene && (cost.Food <= 0 || b.foodLeft >= cost.Food)
}

func (b *bot) parseUnits() {
	b.units = scl.UnitsByTypes{}
	b.mineralFields = scl.UnitsByTypes{}
	b.vespeneGeysers = scl.UnitsByTypes{}
	b.neutralUnits = scl.UnitsByTypes{}
	b.enemyUnits = scl.UnitsByTypes{}
	for _, unit := range b.info.Observation().Observation.RawData.Units {
		var units *scl.UnitsByTypes
		switch unit.Alliance {
		case api.Alliance_Self:
			units = &b.units
		case api.Alliance_Enemy:
			units = &b.enemyUnits
		case api.Alliance_Neutral:
			if unit.MineralContents > 0 {
				units = &b.mineralFields
			} else if unit.VespeneContents > 0 {
				units = &b.vespeneGeysers
			} else {
				units = &b.neutralUnits
			}
		default:
			fmt.Fprintln(os.Stderr, "Not supported alliance: ", unit)
			continue
		}
		units.AddFromApi(unit.UnitType, unit)
	}
}

func (b *bot) parseOrders() {
	b.orders = map[api.AbilityID]int{}
	for _, unitTypes := range b.units {
		for _, unit := range unitTypes {
			for _, order := range unit.Orders {
				b.orders[order.AbilityId]++
			}
		}
	}
}

func (b *bot) parseObservation() {
	b.loop = int(b.obs.GameLoop)
	b.minerals = int(b.obs.PlayerCommon.Minerals)
	b.vespene = int(b.obs.PlayerCommon.Vespene)
	b.foodCap = int(b.obs.PlayerCommon.FoodCap)
	b.foodUsed = int(b.obs.PlayerCommon.FoodUsed)
	b.foodLeft = b.foodCap - b.foodUsed
}
