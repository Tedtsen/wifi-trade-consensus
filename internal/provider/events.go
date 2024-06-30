package provider

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"wifi-trade-consensus/internal/pkg/events"
)

func (p *provider) handleBeaconPayload(payload beaconPayload) {
	currentTimestampMS := time.Now().UnixMilli()

	entry, exists := p.peerScoreMatrix[payload.OriginID.String()]
	if !exists {
		p.peerScoreMatrix[payload.OriginID.String()] = peerScore{
			uptime:         calculateUptime(currentTimestampMS, currentTimestampMS, p.params.KUptime),
			signalStrength: calculateSignalStrength(payload.RSSI, p.params.KStrength),
			load:           calculateLoad(payload.ChannelUtilizationRate, p.params.KLoad),
			beaconTimestamps: beaconTimestamps{
				initial: currentTimestampMS,
				last:    currentTimestampMS,
			},
		}
		return
	}

	// Check if T_{n+1} - T_{n} > T_{limit}
	T_0 := &entry.beaconTimestamps.initial
	T_n := &entry.beaconTimestamps.last
	T_n1 := currentTimestampMS
	T_limit := p.params.BeaconTLimit

	if T_n1-*T_n > T_limit {
		*T_0 = T_n1
		*T_n = T_n1
	}

	entry.uptime = calculateUptime(*T_0, T_n1, p.params.KUptime)
	entry.signalStrength = calculateSignalStrength(payload.RSSI, p.params.KStrength)
	entry.load = calculateLoad(payload.ChannelUtilizationRate, p.params.KLoad)
	p.peerScoreMatrix[payload.OriginID.String()] = entry
}

// Handle BUY event and respond by sending REQUEST_VOTE event
func (p *provider) handleBuyEvent(payload buyPayload) {
	// Init new transaction record
	p.transactions[payload.TransactionID.String()] = transaction{
		transactionID:   payload.TransactionID,
		transactionTime: time.Now().UnixMilli(),
		consumerID:      payload.OriginID,
		consumerAddress: payload.OriginAddress,
		peerList:        payload.PeerList,
		peerCount:       len(payload.PeerList),
		customerQOS:     payload.customerQOS,
	}

	for _, peer := range payload.PeerList {
		go func(peer peerInfo) {
			conn, err := net.Dial("tcp", peer.address)
			if err != nil {
				fmt.Printf("failed to dial remote peer: %v\n", err)
				return
			}
			defer conn.Close()

			// Build response
			response := requestVotePayload{
				PayloadMeta: PayloadMeta{
					PayloadType:   events.REQUEST_VOTE,
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

func (p *provider) handleRequestVote(payload requestVotePayload) {
	conn, err := net.Dial("tcp", payload.OriginAddress)
	if err != nil {
		fmt.Printf("failed to dial remote peer: %v\n", err)
		return
	}
	defer conn.Close()

	transaction, exists := p.transactions[payload.TransactionID.String()]
	if !exists {
		fmt.Printf("transaction doesn't exist: %s", payload.TransactionID.String())
		return
	}

	FFS := p.calculateFFS(transaction)

	// Save FFS calculation to current transaction's allFFS, indexed with self id
	p.transactions[payload.TransactionID.String()].allFFS[p.id.String()] = FFS

	// Build response
	response := replyVotePayload{
		PayloadMeta: PayloadMeta{
			PayloadType:   events.REQUEST_VOTE,
			TransactionID: payload.TransactionID,
		},
		FFS: FFS,
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

func (p *provider) handleReplyVote(payload replyVotePayload) {
	transactionID := payload.TransactionID.String()

	peerCount := p.transactions[transactionID].peerCount
	peerList := p.transactions[transactionID].peerList
	peerID := payload.OriginID.String()

	// Save current FFS to allFFS
	p.transactions[transactionID].allFFS[peerID] = payload.FFS
	allFFS := p.transactions[transactionID].allFFS

	// If haven't received all FFS yet
	if len(allFFS) < peerCount {
		// TODO: create new goroutine with timeout to send
		return
	}

	transaction, exists := p.transactions[transactionID]
	if !exists {
		fmt.Printf("transaction doesn't exist: %s", transactionID)
		return
	}

	FFSnew := p.calculateFFSnew(transaction.peerList, transaction.allFFS)

	// Build response
	response := informVotePayload{
		PayloadMeta: PayloadMeta{
			PayloadType:   events.REQUEST_VOTE,
			TransactionID: payload.TransactionID,
		},
		FFSnew: FFSnew,
	}

	// Send INFORM_VOTE event to all peers concurrently
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
				fmt.Printf("failed to send INFORM_VOTE from %s to address %s\n", p.id, peer.address)
				return
			}
		}(peer)
	}
}

func (p *provider) handleTransactionEnd(payload transactionEndPayload) {
	// consumerAddress := p.transactions[payload.TransactionID.String()].consumerAddress
	// conn, err := net.Dial("tcp", consumerAddress)
	// if err != nil {
	// 	fmt.Printf("failed to dial remote consumer: %v\n", err)
	// 	return
	// }
	// defer conn.Close()

	// TODO

}
