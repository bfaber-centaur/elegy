package main

import (
	"fmt"
	"strings"

	"github.com/bfaber-centaur/elegy/engine"
)

// Display names for the engine's enumerations, for show and order. They
// are Elegy's own words, not the original's message texts.

// eventNames names every engine event kind (TestEventNames checks the
// list against the engine's constants).
var eventNames = map[engine.EventKind]string{
	engine.EventBuilt:                   "built",
	engine.EventQueueCompleted:          "queue completed",
	engine.EventOrderClipped:            "order clipped",
	engine.EventAlchemy:                 "alchemy",
	engine.EventFleetArrived:            "fleet arrived",
	engine.EventOutOfFuel:               "out of fuel",
	engine.EventRamScoopFuel:            "ram scoop fuel",
	engine.EventBattle:                  "battle",
	engine.EventFleetsMissedBattle:      "fleets missed battle",
	engine.EventBattleTech:              "battle tech",
	engine.EventBombed:                  "bombed",
	engine.EventPlanetEmptied:           "planet emptied",
	engine.EventColonized:               "colonized",
	engine.EventDropRepelled:            "drop repelled",
	engine.EventDropRefused:             "drop refused",
	engine.EventColonizeFailed:          "colonize failed",
	engine.EventMergeRefused:            "merge refused",
	engine.EventColonistsLostInFlight:   "colonists lost in flight",
	engine.EventCometStrike:             "comet strike",
	engine.EventClimateChange:           "climate change",
	engine.EventNewMinerals:             "new minerals",
	engine.EventPlayerDied:              "player died",
	engine.EventGameWon:                 "game won",
	engine.EventGameLost:                "game lost",
	engine.EventCargoGiftLost:           "cargo gift lost",
	engine.EventColonistsLostGiven:      "colonists lost given",
	engine.EventShipsBuilt:              "ships built",
	engine.EventFleetRouted:             "fleet routed",
	engine.EventFleetNotRouted:          "fleet not routed",
	engine.EventShipsJoinedFleet:        "ships joined fleet",
	engine.EventShipsLostFleetLimit:     "ships lost fleet limit",
	engine.EventPlansLost:               "plans lost",
	engine.EventStarbaseBuilt:           "starbase built",
	engine.EventColonistsKilledByEngine: "colonists killed by engine",
	engine.EventOriginalDrift:           "original drift",
	engine.EventFleetGiven:              "fleet given",
	engine.EventFleetReceived:           "fleet received",
	engine.EventFleetGiftRefused:        "fleet gift refused",
	engine.EventFleetGiftNoRoom:         "fleet gift no room",
	engine.EventMineHit:                 "mine hit",
	engine.EventMineShipsLost:           "mine ships lost",
	engine.EventMineDetonation:          "mine detonation",
	engine.EventMinesLaid:               "mines laid",
	engine.EventMinesNotLaid:            "mines not laid",
	engine.EventNoDispensers:            "no dispensers",
	engine.EventMinesSwept:              "mines swept",
	engine.EventWormholeTransit:         "wormhole transit",
	engine.EventTraderAppeared:          "trader appeared",
	engine.EventTraderLeft:              "trader left",
	engine.EventTraderTrade:             "trader trade",
	engine.EventTraderRefused:           "trader refused",
	engine.EventTraderPlanetTrade:       "trader planet trade",
	engine.EventTraderGift:              "trader gift",
	engine.EventTraderNoRoom:            "trader no room",
	engine.EventPacketImpact:            "packet impact",
	engine.EventWormholeMoved:           "wormhole moved",
	engine.EventGateRefused:             "gate refused",
	engine.EventGateUnloaded:            "gate unloaded",
	engine.EventGateShipsLost:           "gate ships lost",
	engine.EventGateFleetLost:           "gate fleet lost",
	engine.EventPacketNoDriver:          "packet no driver",
	engine.EventRacePenalized:           "race penalized",
	engine.EventRaceHacked:              "race hacked",
	engine.EventPlanetImproved:          "planet improved",
	engine.EventPlanetDegraded:          "planet degraded",
	engine.EventPacketDesignSeen:        "packet design seen",
	engine.EventFleetScrapped:           "fleet scrapped",
	engine.EventScrapRecycled:           "scrap recycled",
	engine.EventTraderPartFound:         "trader part found",
	engine.EventSalvageEmptiedFirst:     "salvage emptied first",
	engine.EventLoadRefused:             "load refused",
	engine.EventLoadWaiting:             "load waiting",
	engine.EventFollowNoLeader:          "follow no leader",
	engine.EventFollowDone:              "follow done",
	engine.EventColonistsBred:           "colonists bred",
	engine.EventColonistsBredLanded:     "colonists bred landed",
	engine.EventEngineStall:             "engine stall",
	engine.EventWarp10ShipsLost:         "warp-10 ships lost",
	engine.EventWarp10FleetLost:         "warp-10 fleet lost",
	engine.EventResearchStolen:          "research stolen",
	engine.EventRemoteMineNoModules:     "remote mine no modules",
	engine.EventRemoteMineInhabited:     "remote mine inhabited",
	engine.EventRemoteMineDeepSpace:     "remote mine deep space",
	engine.EventTargetGone:              "target gone",
	engine.EventTargetAtPlanet:          "target at planet",
	engine.EventTargetOutOfView:         "target out of view",
	engine.EventArtifactFound:           "artifact found",
	engine.EventTransportNoHold:         "transport no hold",
	engine.EventPlanetNotDegraded:       "planet not degraded",
	engine.EventPlanetNotImproved:       "planet not improved",
	engine.EventBattleSummary:           "battle summary",
	engine.EventLevelUp:                 "level up",
	engine.EventPartAvailable:           "part available",
	engine.EventOvercrowded:             "overcrowded",
	engine.EventNoQueue:                 "no queue",
	engine.EventColonistsBeamedDown:     "colonists beamed down",
	engine.EventTerraformed:             "terraformed",
	engine.EventMinefieldSwept:          "minefield swept",
	engine.EventHostileShrink:           "hostile shrink",
	engine.EventClaimAdjusted:           "claim adjusted",
	engine.EventPlanetCaptured:          "planet captured",
	engine.EventFleetLoadRefused:        "load from a foreign fleet refused",
	engine.EventColonistsToForeignFleet: "colonists refused by a foreign fleet",
}

// nameAt is names[i], or "" when i is not an index of names.
func nameAt(names []string, i int) string {
	for j, n := range names {
		if j == i {
			return n
		}
	}
	return ""
}

func eventName(k engine.EventKind) string {
	if n, ok := eventNames[k]; ok {
		return n
	}
	return fmt.Sprintf("event %d", k)
}

// fieldNames are the research fields, in field order.
var fieldNames = [engine.NumFields]string{"energy", "weapons", "propulsion", "construction", "electronics", "biotechnology"}

func fieldName(f int) string {
	if n := nameAt(fieldNames[:], f); n != "" {
		return n
	}
	return fmt.Sprintf("field %d", f)
}

// nextName is a ResearchState.Next or ResearchOrder.Next value.
func nextName(n int) string {
	switch n {
	case engine.NextSameField:
		return "same field"
	case engine.NextLowestField:
		return "lowest field"
	}
	return fieldName(n)
}

// itemNames are the production item kinds, in kind order. Ship and
// starbase items also name their design (itemText).
var itemNames = []string{
	engine.ItemMine:             "mines",
	engine.ItemFactory:          "factories",
	engine.ItemDefenses:         "defenses",
	engine.ItemMineralAlchemy:   "alchemy",
	engine.ItemAutoMines:        "auto-mines",
	engine.ItemAutoFactories:    "auto-factories",
	engine.ItemAutoDefenses:     "auto-defenses",
	engine.ItemAutoAlchemy:      "auto-alchemy",
	engine.ItemShip:             "ship",
	engine.ItemStarbase:         "starbase",
	engine.ItemIroniumPacket:    "ironium-packet",
	engine.ItemBoraniumPacket:   "boranium-packet",
	engine.ItemGermaniumPacket:  "germanium-packet",
	engine.ItemMixedPacket:      "mixed-packet",
	engine.ItemAutoPackets:      "auto-packets",
	engine.ItemTerraform:        "terraform",
	engine.ItemAutoMinTerraform: "auto-min-terraform",
	engine.ItemAutoMaxTerraform: "auto-max-terraform",
}

func itemName(k engine.ItemKind) string {
	if n := nameAt(itemNames, int(k)); n != "" {
		return n
	}
	return fmt.Sprintf("item %d", k)
}

// taskNames are the waypoint task kinds, in kind order.
var taskNames = []string{
	engine.TaskNone:          "none",
	engine.TaskTransport:     "transport",
	engine.TaskColonize:      "colonize",
	engine.TaskMerge:         "merge",
	engine.TaskRoute:         "route",
	engine.TaskPatrol:        "patrol",
	engine.TaskTransferFleet: "transfer",
	engine.TaskLayMines:      "lay-mines",
	engine.TaskRemoteMine:    "remote-mine",
	engine.TaskScrap:         "scrap",
}

func taskName(k engine.TaskKind) string {
	if n := nameAt(taskNames, int(k)); n != "" {
		return n
	}
	return fmt.Sprintf("task %d", k)
}

// lookup is the index of name in names, ignoring case.
func lookup(what, name string, names []string) (int, error) {
	for i, n := range names {
		if strings.EqualFold(n, name) {
			return i, nil
		}
	}
	return 0, fmt.Errorf("%s %q: one of %s", what, name, strings.Join(names, ", "))
}

// tacticNames are the battle-plan tactics, in tactic order.
var tacticNames = []string{
	engine.TacticDisengage:             "disengage",
	engine.TacticDisengageIfChallenged: "disengage-if-challenged",
	engine.TacticMinimizeDamage:        "minimize-damage",
	engine.TacticMaximizeNet:           "maximize-net",
	engine.TacticMaximizeRatio:         "maximize-ratio",
	engine.TacticMaximizeDamage:        "maximize-damage",
}

// targetNames are the battle-plan target types, in type order.
var targetNames = []string{
	engine.TargetNone:              "none",
	engine.TargetAny:               "any",
	engine.TargetStarbase:          "starbase",
	engine.TargetArmed:             "armed",
	engine.TargetBombersFreighters: "bombers-freighters",
	engine.TargetUnarmed:           "unarmed",
	engine.TargetFuelTransports:    "fuel-transports",
	engine.TargetFreighters:        "freighters",
}

// attackNames are the battle-plan attack choices, in order.
var attackNames = []string{
	engine.AttackNobody:             "nobody",
	engine.AttackEnemies:            "enemies",
	engine.AttackNeutralsAndEnemies: "neutrals-and-enemies",
	engine.AttackEveryone:           "everyone",
	engine.AttackPlayer:             "player",
}
