package main

import (
	"fmt"
	. "hive-arena/common"
)

type GameMapObjectType int

const (
	UNKNOWN GameMapObjectType = iota
	OWN_BEE
	ENEMY_BEE
	OWN_HIVE
	ENEMY_HIVE
	EMPTY_HEX
	EDGE
)

type GameMapObject struct {
	IsEdge bool
	Type   GameMapObjectType
}

type GameMap struct {
	Revealed map[Coords]Hex
	MyHives  map[Coords]bool
	Mapped   map[Coords]GameMapObject
}

func NewGameMap() *GameMap {
	return &GameMap{
		Revealed: make(map[Coords]Hex),
		MyHives:  make(map[Coords]bool),
		Mapped:   make(map[Coords]GameMapObject),
	}
}

func (gm *GameMap) updateGameMap(state *GameState, player int) {
	for coords, visibleHex := range state.Hexes {
		gm.Revealed[coords] = *visibleHex

		unit := visibleHex.Entity
		tile := gm.Mapped[coords]
		if unit != nil && unit.Type == HIVE {
			if unit.Player == player {
				gm.MyHives[coords] = true
				tile.Type = OWN_HIVE
			} else {
				tile.Type = ENEMY_HIVE
			}
		} else if unit != nil && unit.Type == BEE {
			if unit.Player == player {
				tile.Type = OWN_BEE
			} else {
				tile.Type = ENEMY_BEE
			}
		} else {
			tile.Type = EMPTY_HEX
		}
		gm.Mapped[coords] = tile
	}
}
