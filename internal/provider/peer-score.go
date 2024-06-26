package provider

type beaconTimestamps struct {
	initial int64 // T_0
	last    int64 // T_n
}

type peerScore struct {
	uptime           float64
	load             float64
	signalStrength   float64
	uplinkSpeed      float64
	downlinkSpeed    float64
	lastPrice        float64
	consumerFeedback float64
	beaconTimestamps beaconTimestamps
}

type peerScoreMatrix map[string]peerScore

// func (p *Provider) getPeerScore(peerID uuid.UUID) int {
// 	return 1
// }
