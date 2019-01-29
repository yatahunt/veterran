package micro

import (
	"bitbucket.org/aisee/sc2lib"
	"github.com/cenkalti/log"
	"github.com/chippydip/go-sc2ai/enums/protoss"
	"github.com/chippydip/go-sc2ai/enums/terran"
	"github.com/chippydip/go-sc2ai/enums/zerg"
)

var Targets struct {
	All         scl.Units
	Flying      scl.Units
	Armed       scl.Units
	ArmedFlying scl.Units
}

func InitTargets() {
	for _, u := range B.Units.AllEnemy.All() {
		if u.Is(zerg.Larva, zerg.Egg, protoss.AdeptPhaseShift, terran.KD8Charge) {
			continue
		}
		Targets.All.Add(u)
		if u.IsArmed() {
			Targets.Armed.Add(u)
		}
		if u.IsFlying {
			Targets.Flying.Add(u)
			if u.IsArmed() {
				Targets.ArmedFlying.Add(u)
			}
		}
	}
	log.Info(B.Locs.EnemyExps)
}

func Do() {
	InitTargets()
}
