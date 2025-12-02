package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"

	. "hive-arena/common"
	// "github.com/go-text/typesetting/di"
)

var dirs = []Direction{E, SE, SW, W, NW, NE}
var gameMap GameMap
var exploring bool

func dist(one, two Coords) int {
	dx := one.Row - two.Row
	if dx < 0 {
		dx = -dx
	}

	dy := one.Col - two.Col
	if dy < 0 {
		dy = -dy
	}

	if dy < dx {
		return dx
	}
	return dx + (dy-dx)/2
}

func goHome(h Hex, coords Coords) Order {
	distance := 20000
	var o Order
	var target Coords
	o.Coords = coords
	o.Type = MOVE
	for key, _ := range gameMap.MyHives { //find closest hive
		if distance > dist(key, coords) {
			distance = dist(key, coords)
			target = key
		}
	}
	// fmt.Printf("[TURN START] MyPos: %v | Target: %v\n", coords, target)
	if distance == 1 { //if next to a hive of yours, put flower
		// fmt.Println("Giving order FORAGE")
		o.Type = FORAGE
		return o
	}
	temp := aStar(coords, target, true, &gameMap) //a-star algorithm to find path, boolean true tells it to stop next to target, not on it
	// fmt.Printf("[TURN END] Selected Move: %+v\n", temp)
	if (temp != Order{}) {
		return temp
	}
	return (Order{
		Type:      MOVE,
		Coords:    coords,
		Direction: dirs[rand.Intn(len(dirs))],
	}) //fallback: try a random move. TODO:move to random empty hex, not random hex
}

func (gm *GameMap) estimateTurns(beeCount int) int {
	sum := 0
	for field, nonEmpty := range gm.FlowerFields {
		distance := 20000
		if !nonEmpty {
			continue
		}
		var best Coords
		for hive, _ := range gm.MyHives {
			if dist(field, hive) < distance {
				best = hive
				distance = dist(field, hive)
			}
		}
		sum += dist(field, best) * 2 * int(gm.Mapped[field].Flowers) / beeCount
	}
	return sum
}

func isEmpty(c Coords, d Direction) bool {
	target := Coords{
		Row: c.Row + DirectionToOffset[d].Row,
		Col: c.Col + DirectionToOffset[d].Col,
	}
	return gameMap.Mapped[target].IsWalkable
}

func (gm *GameMap) getNearestFlower(coords Coords) Coords {
	distance := 20000
	field := Coords{}
	for temp, there := range gm.FlowerFields {
		if there && dist(coords, temp) < distance {
			distance = dist(coords, temp)
			field = temp
		}
		//TODO: pathfound distance rather than map distance
	}
	// fmt.Printf("Closest flower to %v found at %v", coords, field)
	return field
}

func (gm *GameMap) getNearestUnknown(coords Coords) Coords {
	distance := math.MaxInt16
	target := Coords{}
	for temp, tile := range gm.Mapped {
		temp_distance := dist(coords, temp)
		if tile.Type == UNKNOWN && temp_distance < distance {
			distance = temp_distance
			target = temp
		}
	}
	fmt.Println("Bee location\t: ", coords)
	fmt.Println("Nearest unknown\t: ", target)
	return target
}

func beeOrder(h Hex, coords Coords, player int) Order {
	if h.Entity.HasFlower { //if carrying a flower, go home
		return goHome(h, coords)
	} else if h.Resources > 0 { //if in a field, pick up a flower
		return (Order{
			Type:      FORAGE,
			Coords:    coords,
			Direction: dirs[rand.Intn(len(dirs))],
		})
	} else {
		target := gameMap.getNearestFlower(coords)
		temp := aStar(coords, target, false, &gameMap)
		if (temp != Order{}) {
			return temp
		}
		return (Order{ //fallback: random move
			Type:      MOVE,
			Coords:    coords,
			Direction: dirs[rand.Intn(len(dirs))],
		})
	}
}

func exploreOrder(h Hex, coords Coords, player int) Order {
	target := gameMap.getNearestUnknown(coords)
	temp := aStar(coords, target, false, &gameMap)
	if (temp != Order{}) {
		return temp
	}
	return (Order{ //fallback: random move
		Type:      MOVE,
		Coords:    coords,
		Direction: dirs[rand.Intn(len(dirs))],
	})
}

func spawnBee(c Coords, player int) Order {
	for _, dir := range dirs {
		if isEmpty(c, dir) {
			return (Order{
				Type:      SPAWN,
				Coords:    c,
				Direction: dir,
			})
		}
	}
	return Order{}
}

func think(state *GameState, player int) []Order {
	var orders []Order
	gameMap.updateGameMap(state, player)
	gameMap.ExpandFringe()
	gameMap.updateExploringStatus()
	for coords, hex := range gameMap.MyBees { //first, order flowerbees
		if hex != nil && hex.Entity.HasFlower {
			orders = append(orders, beeOrder(*hex, coords, player))
		}
	}
	for coords, hex := range gameMap.MyBees { //second, order free bees
		if hex != nil && !hex.Entity.HasFlower {
			if exploring && gameMap.Mapped[coords].Type == EXPLORER {
				orders = append(orders, exploreOrder(*hex, coords, player))
				fmt.Println("Giving explore order, exploring is ", exploring)
			} else {
				orders = append(orders, beeOrder(*hex, coords, player))
			}
		}
	}
	for coords, _ := range gameMap.MyHives { //see if we should spawn bees
		timeToEmpty := gameMap.estimateTurns(len(gameMap.MyBees))
		newTime := gameMap.estimateTurns(len(gameMap.MyBees) + 1)
		if timeToEmpty-newTime > 30 && state.PlayerResources[player] > 6 { //TODO: smarter check
			orders = append(orders, spawnBee(coords, player))
		}
	}
	return orders
}

func main() {
	if len(os.Args) <= 3 {
		fmt.Println("Usage: ./agent <host> <gameid> <name>")
		os.Exit(1)
	}
	exploring = true
	gameMap = NewGameMap()
	host := os.Args[1]
	id := os.Args[2]
	name := os.Args[3]

	Run(host, id, name, think)
}
