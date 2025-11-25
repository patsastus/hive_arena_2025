package main

import (
	"fmt"
	"math/rand"
	"math"
	"os"
)

import . "hive-arena/common"

var dirs = []Direction{E, SE, SW, W, NW, NE}
var hives = make(map[Coords]bool)

func dist(one, two Coords) int{
	dx := one.Row - two.Row
	if dx < 0 { dx = -dx }

	dy := one.Col - two.Col
	if dy < 0 { dy = -dy }

	if dy < dx { return dx }
	return dx + (dy - dx) / 2
}

func goHome(h Hex, coords Coords, state *GameState) Order {
	distance := 20000 
	var o Order
	var target Coords
	o.Coords = coords
	o.Type = MOVE
	for key, _ := range hives { 	//find closest hive
		if distance > dist(key,coords) {
			distance = dist(key, coords)
			target = key
		}
	}
	if distance == 1 { //if next to a hive of yours, put flower
		fmt.Println("Giving order FORAGE")
		o.Type = FORAGE
		return o
	}
	for dir, offset := range DirectionToOffset { //loop through all directions, if it's unblocked and reduces distance to target, go there 
        next := Coords{
            Row: coords.Row + offset.Row,
            Col: coords.Col + offset.Col,
        }
		o.Direction = dir
        if state.TargetIsBlocked(&o){
            continue
        }
        newDist := dist(next, target)
        if newDist < distance {
			fmt.Println("Distance %d, giving order MOVE %s", newDist, dir)
			return o 
		}
    }
	return (Order{
			Type:      MOVE,
			Coords:    coords,
			Direction: dirs[rand.Intn(len(dirs))],
	}) //fallback: try a random move
}


func beeOrder(h Hex, coords Coords, state *GameState, player int) Order {
	if h.Entity.HasFlower { //if carrying a flower, go home
		return goHome(h, coords, state)
	} else if h.Resources > 0 { //if in a field, pick up a flower
		return (Order{
			Type:      FORAGE,
			Coords:    coords,
			Direction: dirs[rand.Intn(len(dirs))],
		})
	} else {
		return (Order{ //fallback: random move
			Type:      MOVE,
			Coords:    coords,
			Direction: dirs[rand.Intn(len(dirs))],
		})
	}
}

func think(state *GameState, player int) []Order {

	var orders []Order

	for coords, hex := range state.Hexes {
		unit := hex.Entity
		if unit != nil && unit.Type == HIVE && unit.Player == player {
			if hives[coords] == false {
				fmt.Println(coords, unit)
			}
			hives[coords] = true
		}
		if unit != nil && unit.Type == BEE && unit.Player == player {
			fmt.Println(coords, unit)
			orders = append(orders, beeOrder(*hex, coords, state, player))
		}
	}

	return orders
}

func main() {
	if len(os.Args) <= 3 {
		fmt.Println("Usage: ./agent <host> <gameid> <name>")
		os.Exit(1)
	}

	host := os.Args[1]
	id := os.Args[2]
	name := os.Args[3]

	Run(host, id, name, think)
}
