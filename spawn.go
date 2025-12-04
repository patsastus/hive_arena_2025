package main

import (
// "fmt"
)

import . "hive-arena/common"

// distance-weighted sum of flowers near location
func (gm *GameMap) hiveScore(coords Coords, radius int) float64 {
	var minRow, maxRow, minCol, maxCol int
	minRow = coords.Row - radius
	maxRow = coords.Row + radius
	minCol = coords.Col - (radius * 2)
	maxCol = coords.Col + (radius * 2)
	sum := 0.0
	for r := minRow; r <= maxRow; r++ {
		for c := minCol; c <= maxCol; c++ {
			test := Coords{
				Row: r,
				Col: c,
			}
			d := dist(test, coords)
			if d > radius || d == 0 {
				continue
			}
			tile := gm.Mapped[test]
			if (tile == GameMapObject{}) {
				continue
			}
			sum += float64(tile.Flowers) / float64(d)
		}
	}
	return sum
}

func (gm *GameMap) updateBuilderLoc() {
	if _, exists := gm.MyBees[gm.Builders[1]]; exists { //assign a boolean to 'exists'; then use it. If true, builder bee is in expected position
		gm.Builders[0] = gm.Builders[1] //update the position of the builder
	}
}

// returns the coordinates of the best hive position and a score of how many flowers are nearby
func (gm *GameMap) bestNewHivePos() (Coords, float64) {
	bestScore := 0.0
	bestLocation := Coords{}
	const ( //tunable constants
		minDToOwn   = 6.0
		minDToEnemy = 12.0
		wExpansion  = 0.10
		scanRadius  = 5
	)
	for field, object := range gm.Mapped {
		if object == (GameMapObject{}) || !object.IsWalkable || object.Type == ENEMY_HIVE || object.Type == OWN_HIVE {
			continue
		}
		closestHiveD := 20000
		for hive, _ := range gm.MyHives {
			closestHiveD = min(closestHiveD, dist(hive, field))
		}
		if closestHiveD < minDToOwn {
			continue
		}
		closestEnemyD := 20000
		for hive, _ := range gm.EnemyHives {
			d := dist(hive, field)
			if d < closestEnemyD {
				closestEnemyD = d
			}
		}
		rawScore := gm.hiveScore(field, scanRadius)
		if rawScore < 0.1 {
			continue
		}

		safetyFactor := 1.0
		if float64(closestEnemyD) < minDToEnemy {
			safetyFactor = float64(closestEnemyD) / minDToEnemy
		}

		expansionFactor := 1.0 + (float64(closestHiveD) * wExpansion)
		finalScore := rawScore * safetyFactor * expansionFactor

		if finalScore > bestScore {
			bestScore = finalScore
			bestLocation = field
		}
	}
	// fmt.Printf("Best hive location: %v, score: %f\n", bestLocation, bestScore)
	return bestLocation, bestScore
}

// used to calculate flowers-per-turn and turns-until-depleted
func (gm *GameMap) effectiveDistance() float64 {
	weightedSum := 0.0
	for field, nonEmpty := range gm.FlowerFields {
		distance := 20000
		if !nonEmpty {
			continue
		}
		for hive, _ := range gm.MyHives {
			if dist(field, hive) < distance {
				distance = dist(field, hive)
			}
		}
		weightedSum += float64(gm.Mapped[field].Flowers) / float64(distance)
	}
	if weightedSum < 0.001 {
		return 0.0
	}
	return (float64(gm.FlowerCount) / weightedSum)
}

func (gm *GameMap) estimateFlowersPerTurn(beeCount int) float64 {
	d := gm.effectiveDistance()
	if d < 0.001 {
		return 0.0
	}
	return float64(beeCount) / (2.0*d + 2.0) //+2 is for picking up and dropping off flower
}

// assumes opponents have the same flowerrate as us
func (gm *GameMap) turnsUntilDepleted() int {
	ratio := float64(len(gm.MyBees)+gm.EnemyBees) / float64(len(gm.MyBees))
	FPT := gm.estimateFlowersPerTurn(len(gm.MyBees))
	if FPT < 0.001 {
		return 0
	}
	return int((float64(gm.FlowerCount) / ratio) / FPT)
}

/*//calculate the effect of spawning a new bee, and return whether we'll break even on the cost by
//the time all flowers are depleted (at current estimated rate). slightly overeager spawn?
//because collection rate probably going up early on
func (gm *GameMap) BreakEven(hive Coords) bool {
	currentFPT := gm.estimateFlowersPerTurn(len(gm.MyBees)) //TODO:replace with foragerCount?
	newFPT := gm.estimateFlowersPerTurn(len(gm.MyBees) + 1)
	if (newFPT - currentFPT) * float64( gm.turnsUntilDepleted() ) < 6 { return false }
	return true
}
*/

func (gm *GameMap) BreakEven(hive Coords, beesNear int) bool {
	localPotential := 0.0
	for field, isField := range gm.FlowerFields {
		if !isField {
			continue
		}
		d := dist(hive, field)
		if d == 0 {
			d = 1
		}
		if d <= 12 {
			localPotential += float64(gm.Mapped[field].Flowers) / float64(d)
		}
	}
	if beesNear == 0 {
		beesNear++
	}
	score := localPotential / float64(beesNear)

	// DEBUG
	// fmt.Printf("Hive %v Score: %.2f (Pot: %.1f / Bees: %.1f)\n", hive, score, localPotential, beesNear)

	// Tune this number! Start low (0.5) and raise it if you overspawn.
	return score > 0.5
}

func (gm *GameMap) goBuild() Order {
	if gm.Builders[0] != gm.BuildTarget {
		temp := aStar(gm.Builders[0], gm.BuildTarget, false, gm)
		gm.Builders[1] = getCoords(gm.Builders[0], temp.Direction)
		if temp != (Order{}) {
			return temp
		}
	}
	if dist(gm.BuildTarget, gm.Builders[0]) > 1  && (gm.Mapped[gm.BuildTarget].Type == ENEMY_HIVE || gm.Mapped[gm.BuildTarget].Type == ENEMY_WALL) {
		temp := aStar(gm.Builders[0], gm.BuildTarget, true, gm)
		gm.Builders[1] = getCoords(gm.Builders[0], temp.Direction)
		if temp != (Order{}) {
			return temp
		}
	}
	gm.IsBuilding = false
	HasExplorer = false
	return (Order{
		Type:   BUILD_HIVE,
		Coords: gm.Builders[0],
	})
}
