package consumer

import (
	"fmt"
	"math"
)

func (c *consumer) calculateFFSfinal(tran transaction) (FFS, providerInfo) {
	FFSfinal := FFS{}
	winner := providerInfo{}
	highestFF := -10.0
	for _, targetProvider := range tran.providerList {
		populationN := 0.0
		FFsum := 0.0
		for _, scorerProvider := range tran.providerList {
			if targetProvider.ProviderID == scorerProvider.ProviderID {
				continue
			}

			FFS, exists := tran.allFFS[scorerProvider.ProviderID]
			if !exists {
				fmt.Println("FFS not found for scorer provider:", scorerProvider.ProviderID)
				continue
			}
			fmt.Println("provider id:", scorerProvider.ProviderID, ";FFS:", FFS)

			FF, exists := FFS[targetProvider.ProviderID]
			if !exists {
				fmt.Println("FF not found for target provider:", targetProvider.ProviderID)
				continue
			}

			FFsum += FF
			populationN += 1
		}

		// Calculate FFmu (mean)
		FFmu := FFsum / populationN

		// Calculate FFsigma (standard deviation)
		dividend := 0.0
		for _, scorerProvider := range tran.providerList {
			if targetProvider.ProviderID == scorerProvider.ProviderID {
				continue
			}

			FFS, exists := tran.allFFS[scorerProvider.ProviderID]
			if !exists {
				fmt.Println("FFS not found for scorer provider:", scorerProvider.ProviderID)
				continue
			}

			FF, exists := FFS[targetProvider.ProviderID]
			if !exists {
				fmt.Println("FF not found for target provider:", targetProvider.ProviderID)
				continue
			}

			dividend += math.Pow(FF-FFmu, 2)
		}
		FFsigma := math.Sqrt(dividend / populationN)

		// Calculate FFfinal
		sampleN := 0.0
		FFfinal := 0.0
		for _, scorerProvider := range tran.providerList {
			if targetProvider.ProviderID == scorerProvider.ProviderID {
				continue
			}

			FFS, exists := tran.allFFS[scorerProvider.ProviderID]
			if !exists {
				fmt.Println("FFS not found for scorer provider:", scorerProvider.ProviderID)
				continue
			}

			FF, exists := FFS[targetProvider.ProviderID]
			if !exists {
				fmt.Println("FF not found for target provider:", targetProvider.ProviderID)
				continue
			}

			if math.Abs(calculateZScore(FF, FFmu, FFsigma)) <= c.tau {
				FFfinal += FF
				sampleN += 1
			}

		}

		FFSfinal[targetProvider.ProviderID] = FFfinal / sampleN

		if FFSfinal[targetProvider.ProviderID] > highestFF {
			highestFF = FFSfinal[targetProvider.ProviderID]
			winner = targetProvider
		}
	}

	return FFSfinal, winner
}

func calculateZScore(FF float64, mu float64, sigma float64) float64 {
	return (FF - mu) / (sigma + math.SmallestNonzeroFloat64)
}

func calculateConsumerRating(actualUplink float64, actualDownlink float64, uplinkReq float64, downlinkReq float64) float64 {
	uplinkRating := math.Min(1, actualUplink/uplinkReq)
	downlinkRating := math.Min(1, actualDownlink/downlinkReq)
	return (uplinkRating + downlinkRating) / 2
}
