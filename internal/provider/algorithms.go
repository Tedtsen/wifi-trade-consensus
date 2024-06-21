package provider

import (
	"math"

	"github.com/google/uuid"
)

const oneDayInMS = 86400000 // 1 day = 86400000 milliseconds

func (p *Provider) decideVote(peer peerInfo) (bool, error) {
	// p.getPeerScore(uuid.UUID{})

	// Score comparison logic
	return true, nil
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
