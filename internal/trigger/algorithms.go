package trigger

import "math/rand"

func getRandomizedVal(mean float64, stdDev float64) float64 {
	val := rand.NormFloat64()*stdDev + mean
	return val
}
