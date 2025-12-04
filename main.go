package main

import (
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"

	. "hive-arena/common"
)

var dirs = []Direction{E, SE, SW, W, NW, NE}
var gameMap GameMap
var exploring bool
var should_build_hive bool = false

var previousDirection Coords
var activeExplorerCoords Coords
var previousExplorerCoords Coords
var hasExplorer bool = false
var explorerTarget = Coords{Row: -100, Col: -100}

var (
	BeesPerHive    int     = 5
	ScoreThreshold float64 = 75.0
)

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

func IdentifyExplorer(gm *GameMap) {
	if !hasExplorer {
		RecruitNewExplorer(gm)
		return
	}

	if isMyBee(previousExplorerCoords, gm) {
		activeExplorerCoords = previousExplorerCoords
		return
	}

	// Check neighbors (maybe they moved)
	for _, offset := range directionToOffset {
		neighbor := addCoords(previousExplorerCoords, offset)
		if isMyBee(neighbor, gm) {
			activeExplorerCoords = neighbor
			// Found them! Update the "previous" tracker for next turn
			previousExplorerCoords = neighbor
			return
		}
	}

	// We must recruit a replacement.
	fmt.Println("Explorer MIA! Recruiting replacement...")
	RecruitNewExplorer(gm)
}

func isMyBee(c Coords, gm *GameMap) bool {
	tile, ok := gm.Mapped[c]
	return ok && tile.Type == OWN_BEE
}

func RecruitNewExplorer(gm *GameMap) {
	// Simple logic: Pick bee furthest from Hive (closest to the unknown)
	var bestBee Coords
	maxDist := -1
	found := false

	for coords, tile := range gm.Mapped {
		if tile.Type == OWN_BEE && !tile.BeeHasFlower {
			d := getDistanceToNearestHive(coords, gm)
			if d > maxDist {
				maxDist = d
				bestBee = coords
				found = true
			}
		}
	}

	if found {
		activeExplorerCoords = bestBee
		previousExplorerCoords = bestBee
		hasExplorer = true
		fmt.Printf("Recruited NEW Explorer at %v\n", bestBee)
		tile := gameMap.Mapped[activeExplorerCoords]
		tile.Type = EXPLORER
		gameMap.Mapped[activeExplorerCoords] = tile
	} else {
		hasExplorer = false
	}
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
	if distance == 1 { //if next to a hive of yours, put flower
		o.Type = FORAGE
		return o
	}
	temp := aStar(coords, target, true, &gameMap) //a-star algorithm to find path, boolean true tells it to stop next to target, not on it
	if (temp != Order{}) {
		return temp
	}
	return (Order{
		Type:      MOVE,
		Coords:    coords,
		Direction: dirs[rand.Intn(len(dirs))],
	}) //fallback: try a random move. TODO:move to random empty hex, not random hex
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
	return field
}

func (gm *GameMap) getNearestUnknown(coords Coords) Coords {
	if explorerTarget.Row != -100 {
		tile, exists := gm.Mapped[explorerTarget]
		if exists && tile.Type == UNKNOWN {
			return explorerTarget
		}
	}
	distance := math.MaxInt16
	target := Coords{}
	found := false
	for temp, tile := range gm.Mapped {
		if tile.Type == UNKNOWN {
			temp_distance := dist(coords, temp)
			if temp_distance < distance {
				distance = temp_distance
				target = temp
				found = true
			}
		}
	}
	if found {
		fmt.Println("New Explorer Target Acquired: ", target)
		explorerTarget = target
		return target
	}
	return coords
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
	if target == coords {
		return (Order{ //fallback: random move
			Type:      MOVE,
			Coords:    coords,
			Direction: dirs[rand.Intn(len(dirs))],
		})

	}
	temp := aStar(coords, target, true, &gameMap)
	if (temp != Order{}) {
		return temp
	}
	// If A* still fails (e.g., surrounded by rocks), try random move
	return Order{
		Type:      MOVE,
		Coords:    coords,
		Direction: dirs[rand.Intn(len(dirs))],
	}
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

func (gm *GameMap) getNearestFreeBee(c Coords) Coords {
	var closest Coords = Coords{}
	distance := 20000
	for loc, _ := range gm.MyBees {
		if gm.Mapped[loc].BeeHasFlower {
			continue
		}
		d := dist(loc, c)
		if d < distance {
			distance = d
			closest = loc
		}
	}
	return closest
}

func think(state *GameState, player int) []Order {
	var orders []Order
	gameMap.updateGameMap(state, player)
	gameMap.ExpandFringe()
	gameMap.updateExploringStatus()
	if exploring && len(gameMap.MyBees) > 2 {
		RecruitNewExplorer(&gameMap)
	}

	//building a new hive logic
	gameMap.updateBuilderLoc()
	loc, score := gameMap.bestNewHivePos()
	if !exploring || Unknown_count < 7 || score > ScoreThreshold {
		should_build_hive = true
		if len(gameMap.MyHives) < 2 && state.PlayerResources[player] >= 12 {
			gameMap.IsBuilding = true
			gameMap.BuildTarget = loc
			gameMap.Builders[0] = gameMap.getNearestFreeBee(loc)
			should_build_hive = false
			orders = append(orders, gameMap.goBuild())
		}
	}

	//sending out blockers logic
	gameMap.updateBlockers()
	if (gameMap.TargetHive == Coords{}) {
		newBlocker := (len(gameMap.MyBees) >= BeesPerHive*len(gameMap.MyHives))
		fmt.Printf("[TURN %d DEBUG] Blocker Check: Bees=%d/%d | CurrentBlockers=%d | AllowNew=%v\n", state.Turn, len(gameMap.MyBees), BeesPerHive*len(gameMap.MyHives), gameMap.blockerCount(), newBlocker)
		if !exploring && newBlocker && gameMap.blockerCount() < state.NumPlayers-1 { //we should make a new blocker
			gameMap.makeBlockTargets()
			for hive, _ := range gameMap.EnemyHives {
				if !gameMap.IsBlocking[hive] { //reject hives already blocked
					gameMap.TargetHive = hive              //set this hive as target
					target := gameMap.BlockerTargets[hive] //target for the bee to go to
					nearestBee := gameMap.getNearestFreeBee(target)
					fmt.Printf("[TURN %d DEBUG] ⚔️ ASSIGNING BLOCKER! Bee %v -> Hive %v (Target Spot: %v)\n", state.Turn, nearestBee, hive, target)
					gameMap.BlockerPositions[0] = nearestBee
					orders = append(orders, gameMap.goSabotage(hive, target, nearestBee))
					break
				}
			}
		}
	} else {
		bee := gameMap.BlockerPositions[0]
		target := gameMap.BlockerTargets[gameMap.TargetHive]
		fmt.Printf("[TURN %d DEBUG] 🏃 Blocker %v is moving toward %v\n", state.Turn, bee, target)
		orders = append(orders, gameMap.goSabotage(gameMap.TargetHive, target, bee))
	}

	//permablockers logic
	for bee, _ := range gameMap.MySaboteurs {
		targetHive := Coords{}
		for hive, _ := range gameMap.EnemyHives {
			if dist(hive, bee) == 1 {
				targetHive = hive
				break
			}
		}
		orders = append(orders, gameMap.attackOrWait(targetHive, bee))
	}

	//basic bee logic
	for coords, hex := range gameMap.MyBees { //first, order flowerbees
		if hex != nil && hex.Entity.HasFlower {
			orders = append(orders, beeOrder(*hex, coords, player))
		}
	}
	for coords, hex := range gameMap.MyBees { //second, order free bees
		isActiveBlocker := (coords == gameMap.BlockerPositions[0])
		isSaboteur := gameMap.MySaboteurs[coords]

		if isActiveBlocker || isSaboteur {
			continue
		}

		// If we are building, and this is the designated builder
		if gameMap.IsBuilding && coords == gameMap.Builders[0] {
			continue
		}
		if hex != nil && !hex.Entity.HasFlower {
			if exploring && gameMap.Mapped[coords].Type == EXPLORER {
				orders = append(orders, exploreOrder(*hex, coords, player))
			} else {
				orders = append(orders, beeOrder(*hex, coords, player))
			}
		}
	}
	for coords, _ := range gameMap.MyHives { //see if we should spawn bees
		if !should_build_hive && len(gameMap.MyBees) >= BeesPerHive*len(gameMap.MyHives)+gameMap.blockerCount() ||
			int(gameMap.FlowerCount)/state.NumPlayers < 6 {
			break
		}
		beesNear := 0
		for bee := range gameMap.MyBees {
			if dist(coords, bee) < 6 {
				beesNear++
			}
		}

		empty := beesNear < 3
		isWorthIt := gameMap.BreakEven(coords, beesNear)
		haveMoney := state.PlayerResources[player] >= 6

		// PRINT THE TRUTH
		//		fmt.Printf("[TURN %d] Hive %v Analysis:\n", state.Turn, coords)
		//		fmt.Printf("\tBreakEven: %v\n", isWorthIt)
		if (empty || isWorthIt) && haveMoney {
			o := spawnBee(coords, player)
			if o.Type == "" {
				fmt.Printf("\tCRITICAL: Conditions met, but spawnBee returned empty! (Hive blocked?)\n")
			}
			orders = append(orders, o)
		}
	}
	return orders
}

func main() {
	flag.IntVar(&BeesPerHive, "bees", 5, "Target number of bees per hive")
	flag.Float64Var(&ScoreThreshold, "score", 50.0, "Score threshold for new hive")

	flag.Parse()

	args := flag.Args()
	if len(args) < 3 {
		fmt.Println("Usage: ./agent [flags] <host> <gameid> <name>")
		os.Exit(1)
	}

	exploring = true
	gameMap = NewGameMap()
	host := args[0]
	id := args[1]
	name := args[2]

	Run(host, id, name, think)
}
