package consumer

import (
	"fmt"
	"math"
)

func calculateFFSfinal(tran transaction) (FFS, providerInfo) {
	FFSfinal := FFS{}
	winner := providerInfo{}
	highestFF := -10.0
	for _, targetProvider := range tran.providerList {
		divisor := 0.0
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
				fmt.Println("FF not found for targer provider:", targetProvider.ProviderID)
				continue
			}

			_, exists = FFSfinal[targetProvider.ProviderID]
			if !exists {
				FFSfinal[targetProvider.ProviderID] = FF
				divisor += 1
			} else {
				FFSfinal[targetProvider.ProviderID] += FF
				divisor += 1
			}
		}
		FFSfinal[targetProvider.ProviderID] /= divisor

		if FFSfinal[targetProvider.ProviderID] > highestFF {
			highestFF = FFSfinal[targetProvider.ProviderID]
			winner = targetProvider
		}
	}

	return FFSfinal, winner
}

func calculateConsumerRating(actualUplink float64, actualDownlink float64, uplinkReq float64, downlinkReq float64) float64 {
	uplinkRating := math.Min(1, actualUplink/uplinkReq)
	downlinkRating := math.Min(1, actualDownlink/downlinkReq)
	return (uplinkRating + downlinkRating) / 2
}
