package main

import (
	"fmt"
	. "hive-arena/common"
	"os"
)

type GameMapObjectType int

var directionToOffset = map[Direction]Coords{
	E:  {0, 2},
	NE: {-1, 1},
	NW: {-1, -1},
	W:  {0, -2},
	SW: {1, -1},
	SE: {1, 1},
}

const (
	UNKNOWN GameMapObjectType = iota
	OWN_BEE
	ENEMY_BEE
	OWN_HIVE
	ENEMY_HIVE
	OWN_WALL
	ENEMY_WALL
	EMPTY_HEX
	WALL_HEX
	ROCK_HEX
	EDGE
)

type GameMapObject struct {
	BeeHasFlower  bool
	Flowers       uint
	IsFlowerField bool
	IsWalkable    bool
	Player        int
	Type          GameMapObjectType
}

type GameMap struct {
	Revealed     map[Coords]Hex
	MyHives      map[Coords]bool
	EnemyHives   map[Coords]bool
	FlowerFields map[Coords]bool
	Mapped       map[Coords]GameMapObject
}

func NewGameMap() GameMap {
	return GameMap{
		Revealed:     make(map[Coords]Hex),
		MyHives:      make(map[Coords]bool),
		EnemyHives:   make(map[Coords]bool),
		FlowerFields: make(map[Coords]bool),
		Mapped:       make(map[Coords]GameMapObject),
	}
}

func addCoords(pos, offset Coords) Coords {
	return Coords{
		Row: pos.Row + offset.Row,
		Col: pos.Col + offset.Col,
	}
}

func (gm *GameMap) MarkAsEdge(c Coords) {
	if tile, ok := gm.Mapped[c]; ok {
		if tile.Type != EDGE {
			tile.Type = EDGE
			tile.IsWalkable = false
			tile.IsFlowerField = false
			tile.BeeHasFlower = false
			tile.Flowers = 0
			gm.Mapped[c] = tile
			fmt.Printf("Edge found at %v\n", c)
		}
	}
}

func (gm *GameMap) scanForEdges(viewer Coords, state *GameState) {
	for _, offest := range directionToOffset {
		currentPos := viewer
		for i := 0; i < 4; i++ {
			nextPos := addCoords(currentPos, offest)
			_, nextExists := state.Hexes[nextPos]
			if !nextExists {
				if i < 3 {
					tile := gm.Mapped[nextPos]
					if tile.Type != EDGE {
						tile.Type = EDGE
						gm.Mapped[nextPos] = tile
					}
				}
				break
				// gm.MarkAsEdge(currentPos)
			}
			currentPos = nextPos
		}
	}
}

func (gm *GameMap) updateGameMap(state *GameState, player int) {
	for coords, visibleHex := range state.Hexes {
		gm.Revealed[coords] = *visibleHex

		tile := gm.Mapped[coords]
		tile.Type = UNKNOWN // Default to unknown before classification
		tile.BeeHasFlower = false
		tile.IsFlowerField = false
		tile.Flowers = 0
		wasEdge := (tile.Type == EDGE)
		unit := visibleHex.Entity
		if unit != nil && unit.Type == HIVE {
			if unit.Player == player {
				gm.MyHives[coords] = true
				tile.Type = OWN_HIVE
				tile.Player = unit.Player
			} else {
				tile.Type = ENEMY_HIVE
				gm.EnemyHives[coords] = true
				tile.Player = unit.Player
			}
		} else if unit != nil && unit.Type == BEE {
			if unit.Player == player {
				tile.Type = OWN_BEE
				tile.Player = unit.Player
				tile.BeeHasFlower = unit.HasFlower
			} else {
				tile.Type = ENEMY_BEE
				tile.Player = unit.Player
				tile.BeeHasFlower = unit.HasFlower
			}
		} else if unit != nil && unit.Type == WALL {
			if unit.Player == player {
				tile.Type = OWN_WALL
				tile.Player = unit.Player
			} else {
				tile.Type = ENEMY_WALL
				tile.Player = unit.Player
			}
		} else {
			if visibleHex.Terrain == "ROCK" {
				tile.Type = ROCK_HEX
			} else {
				if wasEdge {
					tile.Type = EDGE
				} else {
					tile.Type = EMPTY_HEX
				}
			}
		}

		if visibleHex.Resources > 0 {
			tile.IsFlowerField = true
			tile.Flowers = visibleHex.Resources
			gm.FlowerFields[coords] = true
		} else {
			tile.IsFlowerField = false
			tile.Flowers = 0
			gm.FlowerFields[coords] = false
		}
		tile.IsWalkable = visibleHex.Terrain.IsWalkable()
		gm.Mapped[coords] = tile
	}
	for coords, visibleHex := range state.Hexes {
		unit := visibleHex.Entity
		if unit != nil && unit.Player == player {
			gm.scanForEdges(coords, state)
		}
	}
}
func (gm *GameMap) DumpToFile(filename string) error {

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	// 1. Calculate Bounds
	minR, maxR := 1000, -1000
	minC, maxC := 1000, -1000

	for c := range gm.Mapped {
		if c.Row < minR {
			minR = c.Row
		}
		if c.Row > maxR {
			maxR = c.Row
		}
		if c.Col < minC {
			minC = c.Col
		}
		if c.Col > maxC {
			maxC = c.Col
		}
	}

	// fmt.Fprintf(f, "Map Bounds: [%d, %d] to [%d, %d]\n", minR, minC, maxR, maxC)

	// 2. Iterate Rows
	for r := minR; r <= maxR; r++ {

		// Row Label
		// fmt.Fprintf(f, "[%2d] ", r)

		// 3. Iterate Columns
		for c := minC; c <= maxC; c++ {

			tile, exists := gm.Mapped[Coords{Row: r, Col: c}]

			// DEFAULT: Two standard spaces (ASCII 32)
			symbol := "  "

			if exists {
				switch tile.Type {
				case EDGE:
					symbol = "# "
				case OWN_BEE:
					symbol = "B "
				case ENEMY_BEE:
					symbol = "E "
				case OWN_HIVE:
					symbol = "H "
				case ENEMY_HIVE:
					symbol = "X "
				case ROCK_HEX:
					symbol = "R "
				case EMPTY_HEX:
					if tile.IsFlowerField {
						symbol = "F "
					} else {
						symbol = ". "
					}
				case UNKNOWN:
					symbol = "? "
				default:
					symbol = "? "
				}
			}
			fmt.Fprint(f, symbol)
		}
		fmt.Fprint(f, "\n")
	}
	return nil
}
