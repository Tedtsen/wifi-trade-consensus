package trigger

import "math/rand"

func getRandomizedVal(mean, stdDev, lowest, highest float64) float64 {
	val := rand.NormFloat64()*stdDev + mean
	if val < lowest {
		return lowest
	} else if val > highest {
		return highest
	} else {
		return val
	}
}
