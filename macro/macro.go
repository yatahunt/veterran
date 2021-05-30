package macro

import (
	"bitbucket.org/aisee/minilog"
	"bitbucket.org/aisee/veterran/bot"
	"github.com/aiseeq/s2l/lib/scl"
	"github.com/aiseeq/s2l/protocol/api"
	"github.com/aiseeq/s2l/protocol/enums/ability"
	"github.com/aiseeq/s2l/protocol/enums/effect"
	"github.com/aiseeq/s2l/protocol/enums/protoss"
	"github.com/aiseeq/s2l/protocol/enums/terran"
	"github.com/aiseeq/s2l/protocol/enums/zerg"
)

const SafeBuildRange = 7

var B *bot.Bot
var LastBuildLoop int

func Morph() {
	cc := B.Units.My[terran.CommandCenter].First(scl.Ready, scl.Idle)
	if cc != nil && B.Units.My[terran.Barracks].First(scl.Ready) != nil {
		if B.CanBuy(ability.Morph_OrbitalCommand) {
			OrderTrain(cc, ability.Morph_OrbitalCommand, nil)
		} else if B.Units.My[terran.SCV].Len() >= 16 {
			B.DeductResources(ability.Morph_OrbitalCommand)
		}
	}
	groundEnemies := B.Enemies.All.Filter(scl.Ground)
	for _, supply := range B.Units.My[terran.SupplyDepot].Filter(scl.Ready) {
		if groundEnemies.CloserThan(7, supply).Empty() {
			supply.Command(ability.Morph_SupplyDepot_Lower)
		}
	}
	for _, supply := range B.Units.My[terran.SupplyDepotLowered] {
		if groundEnemies.CloserThan(7, supply).Exists() {
			supply.Command(ability.Morph_SupplyDepot_Raise)
		}
	}
}

func Cast() {
	cc := B.Units.My[terran.OrbitalCommand].
		Filter(func(unit *scl.Unit) bool { return unit.Energy >= 50 }).
		Max(func(unit *scl.Unit) float64 { return float64(unit.Energy) })
	if cc != nil {
		// Scan
		if B.Orders[ability.Effect_Scan] == 0 && B.EffectPoints(effect.ScannerSweep).Empty() {
			allEnemies := B.Enemies.All
			visibleEnemies := allEnemies.Filter(scl.PosVisible)
			units := B.Units.My.All()
			// Reveal hidden units that can be attacked
			hiddenEnemies := allEnemies.Filter(scl.Hidden, scl.PosVisible)
			if hiddenEnemies.Exists() {
				army := units.Filter(scl.DpsGt5)
				for _, he := range hiddenEnemies {
					if army.CanAttack(he, 0).Exists() {
						cc.CommandPos(ability.Effect_Scan, he)
						log.Debug("Hidden enemy scan")
					}
				}
			}

			// Reaper wants to see highground
			/*if B.Units.My[terran.Raven].Empty() {
				if reaper := B.Groups.Get(bot.Reapers).Units.ClosestTo(B.Locs.EnemyStart); reaper != nil {
					if enemy := allEnemies.CanAttack(reaper, 1).ClosestTo(reaper); enemy != nil {
						if !B.IsVisible(enemy) && B.Grid.HeightAt(enemy) > B.Grid.HeightAt(reaper) {
							pos := enemy.Towards(B.Locs.EnemyStart, 8)
							cc.CommandPos(ability.Effect_Scan, pos)
							log.Debug("Reaper sight scan")
							return
						}
					}
				}
			}*/

			// Vision for tanks
			tanks := B.Units.My[terran.SiegeTankSieged]
			tanks.OrderByDistanceTo(B.Locs.EnemyStart, false)
			for _, tank := range tanks {
				targets := allEnemies.InRangeOf(tank, 0)
				if targets.Exists() && visibleEnemies.InRangeOf(tank, 0).Empty() {
					target := targets.ClosestTo(B.Locs.EnemyStart)
					cc.CommandPos(ability.Effect_Scan, target)
					log.Debug("Tank sight scan")
				}
			}

			// Lurkers
			if eps := B.EffectPoints(effect.LurkerSpines); eps.Exists() {
				// todo: check if bot already sees the lurker using his position approximation
				cc.CommandPos(ability.Effect_Scan, eps.ClosestTo(B.Locs.EnemyStart))
				log.Debug("Lurker scan")
				return
			}

			// Recon scan at 4:30
			pos := B.Locs.EnemyMainCenter
			if B.EnemyRace == api.Race_Zerg {
				pos = B.Locs.EnemyStart
			}
			if B.Loop >= scl.TimeToLoop(4, 30) && !B.Grid.IsExplored(pos) {
				cc.CommandPos(ability.Effect_Scan, pos)
				log.Debug("Recon scan")
				return
			}
		}
		// Mule on 50 energy until 4:00, until 7:00 if vs zerg
		if cc.Energy >= 75 || ((B.Loop < 5376 || B.Loop < 9408 && B.EnemyRace == api.Race_Zerg) && cc.Energy >= 50) {
			ccs := B.Units.My.OfType(terran.CommandCenter, terran.OrbitalCommand,
				terran.PlanetaryFortress).Filter(scl.Ready)
			ccs.OrderByDistanceTo(cc, false)
			for _, target := range ccs {
				homeMineral := B.Units.Minerals.All().CloserThan(scl.ResourceSpreadDistance, target).
					Filter(func(unit *scl.Unit) bool { return unit.MineralContents > 400 }).
					Max(func(unit *scl.Unit) float64 { return float64(unit.MineralContents) })
				if homeMineral != nil {
					// cc.CommandTag(ability.Effect_CalldownMULE, homeMineral.Tag)
					cc.CommandPos(ability.Effect_CalldownMULE, homeMineral.Towards(target, 1))
				}
			}
		}
	}
}

func ReserveSCVs() {
	// Fast first supply
	if B.Units.My.OfType(B.U.UnitAliases.For(terran.SupplyDepot)...).Empty() &&
		B.Groups.Get(bot.ScvReserve).Tags.Empty() {
		pos := B.BuildPos[scl.S2x2][0]
		scv := bot.GetSCV(pos, 0, 45) // Get SCV but don't change its group
		if scv != nil && scv.FramesToPos(pos)*B.MineralsPerFrame+float64(B.Minerals) >= 100 {
			B.Groups.Add(bot.ScvReserve, scv)
			scv.CommandPos(ability.Move, pos)
		}
	}
	// Fast expansion
	if B.Units.My.OfType(B.U.UnitAliases.For(terran.CommandCenter)...).Len() == 1 &&
		B.Minerals >= 350 && B.Groups.Get(bot.ScvReserve).Tags.Empty() && !B.WorkerRush {
		pos := B.Locs.MyExps[0]
		if scv := bot.GetSCV(pos, bot.ScvReserve, 45); scv != nil {
			scv.CommandPos(ability.Move, pos)
		}
	}
}

func Macro(b *bot.Bot) {
	B = b // todo: better

	if !B.BuildTurrets && B.Units.Enemy.OfType(terran.Banshee, terran.Ghost, terran.WidowMine, terran.Medivac,
		terran.VikingFighter, terran.Liberator, terran.Battlecruiser, terran.Starport, zerg.Mutalisk, zerg.LurkerMP,
		zerg.Corruptor, zerg.Spire, zerg.GreaterSpire, protoss.DarkTemplar, protoss.WarpPrism, protoss.Phoenix,
		protoss.VoidRay, protoss.Oracle, protoss.Tempest, protoss.Carrier, protoss.Stargate, protoss.DarkShrine).
		Exists() {
		B.BuildTurrets = true
	}

	if LastBuildLoop+B.FramesPerOrder < B.Loop {
		if B.Loop >= 5376 { // 4:00
			OrderUpgrades()
		}
		ProcessBuildOrder(RootBuildOrder)
		Morph()
		OrderUnits()
		ReserveSCVs()
		LastBuildLoop = B.Loop
	}
	Cast()
}
