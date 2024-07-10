package provider

import (
	"fmt"
	"math"

	"github.com/google/uuid"
)

const oneDayInMS = 86400000 // 1 day = 86400000 milliseconds

func (p *provider) calculateFFSnew(peerList peers, allFFS allFFS) FFS {
	FFSnew := FFS{}

	for _, peer := range peerList {
		if peer.ProviderID == p.id {
			continue
		}
		FF := p.calculateFFnew(peer, peerList, allFFS)
		FFSnew[peer.ProviderID] = FF
	}

	return FFSnew
}

func (p *provider) calculateFFnew(targetPeer peerInfo, peerList peers, allFFS allFFS) float64 {
	// Calculate FFsum
	selfTargetPeerFF := allFFS[p.id][targetPeer.ProviderID]
	FFsum := selfTargetPeerFF // Init
	for _, peer := range peerList {
		if peer.ProviderID != targetPeer.ProviderID {
			FFsum += allFFS[peer.ProviderID][targetPeer.ProviderID]
		}
	}

	// Calculate FFmu (mean)
	populationN := len(peerList)
	FFmu := FFsum / float64(populationN)

	// Calculate FFsigma (standard deviation)
	dividend := math.Pow(selfTargetPeerFF-FFmu, 2)
	divisor := populationN
	for _, peer := range peerList {
		if peer.ProviderID != targetPeer.ProviderID {
			dividend += math.Pow(allFFS[peer.ProviderID][targetPeer.ProviderID]-FFmu, 2)
		}
	}
	FFsigma := math.Sqrt(dividend / float64(divisor))

	// Calculate FFnew
	sampleN := 0
	FFnew := 0.0
	if calculateZScore(selfTargetPeerFF, FFmu, FFsigma) <= p.params.Tau {
		FFnew = selfTargetPeerFF
		sampleN += 1
	}
	for _, peer := range peerList {
		zScore := calculateZScore(allFFS[peer.ProviderID][targetPeer.ProviderID], FFmu, FFsigma)
		if zScore <= p.params.Tau {
			FFnew += allFFS[peer.ProviderID][targetPeer.ProviderID]
			sampleN += 1
		}
	}
	FFnew = FFnew / float64(sampleN)

	return FFnew
}

func calculateZScore(FF float64, mu float64, sigma float64) float64 {
	return (FF - mu) / (sigma + math.SmallestNonzeroFloat64)
}

// Calculate FFS (Fittingness Factor Set) for all other providers except self
func (p *provider) calculateFFS(transaction transaction) map[string]float64 {
	customerQOS := transaction.customerQOS

	// Calculate FF for other providers, except self
	FFS := map[string]float64{}
	for _, peer := range transaction.peerList {
		peerScore, exists := p.peerScoreMatrix[peer.ProviderID]
		if !exists {
			fmt.Printf("failed to get peer score for provider %s\n", peer.ProviderID)
			FFS[peer.ProviderID] = p.params.DefaultPeerFF
			continue
		}

		fmt.Println("customerQOS:", customerQOS)
		fmt.Println("peerScore:", peerScore)

		PF := calculatePriceFittingness(customerQOS.PriceConsumer, peerScore.lastPrice, customerQOS.Epsilon)
		SF := calculateSpeedFittingness(customerQOS.UplinkSpeedConsumer, peerScore.uplinkSpeed, customerQOS.Mu,
			customerQOS.DownlinkSpeedConsumer, peerScore.downlinkSpeed, customerQOS.Delta)
		FF := calculateFittingnessFactor(PF, SF, peerScore.uptime, peerScore.load, peerScore.signalStrength, peerScore.consumerFeedback)
		FFS[peer.ProviderID] = FF
	}

	return FFS
}

func (p *provider) checkVoteStatus(transactionID uuid.UUIDs) bool {
	return false
}

func calculateUptime(T_0 int64, T_new int64, k float64) float64 {
	dividend := math.Min(float64(T_new)-float64(T_0), oneDayInMS)
	divisor := k * oneDayInMS
	exponent := -float64(dividend / divisor)
	return 1 / (1 + math.Pow(math.E, exponent))
}

func calculateLoad(channelUtilizationRate int, k float64) float64 {
	exponent := float64(channelUtilizationRate) / (k * 255)
	// TODO: remember to update the equation in the paper, the one below is
	// correct :D, the higher the channel utilization the, lower the load
	return 1 - (1 / (1 + math.Pow(math.E, exponent)))
}

func calculateSignalStrength(RSSI int, k float64) float64 {
	exponent := float64(RSSI) / (k * 255)
	return 1 / (1 + math.Pow(math.E, exponent))
}

func calculatePriceFittingness(priceConsumer float64, priceProvider float64, epsilon float64) float64 {
	dividend := (1 - (priceProvider / priceConsumer)) + (epsilon - 1)
	divisor := epsilon
	return dividend / divisor
}

func calculateSpeedFittingness(uplinkConsumer float64, uplinkProvider float64, mu float64, downlinkConsumer float64, downlinkProvider float64, delta float64) float64 {
	upDividend := math.Pow((uplinkProvider / uplinkConsumer), mu)
	upDivisor := 1 + math.Pow((uplinkProvider/uplinkConsumer), mu)
	downDividend := math.Pow((downlinkProvider / downlinkConsumer), mu)
	downDivisor := 1 + math.Pow((downlinkProvider/downlinkConsumer), mu)
	return (upDividend / upDivisor) * (downDividend / downDivisor)
}

func calculateFittingnessFactor(PF float64, SF float64, uptime float64, load float64, strength float64, feedback float64) float64 {
	return PF * SF * uptime * load * strength * feedback
}

func calculateCustomerFeedback(old float64, new float64, gamma float64) float64 {
	return (gamma * new) + ((1 - gamma) * old)
}

func calculateChannelUtilizationRate(activeFlowCount int) int {
	return int(math.Min(255, float64(25*activeFlowCount)))
}
