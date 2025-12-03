package main

import (
	"fmt"
)

import . "hive-arena/common"

func (gm *GameMap) updateBlockers(){
	if (gm.TargetHive == Coords{}) { return } //when no target, skip
	// If a bee exists at the location we expected to move to, update our tracking
	if _, exists := gm.MyBees[gm.BlockerPositions[1]]; exists {
		gm.BlockerPositions[0] = gm.BlockerPositions[1]
	}
}

func (gm *GameMap) findFlanks(hive, blocker Coords) (Coords, Coords) {
	var flanks []Coords
	for _, dir := range dirs {
		neighbor := getCoords(blocker, dir)
        if dist(neighbor, hive) == 1 {
            flanks = append(flanks, neighbor)
        }
	}
	return flanks[0], flanks[1]
}

func (gm *GameMap) makeBlockTargets() {
	for hive, _ := range gm.EnemyHives {
		if gm.IsBlocking[hive] {continue}
		bestFlanks := 0
		bestTarget := Coords{}
		for _, dir := range dirs {
			target := getCoords(hive, dir)
			if (!gm.Mapped[target].IsWalkable) {continue}
			flankOne, flankTwo := gm.findFlanks(hive, target)
			numFlanks := 0
			if gm.Mapped[flankOne].IsWalkable {numFlanks++}
			if gm.Mapped[flankTwo].IsWalkable {numFlanks++}
			if numFlanks == 2 {
				bestFlanks = 2
				bestTarget = target
				break
			}
			if numFlanks > bestFlanks {
				bestFlanks = numFlanks
				bestTarget = target
			}
		}
		gm.BlockerTargets[hive] = append(gm.BlockerTargets[hive], bestTarget)
		fmt.Println("best target of %v is %v\n", hive, bestTarget)
	}
}

func (gm *GameMap) attackOrWait(hive, bee Coords) Order {
	flanks := gm.findFlanks(hive, bee)
	target := Coords{}
	if gm.Mapped[flanks[0]].Type == ENEMY_BEE {
		target = flanks[0]
	} else if gm.Mapped[flanks[1]].Type == ENEMY_BEE {
		target = flanks[1]
	} else {
		for _, dir := range dirs {
			temp := getCoords(bee, dir)
			if gm.Mapped[temp].Type == ENEMY_BEE {
				target = temp
				break
			}
		}
	}
	if (target == Coords{}) {return (Order{})}
	var o Order
	o.Type = ATTACK
	o.Coords = bee
	d,_ := getDirection(bee, target)
	o.Direction = d
	return o
}

func (gm *GameMap) blockerCount() int{
	sum = 0;
	for hive, blocked := range gm.IsBlocking {
		if (blocked) { sum++ }
	}
	return sum
}

func (gm *GameMap) goSabotage(hive, target, bee Coords) Order {
	if bee == target {
		gm.IsBlocking[hive] = true
		gm.MySaboteurs[bee] = true
		if gm.TargetHive == hive { //reset targets if this is the first time this bee is in the correct place
			gm.TargetHive = Coords{}
			clear(gm.BlockerPositions)
		}
		return gm.attackOrWait(hive, bee)
	}
	order := aStar(bee, target, false, gm)
	gm.BlockerPositions[1] = getCoords(bee, order.Direction)
	return order
}
