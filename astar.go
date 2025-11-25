package main

import "sort"

/*
	builds on https://reintech.io/blog/a-star-search-algorithm-in-go
*/

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

func goTo(targetHex Coords, state *GameState) Order {
    return Order{
        Command: "MOVE",
        Target:  targetHex,
        // You might need an EntityID here depending on your Order struct
    }
}

func aStar(loc, target Coords, state *GameState) Order {
	startNode := &Node{
		hex:	loc,
		cost:	0,
		dist:	dist(loc, target),
		total:	dist(loc, target),
		prev:	nil,
	}
	var candidates []*Node{startNode}
	var	candidateMap = map[Coords]*Node{loc: startNode}
	var rejects	= make(map[Coords]bool)
	for len(candidates) > 0 { //Use a for-loop as a while-loop
		sortByTotal(candidates)
		current := candidates[0]
		candidates = candidates[1:]
		delete(candidateMap, current.hex)

		if current.hex == target { //loop back to the first step
			for current.prev != nil && current.prev.hex != loc { current = current.prev }
			return goTo(current.hex, state)
		}

		rejects[current.hex] = true //never come back here

		for _, offset := range DirectionToOffset {
			neighborCoords := Coords{
				Row: current.hex.Row + offset.Row,
				Col: current.hex.Col + offset.Col,
    		}
			neighborHex := state.Hexes[neighborCoords]
			if (neighborHex == nil || !neighborHex.Terrain.IsWalkable() || neighborHex.Entity != nil) {
				continue
			}
			if rejects[neighborCoords] { continue }
			neighborCost := current.cost + 1 //alternate cost for breakable walls?
			existing, inCandidates := candidateMap[neighborCoords] //looks for the candidate coordinates in the candidate map (inCandidates is a bool whether the key was found, existing is a Coord struct that is either a value or nil
			if inCandidates { //if location was alrready in candidates, check and update if better than old version
				if neighborCost >= existing.cost { continue }
				existing.prev = current
				existing.cost = neighborCost
				existing.total = existing.cost + existing.dist
			} else {
				newNode := &Node{
					hex: neighborCoords,
					cost: neighborCost,
					dist: dist(neighborCoords, target)
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
