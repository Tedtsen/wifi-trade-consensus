package provider

import (
	"fmt"
	"math"

	"github.com/google/uuid"
)

const oneDayInMS = 86400000 // 1 day = 86400000 milliseconds

func (p *Provider) calculateFFSnew(peerList peers, allFFS allFFS) FFS {
	FFSnew := FFS{}

	for _, peer := range peerList {
		FF := p.calculateFFnew(peer, peerList, allFFS)
		FFSnew[peer.providerID] = FF
	}

	return FFSnew
}

func (p *Provider) calculateFFnew(targetPeer peerInfo, peerList peers, allFFS allFFS) float64 {
	// Calculate FFsum
	selfTargetPeerFF := allFFS[p.id.String()][targetPeer.providerID]
	FFsum := selfTargetPeerFF // Init
	for _, peer := range peerList {
		if peer.providerID != targetPeer.providerID {
			FFsum += allFFS[peer.providerID][targetPeer.providerID]
		}
	}

	// Calculate FFmu (mean)
	populationN := len(peerList)
	FFmu := FFsum / float64(populationN)

	// Calculate FFsigma (standard deviation)
	dividend := math.Pow(selfTargetPeerFF-FFmu, 2)
	divisor := populationN
	for _, peer := range peerList {
		if peer.providerID != targetPeer.providerID {
			dividend += math.Pow(allFFS[peer.providerID][targetPeer.providerID]-FFmu, 2)
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
		zScore := calculateZScore(allFFS[peer.providerID][targetPeer.providerID], FFmu, FFsigma)
		if zScore <= p.params.Tau {
			FFnew += allFFS[peer.providerID][targetPeer.providerID]
			sampleN += 1
		}
	}
	FFnew = FFnew / float64(sampleN)

	return FFnew
}

func calculateZScore(FF float64, mu float64, sigma float64) float64 {
	return (FF - mu) / (sigma)
}

// Calculate FFS (Fittingness Factor Set or plural-prefix) for all other providers except self
func (p *Provider) calculateFFS(transaction transaction) map[string]float64 {
	customerQOS := transaction.customerQOS

	// Calculate FF for other providers, except self
	FFS := map[string]float64{}
	for _, peer := range transaction.peerList {
		peerScore, exists := p.peerScoreMatrix[peer.providerID]
		if !exists {
			fmt.Printf("failed to get peer score for provider %s\n", peer.providerID)
			FFS[peer.providerID] = p.params.DefaultPeerFF
			continue
		}

		PF := calculatePriceFittingness(customerQOS.PriceConsumer, peerScore.lastPrice, customerQOS.Epsilon)
		SF := calculateSpeedFittingness(customerQOS.UplinkSpeedConsumer, peerScore.uplinkSpeed, customerQOS.Mu,
			customerQOS.DownlinkSpeedConsumer, peerScore.downlinkSpeed, customerQOS.Delta)
		FF := calculateFittingnessFactor(PF, SF, peerScore.uptime, peerScore.load, peerScore.signalStrength, peerScore.consumerFeedback)
		FFS[peer.providerID] = FF
	}

	return FFS
}

func (p *Provider) checkVoteStatus(transactionID uuid.UUIDs) bool {
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
	return 1 / (1 + math.Pow(math.E, exponent))
}

func calculateSignalStrength(RSSI int, k float64) float64 {
	exponent := float64(RSSI) / (k * 255)
	return 1 / (1 + math.Pow(math.E, exponent))
}

func calculatePriceFittingness(priceConsumer float64, priceProvider float64, epsilon float64) float64 {
	dividend := (1 - (priceConsumer / priceProvider)) + (epsilon - 1)
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
