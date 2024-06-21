package provider

import (
	"encoding/json"
	"fmt"
	"net"
)

// Handle BUY event and respond by sending REQUEST_VOTE event
func (p *Provider) handleBuyEvent(payload BuyPayload) {
	for _, peer := range payload.PeerList {
		go func(peer peerInfo) {
			conn, err := net.Dial("tcp", peer.address)
			if err != nil {
				fmt.Printf("failed to dial remote peer: %v\n", err)
				return
			}
			defer conn.Close()

			// Init new transaction record
			p.transactions[payload.TransactionID.String()] = transaction{
				transactionID:   payload.TransactionID,
				consumerID:      payload.OriginID,
				consumerAddress: payload.OriginAddress,
				peerList:        payload.PeerList,
				peerCount:       len(payload.PeerList),
				votes: votes{
					p.id.String(): true,
				},
			}

			// Build response
			response := requestVotePayload{
				PayloadMeta: PayloadMeta{
					PayloadType:   REQUEST_VOTE,
					TransactionID: payload.TransactionID,
				},
				CandidateID: p.id,
				Price:       p.price,
			}

			// Send REQUEST_VOTE event
			jsonResponse, err := json.Marshal(response)
			if err != nil {
				fmt.Printf("failed to marshal payload: %v\n", err)
				return
			}
			if _, err = fmt.Fprint(conn, jsonResponse); err != nil {
				fmt.Printf("failed to send REQUEST_VOTE from %s to address %s\n", p.id, peer.address)
				return
			}
		}(peer)
	}
}

func (p *Provider) handleRequestVote(payload requestVotePayload) {
	conn, err := net.Dial("tcp", payload.OriginAddress)
	if err != nil {
		fmt.Printf("failed to dial remote peer: %v\n", err)
		return
	}
	defer conn.Close()

	decision, err := p.decideVote(peerInfo{})

	// Build response
	response := replyVotePayload{
		PayloadMeta: PayloadMeta{
			PayloadType:   REQUEST_VOTE,
			TransactionID: payload.TransactionID,
		},
		Decision: decision,
	}

	// Send REPLY_VOTE event
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		fmt.Printf("failed to marshal payload: %v\n", err)
		return
	}
	if _, err = fmt.Fprint(conn, string(jsonResponse)); err != nil {
		fmt.Printf("failed to send REPLY_VOTE from %s to address %s\n", p.id, payload.OriginAddress)
		return
	}
}

func (p *Provider) handleReplyVote(payload replyVotePayload) {

	transactionID := payload.TransactionID.String()
	voterID := payload.OriginID.String()

	p.transactions[transactionID].votes[voterID] = payload.Decision
	// Create copy
	votes := p.transactions[transactionID].votes
	peerCount := p.transactions[transactionID].peerCount
	peerList := p.transactions[transactionID].peerList

	// If not all voted
	if len(votes) < peerCount {
		// TODO: create new goroutine with timeout to send
		return
	}

	votedForCount := 0
	for _, vote := range votes {
		if vote == true {
			votedForCount++
		}
	}
	// If all voted but VOTED_FOR count is less than (n/2 + 1)
	if votedForCount < (peerCount/2 + 1) {
		return
	}

	// Build response
	response := declareVictoryPayload{
		PayloadMeta: PayloadMeta{
			PayloadType:   REQUEST_VOTE,
			TransactionID: payload.TransactionID,
		},
		votes: votes,
	}

	// Send DECLARE_VICTORY event to all peers concurrently
	for _, peer := range peerList {
		go func(peer peerInfo) {
			conn, err := net.Dial("tcp", peer.address)
			if err != nil {
				fmt.Printf("failed to dial remote peer: %v\n", err)
				return
			}
			defer conn.Close()

			jsonResponse, err := json.Marshal(response)
			if err != nil {
				fmt.Printf("failed to marshal payload: %v\n", err)
				return
			}
			if _, err = fmt.Fprint(conn, string(jsonResponse)); err != nil {
				fmt.Printf("failed to send DECLARE_VICTORY from %s to address %s\n", p.id, peer.address)
				return
			}
		}(peer)
	}
}

func (p *Provider) handleDeclareVictory(payload declareVictoryPayload) {
	consumerAddress := p.transactions[payload.TransactionID.String()].consumerAddress
	conn, err := net.Dial("tcp", consumerAddress)
	if err != nil {
		fmt.Printf("failed to dial remote consumer: %v\n", err)
		return
	}
	defer conn.Close()

	// Build response
	response := informWinnerPayload{
		PayloadMeta: PayloadMeta{
			PayloadType:   INFORM_WINNER,
			TransactionID: payload.TransactionID,
			OriginID:      p.id,
			OriginAddress: p.address,
		},
		winnerID: payload.OriginID,
	}

	// Send INFORM_WINNER event to consumer
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		fmt.Printf("failed to marshal payload: %v\n", err)
		return
	}
	if _, err = fmt.Fprint(conn, string(jsonResponse)); err != nil {
		fmt.Printf("failed to send INFORM_WINNER from %s to address %s\n", p.id, consumerAddress)
		return
	}
}
