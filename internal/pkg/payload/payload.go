package payload

import "github.com/google/uuid"

type PayloadType int

const (
	BEACON int = iota
	BUY
	REQUEST_VOTE
	REPLY_VOTE
	DECLARE_VICTORY
	INFORM_WINNER
	TRANSACTION_END
)

type PayloadMeta struct {
	PayloadType   int       `json:"type"`
	TransactionID uuid.UUID `json:"transaction_id"`
	OriginID      uuid.UUID `json:"origin_id"`
	OriginAddress string    `json:"origin_address"`
	// Size                  int       `json:"size"`
	// Utilization           int       `json:"utilization"`
}

type BuyPayload struct {
	PayloadMeta
	PeerList peers `json:"peer_list"`
}

type requestVotePayload struct {
	PayloadMeta
	CandidateID uuid.UUID
	Price       float64
}

type replyVotePayload struct {
	PayloadMeta
	Decision bool `json:"decision"`
}

type declareVictoryPayload struct {
	PayloadMeta
	votes
}

type informWinnerPayload struct {
	PayloadMeta
	winnerID uuid.UUID
}

type peerInfo struct {
	providerID string
	address    string
}

type peers []peerInfo

type transactions map[string]transaction

type transaction struct {
	transactionID   uuid.UUID
	consumerID      uuid.UUID
	consumerAddress string
	peerList        peers
	peerCount       int
	votes           map[string]bool
}

type votes map[string]bool
