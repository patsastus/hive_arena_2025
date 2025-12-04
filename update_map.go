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

var Unknown_count int

const (
	UNKNOWN GameMapObjectType = iota
	OWN_BEE
	EXPLORER
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
	Revealed         map[Coords]Hex
	MyBees           map[Coords]*Hex
	MySaboteurs      map[Coords]bool
	MyHives          map[Coords]bool
	EnemyHives       map[Coords]bool
	FlowerFields     map[Coords]bool
	Mapped           map[Coords]GameMapObject
	Targeted         map[Coords]bool
	Explorers        []Coords
	StillUnexplored  bool
	EnemyBees        int
	FlowerCount      uint
	Builders         []Coords
	IsBuilding       bool
	BuildTarget      Coords
	IsBlocking       map[Coords]bool
	BlockerTargets   map[Coords]Coords //map of enemy hive coordinates to blocker target coordinates
	BlockerPositions []Coords
	TargetHive       Coords
}

func NewGameMap() GameMap {
	return GameMap{
		Revealed:         make(map[Coords]Hex),
		MyBees:           make(map[Coords]*Hex),
		MyHives:          make(map[Coords]bool),
		EnemyHives:       make(map[Coords]bool),
		FlowerFields:     make(map[Coords]bool),
		Targeted:         make(map[Coords]bool),
		Mapped:           make(map[Coords]GameMapObject),
		Explorers:        make([]Coords, 2),
		Builders:         make([]Coords, 2),
		BlockerTargets:   make(map[Coords]Coords),
		IsBlocking:       make(map[Coords]bool),
		BlockerPositions: make([]Coords, 2),
		MySaboteurs:      make(map[Coords]bool),
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
func (gm *GameMap) ExpandFringe() {

	// 1. Snapshot keys to avoid "concurrent map iteration" panic
	var currentKeys []Coords
	for c := range gm.Mapped {
		currentKeys = append(currentKeys, c)
	}

	// 2. Iterate
	for _, c := range currentKeys {
		tile := gm.Mapped[c]

		if tile.Type == UNKNOWN || tile.Type == EDGE {
			continue
		}

		for _, offset := range directionToOffset {
			neighbor := addCoords(c, offset)

			if neighbor.Row < 0 || neighbor.Col < 0 {
				continue
			}
			// 3. Check neighbor
			_, exists := gm.Mapped[neighbor]

			if !exists {
				// Add the fringe tile
				gm.Mapped[neighbor] = GameMapObject{
					Type: UNKNOWN,
				}
			}
		}
	}
}

func getDistanceToNearestHive(c Coords, gm *GameMap) int {
	shortest := 10000
	for hiveCoords := range gm.MyHives {
		d := dist(c, hiveCoords)
		if d < shortest {
			shortest = d
		}
	}
	return shortest
}

func (gm *GameMap) updateExploringStatus() {
	fmt.Println("MY BEE COUNT: ", len(gm.MyBees))
	fmt.Println("EXPLORING: ", exploring)
	// Set exploring status based on number of unknown tiles
	if exploring {
		Unknown_count = 0
		for _, tile := range gm.Mapped {
			if tile.Type == UNKNOWN {
				Unknown_count++
			}
		}
		if Unknown_count == 0 {
			exploring = false
		}
		fmt.Println("UNKNOWN COUNT: ", Unknown_count)
	}
	// Assign an explorer role to the bee furthest from a hive not carrying a flower if there are more than 2 bees
}

func (gm *GameMap) scanForEdges(viewer Coords, state *GameState) {
	for _, offset := range directionToOffset {
		currentPos := viewer
		for i := 0; i < 4; i++ {
			nextPos := addCoords(currentPos, offset)
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
	clear(gm.MyBees)   //remove all old bees from map
	gm.EnemyBees = 0   //forget old bees
	clear(gm.Targeted) //remove all targeted tiles from last turn
	for coords, visibleHex := range state.Hexes {
		gm.Revealed[coords] = *visibleHex
		index := 0
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
				gm.MyBees[coords] = visibleHex
				index++
				tile.Type = OWN_BEE
				tile.Player = unit.Player
				tile.BeeHasFlower = unit.HasFlower
			} else {
				gm.EnemyBees++
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
			gm.FlowerCount += tile.Flowers
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
	gm.FlowerCount = 0
	for coords, isField := range gm.FlowerFields {
		if isField {
			gm.FlowerCount += gm.Mapped[coords].Flowers
		}
	}
}

func ClearScreen() {
	// \033[2J  : Clear the entire screen
	// \033[H   : Move cursor to the top-left (Home) position
	fmt.Print("\033[2J\033[H")
}

func (gm *GameMap) DumpToFile(filename string) error {
	ClearScreen()
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
				case EXPLORER:
					symbol = "O "
				default:
					symbol = "? "
				}
			}
			fmt.Print(symbol)
			fmt.Fprint(f, symbol)
		}
		fmt.Print("\n")
		fmt.Fprint(f, "\n")
	}

	return nil
}

/*

	var bestExplorerCoords Coords
	// This tracks the best distance found across ALL bees
	longestDistanceFromHive := 0

	// Loop 1: Check every bee
	for beeCoords, bee := range gm.MyBees {
		if bee.Entity.HasFlower {
			continue
		}

		// RESET PER BEE: This tracks the closest unknown for THIS specific bee
		myDistanceToHive := 0

		// Loop 2: Check every unknown tile
		for hiveCoords, _ := range gm.MyHives {
			d := dist(hiveCoords, beeCoords)
			if d > myDistanceToHive {
				myDistanceToHive = d
			}
		}

		// Compare this bee against the current champion
		if myDistanceToHive > longestDistanceFromHive {
			longestDistanceFromHive = myDistanceToHive
			bestExplorerCoords = beeCoords
		}
	}

	// Apply the role to the winner
	if longestDistanceFromHive != 0 {
		explorerTile := gm.Mapped[bestExplorerCoords]
		// Assuming you added EXPLORER to your enum or want to overwrite Type
		explorerTile.Type = EXPLORER
		// Don't forget to save it back!
		gm.Mapped[bestExplorerCoords] = explorerTile
		fmt.Printf("Bee at %v assigned EXPLORER role (Dist: %d)\n", bestExplorerCoords, longestDistanceFromHive)
	}
*/
