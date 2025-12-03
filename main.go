package main

import (
	"fmt"
	"math/rand"
	"os"
)

import . "hive-arena/common"

var dirs = []Direction{E, SE, SW, W, NW, NE}
var gameMap GameMap
var explorers int

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
//	fmt.Printf("[TURN START] MyPos: %v | Target: %v\n", coords, target)
	if distance == 1 { //if next to a hive of yours, put flower
		o.Type = FORAGE
		return o
	}
	temp := aStar(coords, target, true, &gameMap) //a-star algorithm to find path, boolean true tells it to stop next to target, not on it
//	fmt.Printf("[TURN END] Selected Move: %+v\n", temp)
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

func (gm *GameMap) getNearestFlower(coords Coords) Coords{
	distance := 20000
	field := Coords{}
	for temp, there := range gm.FlowerFields {
		if (there && dist(coords, temp) < distance) { 
			distance = dist(coords, temp)
			field = temp
		}
		//TODO: pathfound distance rather than map distance
	}
//	fmt.Printf("Closest flower to %v found at %v", coords, field)
	return field
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
		if explorers > 1 {
			explorers--
			return (Order{ //TODO: targeted explore
				Type:      MOVE,
				Coords:    coords,
				Direction: dirs[rand.Intn(len(dirs))],
			})

		}
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
	for loc,_ := range gm.MyBees {
		if gm.Mapped[loc].BeeHasFlower {continue}
		d := dist(loc, c)
		if (d < distance) {
			distance = d
			closest = loc
		}
	}
	return closest
}

const (
	BeesPerHive = 5
	ScoreThreshold = 50.0
)

func think(state *GameState, player int) []Order {
	var orders []Order
	if (state.Turn < 100) {	explorers = 1 }
	gameMap.updateGameMap(state, player)
	if (state.Turn % 20) == 0 { gameMap.makeBlockTargets() }

	//building a new hive logic
	gameMap.updateBuilderLoc()
	if len(gameMap.MyHives) < 2 && state.PlayerResources[player] >= 12 && state.Turn % 10 == 0  {
		gameMap.IsBuilding = true
		loc, score := gameMap.bestNewHivePos()
		if loc != gameMap.BuildTarget && score > ScoreThreshold {
			gameMap.BuildTarget = loc
			gameMap.Builders[0] = gameMap.getNearestFreeBee(loc)
			orders = append(orders, gameMap.goBuild())
		}
	}
	if gameMap.IsBuilding{
		orders = append(orders, gameMap.goBuild())
	}

	//sending out blockers logic
	gameMap.updateBlockers()
	if (gameMap.TargetHive == Coords{}) {
		newBlocker := (len(gameMap.MyBees) >= BeesPerHive * len(gameMap.MyHives))
		fmt.Printf("[TURN %d DEBUG] Blocker Check: Bees=%d/%d | CurrentBlockers=%d | AllowNew=%v\n", state.Turn, len(gameMap.MyBees), BeesPerHive * len(gameMap.MyHives), gameMap.blockerCount(), newBlocker)
		if newBlocker && gameMap.blockerCount() < state.NumPlayers - 1 {//we should make a new blocker
			for hive, _ := range gameMap.EnemyHives { 
				if !gameMap.IsBlocking[hive] { //reject hives already blocked
					gameMap.TargetHive = hive	//set this hive as target
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
		if (hex != nil && hex.Entity.HasFlower){
			orders = append(orders, beeOrder(*hex, coords, player))
		}
	}
	for coords, hex := range gameMap.MyBees { //second, order free bees
		if (hex != nil && !hex.Entity.HasFlower){
			orders = append(orders, beeOrder(*hex, coords, player))
		}
	}
	for coords, _ := range gameMap.MyHives { //see if we should spawn bees
		if len(gameMap.MyBees) >= BeesPerHive * len(gameMap.MyHives) + gameMap.blockerCount() ||
			int(gameMap.FlowerCount) / state.NumPlayers < 6 {
			 break 
		}
		beesNear := 0
		for bee := range gameMap.MyBees {
			if dist(coords, bee) < 6 {beesNear++}
		}
		
		empty := beesNear < 3
		isWorthIt := gameMap.BreakEven(coords, beesNear)
		haveMoney := state.PlayerResources[player] >= 6
    
		// PRINT THE TRUTH
		fmt.Printf("[TURN %d] Hive %v Analysis:\n", state.Turn, coords)
		fmt.Printf("\tBreakEven: %v\n", isWorthIt)		
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
	if len(os.Args) <= 3 {
		fmt.Println("Usage: ./agent <host> <gameid> <name>")
		os.Exit(1)
	}

	gameMap = NewGameMap()
	host := os.Args[1]
	id := os.Args[2]
	name := os.Args[3]

	Run(host, id, name, think)
}
