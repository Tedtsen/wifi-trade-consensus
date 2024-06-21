package provider

import "github.com/google/uuid"

type peerScoreMatrix struct {
}

func (p *Provider) getPeerScore(peerID uuid.UUID) int {
	return 1
}
