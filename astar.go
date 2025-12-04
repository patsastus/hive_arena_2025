package main

import "sort"
import "fmt"
import "math/rand"
import . "hive-arena/common"

type Node struct {
	hex 				Coords
	cost, dist, total	int
	prev				*Node
}

func sortByTotal(candidates []*Node) {
    sort.Slice(candidates, func(i, j int) bool {
        // We want the LOWEST total cost to be first (index 0)
        return candidates[i].total < candidates[j].total
    })
}

func getCoords(loc Coords, dir Direction) Coords {
    offset, ok := DirectionToOffset[dir]
    if !ok {
        return Coords{}
    }
    target := Coords{
        Row: loc.Row + offset.Row,
        Col: loc.Col + offset.Col,
    }
    return target
}

func getDirection(loc, target Coords) (Direction, bool) {
	offset := Coords{
		Row: target.Row - loc.Row,
		Col: target.Col - loc.Col,
	}
	for dir, coords := range DirectionToOffset {
        if coords == offset {
            return dir, true
        }
    }
    return "", false // Return empty string and false if no match
}

func goTo(loc, targetHex Coords, myMap *GameMap) Order {
	dir, found := getDirection(loc, targetHex)
	if (!found) {
		fmt.Println("goTo got invalid source/target combo")
		dir = dirs[rand.Intn(len(dirs))]
	}
	o := Order{
        Type: MOVE,
        Coords: loc,
		Direction: dir,
    }
	if myMap.Mapped[targetHex].Type == ENEMY_WALL {
		o.Type = ATTACK
	} else {
		myMap.Targeted[targetHex] = true
	}
	return o
}

/*
	A-* pathfinding algorithm
	builds on https://reintech.io/blog/a-star-search-algorithm-in-go
	and https://en.wikipedia.org/wiki/A*_search_algorithm
	returns an order with the first step of the path
	stopNextTo is so you can go to non-walkable target (hive, wall, enemy) == true, or walkable space(empty, field) == false
	also return total distance ?
*/
func aStar(loc, target Coords, stopNextTo bool, myMap *GameMap) Order {
	startNode := &Node{
		hex:	loc,
		cost:	0,
		dist:	dist(loc, target),
		total:	dist(loc, target),
		prev:	nil,
	}
	candidates := []*Node{startNode}
	candidateMap := map[Coords]*Node{loc: startNode}
	rejects	:= make(map[Coords]bool)
	for len(candidates) > 0 { //Use a for-loop as a while-loop
		sortByTotal(candidates)
		current := candidates[0]
		candidates = candidates[1:]
		delete(candidateMap, current.hex)

		atTarget := current.hex == target //on the target square
		nextToTarget := stopNextTo && dist(current.hex, target) == 1 //target isn't walkable and next to it
		if atTarget || nextToTarget { //loop back to the first step
			for current.prev != nil && current.prev.hex != loc { current = current.prev }
			return goTo(loc, current.hex, myMap)
		}

		rejects[current.hex] = true //never come back here
		for _, offset := range DirectionToOffset {
			neighborCoords := Coords{
				Row: current.hex.Row + offset.Row,
				Col: current.hex.Col + offset.Col,
    		}
			neighborGMO := myMap.Mapped[neighborCoords]
			if (neighborGMO == (GameMapObject{}) ||
				myMap.Targeted[neighborCoords] ||
				!(neighborGMO.Type == EMPTY_HEX || neighborGMO.Type == ENEMY_WALL)) { 
				continue 
			}
			if rejects[neighborCoords] { continue }
			neighborCost := current.cost + 1
			if neighborGMO.Type == ENEMY_WALL { neighborCost += 6 }
			cand, exists := candidateMap[neighborCoords] //looks for the candidate coordinates in the candidate map (exists is a bool whether the key was found, cand is a Coord struct that is either a value or nil
			if exists { //if location was already in candidates, check and update if better than old version
				if neighborCost >= cand.cost { continue }
				cand.prev = current
				cand.cost = neighborCost
				cand.total = cand.cost + cand.dist
			} else {
				newNode := &Node{
					hex: neighborCoords,
					cost: neighborCost,
					dist: dist(neighborCoords, target),
					prev: current,
				}
				newNode.total = newNode.cost + newNode.dist
				candidates = append(candidates, newNode)
				candidateMap[neighborCoords] = newNode
			}
		}
	}
	return Order{}
}
