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

	p.mutex.Lock()
	entry, exists := p.peerScoreMatrix[payload.OriginID]
	p.mutex.Unlock()
	if !exists {
		p.mutex.Lock()
		p.peerScoreMatrix[payload.OriginID] = peerScore{
			uptime:           calculateUptime(currentTimestampMS, currentTimestampMS, p.params.KUptime),
			signalStrength:   calculateSignalStrength(payload.RSSI, p.params.KStrength),
			load:             calculateLoad(payload.ChannelUtilizationRate, p.params.KLoad),
			uplinkSpeed:      p.defaultPeerUplinkSpeed,
			downlinkSpeed:    p.defaultPeerDownlinkSpeed,
			lastPrice:        p.defaultPeerLastPrice,
			consumerFeedback: p.defaultPeerConsumerFeedback,
			beaconTimestamps: beaconTimestamps{
				initial: currentTimestampMS,
				last:    currentTimestampMS,
			},
		}
		p.mutex.Unlock()
		return
	}

	// Check if T_{n+1} - T_{n} > T_{limit}
	T_0 := &entry.beaconTimestamps.initial
	T_n := &entry.beaconTimestamps.last
	T_n1 := currentTimestampMS
	T_limit := p.params.BeaconTLimit

	// fmt.Println("initial:", *T_0)
	// fmt.Println("last:", *T_n)
	// fmt.Println("now:", T_n1)
	// fmt.Println("diff:", T_n1-*T_n)
	// fmt.Println("limit:", T_limit)
	if T_n1-*T_n > T_limit {
		*T_0 = T_n1
		*T_n = T_n1
	} else {
		*T_n = T_n1
	}

	entry.uptime = calculateUptime(*T_0, T_n1, p.params.KUptime)
	entry.signalStrength = calculateSignalStrength(payload.RSSI, p.params.KStrength)
	entry.load = calculateLoad(payload.ChannelUtilizationRate, p.params.KLoad)
	p.mutex.Lock()
	p.peerScoreMatrix[payload.OriginID] = entry
	p.mutex.Unlock()
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
		allFFS:          make(allFFS),
		customerQOS:     payload.customerQOS,
	}

	p.mutex.Lock()
	FFS := p.calculateFFS(p.transactions[payload.TransactionID.String()])

	// Save FFS calculation to current transaction's allFFS, indexed with self id
	p.transactions[payload.TransactionID.String()].allFFS[p.id] = FFS
	p.mutex.Unlock()

	fmt.Println("FFS calculation:", FFS)

	// TODO: Register timeout goroutine to send INFORM_VOTE

	for _, peer := range payload.PeerList {
		// Exclude itself
		if peer.ProviderID == p.id {
			continue
		}
		go func(peer peerInfo) {
			conn, err := net.Dial("tcp", peer.Address)
			if err != nil {
				fmt.Printf("failed to dial remote peer: %v\n", err)
				return
			}
			defer conn.Close()

			myPrice := p.price
			if p.isFaulty {
				myPrice = getRandomizedVal(p.price, 0.5, 0.1, 1)
			}

			// Build response
			response := requestVotePayload{
				PayloadMeta: PayloadMeta{
					PayloadType:   events.REQUEST_VOTE,
					TransactionID: payload.TransactionID,
					OriginID:      p.id,
					OriginAddress: p.address,
				},
				CandidateID: p.id,
				Price:       myPrice,
			}

			// Send REQUEST_VOTE event
			jsonResponse, err := json.Marshal(response)
			if err != nil {
				fmt.Printf("failed to marshal payload: %v\n", err)
				return
			}
			if _, err = conn.Write(jsonResponse); err != nil {
				fmt.Printf("failed to send REQUEST_VOTE from %s to address %s\n", p.id, peer.Address)
				return
			} else {
				fmt.Println("sent REQUEST_VOTE to", peer.Address)
				return
			}
		}(peer)
	}
}

// Handle REQUEST_VOTE event and respond by sending REPLY_VOTE event
func (p *provider) handleRequestVote(payload requestVotePayload) {
	p.mutex.Lock()
	senderAddress, err := p.getPeerAddressByID(payload.TransactionID.String(), payload.OriginID)
	p.mutex.Unlock()
	if err != nil {

		for retryCount := 0; retryCount < 5; retryCount++ {
			time.Sleep(time.Millisecond * 50)
			p.mutex.Lock()
			senderAddress, err = p.getPeerAddressByID(payload.TransactionID.String(), payload.OriginID)
			p.mutex.Unlock()
			if err == nil {
				break
			}
		}
		if err != nil {
			fmt.Printf("failed to obtain sender address: %v\n", err)
			return
		}
	}

	conn, err := net.Dial("tcp", senderAddress)
	if err != nil {
		fmt.Printf("failed to dial remote peer: %v\n", err)
		return
	}
	defer conn.Close()

	transactionID := payload.TransactionID.String()
	trans, exists := p.transactions[transactionID]
	if !exists {
		fmt.Printf("transaction doesn't exist: %s\n", transactionID)
		return
	}

	// TODO wait for all connections
	// Update peer's price and update its FF
	p.mutex.Lock()
	peerScore := p.peerScoreMatrix[payload.CandidateID]
	peerScore.lastPrice = payload.Price
	p.peerScoreMatrix[payload.CandidateID] = peerScore
	p.mutex.Unlock()

	// Check if all peers' prices have been received
	hasReceivedAll := false
	for {
		p.mutex.Lock()
		hasReceivedAll = true
		for _, peer := range trans.peerList {
			if peer.ProviderID == p.id {
				continue
			} else if p.peerScoreMatrix[peer.ProviderID].lastPrice == 0 {
				hasReceivedAll = false
				fmt.Println("haven't received price for provider:", peer.ProviderID)
			}
		}

		p.mutex.Unlock()
		if hasReceivedAll {
			break
		}
		time.Sleep(time.Millisecond * 50)
	}

	p.mutex.Lock()
	if p.isFaulty {
		trans.allFFS[p.id] = p.calculateFaultyFFS(p.transactions[transactionID])
	} else {
		trans.allFFS[p.id] = p.calculateFFS(p.transactions[transactionID])
	}

	FFS := trans.allFFS[p.id]
	p.mutex.Unlock()

	// // Save FFS calculation to current transaction's allFFS, indexed with self id (moved to handle BUY)
	// p.transactions[payload.TransactionID.String()].allFFS[p.id] = FFS

	// Build response
	response := replyVotePayload{
		PayloadMeta: PayloadMeta{
			PayloadType:   events.REPLY_VOTE,
			TransactionID: payload.TransactionID,
			OriginID:      p.id,
			OriginAddress: p.address,
		},
		FFS: FFS,
	}

	// Send REPLY_VOTE event
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		fmt.Printf("failed to marshal payload: %v\n", err)
		return
	}
	if _, err = conn.Write(jsonResponse); err != nil {
		fmt.Printf("failed to send REPLY_VOTE from %s to address %s: %v\n", p.id, payload.OriginAddress, err)
		return
	} else {
		fmt.Println("sent REPLY_VOTE to address:", payload.OriginAddress)
		return
	}
}

// Handle REPLY_VOTE event and respond by sending INFORM_VOTE event to consumer
func (p *provider) handleReplyVote(payload replyVotePayload) {
	transactionID := payload.TransactionID.String()

	peerCount := p.transactions[transactionID].peerCount
	peerID := payload.OriginID

	// Save current FFS to allFFS
	p.mutex.Lock()
	p.transactions[transactionID].allFFS[peerID] = payload.FFS
	allFFS := p.transactions[transactionID].allFFS

	// If haven't received all FFS yet
	// fmt.Println("len of allFFS:", allFFS)
	// fmt.Println("peer count:", peerCount)
	if len(allFFS) < peerCount {
		p.mutex.Unlock()
		fmt.Println("haven't received all FFS yet, current count:", len(allFFS))
		// TODO: create new goroutine with timeout to send
		return
	}
	p.mutex.Unlock()

	transaction, exists := p.transactions[transactionID]
	if !exists {
		fmt.Printf("transaction doesn't exist: %s\n", transactionID)
		return
	}

	fmt.Println("allFFS calculation:", allFFS)
	var FFSnew FFS
	if p.isFaulty {
		FFSnew = p.calculateFaultyFFSnew(transaction.peerList, transaction.allFFS)
	} else {
		FFSnew = p.calculateFFSnew(transaction.peerList, transaction.allFFS)
	}
	fmt.Println("FFSnew calculation:", FFSnew)

	// Build response
	response := informVotePayload{
		PayloadMeta: PayloadMeta{
			PayloadType:   events.INFORM_VOTE,
			TransactionID: payload.TransactionID,
			OriginID:      p.id,
			OriginAddress: p.address,
		},
		peerInfo: peerInfo{
			ProviderID:           p.id,
			Address:              p.address,
			Iperf3BaseServerPort: p.iperf3BaseServerPort,
			Iperf3ServerCount:    p.iperf3ServerCount,
		},
		FFSnew: FFSnew,
		Price:  p.price,
	}

	// TODO: This should be sent to consumer, not peers
	// Send INFORM_VOTE event to all peers concurrently
	conn, err := net.Dial("tcp", transaction.consumerAddress)
	if err != nil {
		fmt.Println("failed to dial consumer:", err)
		return
	}
	defer conn.Close()

	jsonResponse, err := json.Marshal(response)
	if err != nil {
		fmt.Println("failed to marshal payload", err)
		return
	}

	if _, err := conn.Write(jsonResponse); err != nil {
		fmt.Printf("failed to send INFORM_VOTE to consumer %s: %v\n", transaction.consumerAddress, err)
		return
	} else {
		fmt.Println("sent INFORM_VOTE to consumer:", transaction.consumerAddress)
		return
	}
}

func (p *provider) handleStartFlow(payload startFlowPayload) {
	transaction, exists := p.transactions[payload.TransactionID.String()]
	if !exists {
		fmt.Printf("transaction doesn't exist: %s\n", payload.TransactionID.String())
		return
	}

	if payload.Winner.ProviderID == p.id {
		// Increase active flow count, to calculate current channel utilization
		// rate (sent in beacon)
		p.activeFlowCount += 1
	}

	transaction.winner = payload.Winner

	// Reassign
	p.mutex.Lock()
	p.transactions[payload.TransactionID.String()] = transaction
	p.mutex.Unlock()
}

func (p *provider) handleTransactionEnd(payload transactionEndPayload) {
	transaction, exists := p.transactions[payload.TransactionID.String()]
	if !exists {
		fmt.Printf("transaction doesn't exist: %s\n", payload.TransactionID.String())
		return
	}

	if transaction.winner.ProviderID == p.id {
		// Decrease active flow count, to calculate current channel utilization
		// rate (sent in beacon)
		p.activeFlowCount -= 1
	}

	for _, peer := range transaction.peerList {
		if peer.ProviderID != transaction.winner.ProviderID {
			continue
		}

		p.mutex.Lock()
		peerScore := p.peerScoreMatrix[peer.ProviderID]

		peerScore.uplinkSpeed = payload.UplinkSpeed
		peerScore.downlinkSpeed = payload.DownlinkSpeed
		peerScore.consumerFeedback = calculateCustomerFeedback(peerScore.consumerFeedback,
			payload.Rating, p.params.Gamma)

		// Reassign
		p.peerScoreMatrix[peer.ProviderID] = peerScore
		p.mutex.Unlock()
	}
}

func (p *provider) handleGetProviderStats(conn net.Conn) {

	jsonResponse, err := json.Marshal(struct {
		ID               string          `json:"id"`
		Address          string          `json:"address"`
		Price            float64         `json:"price"`
		UplinkSpeed      float64         `json:"uplink_speed"`
		DownlinkSpeed    float64         `json:"downlink_speed"`
		Params           params          `json:"params"`
		PeerScoreMatrix  peerScoreMatrix `json:"peer_score_matrix"`
		Transactions     transactions    `json:"transactions"`
		Iperf3ServerPort string          `json:"iperf3_server_port"`
	}{
		ID:               p.id,
		Address:          p.address,
		Price:            p.price,
		UplinkSpeed:      p.uplinkSpeed,
		DownlinkSpeed:    p.downlinkSpeed,
		Params:           p.params,
		PeerScoreMatrix:  p.peerScoreMatrix,
		Transactions:     p.transactions,
		Iperf3ServerPort: p.iperf3BaseServerPort,
	})

	fmt.Println("provider stats:", string(jsonResponse))

	if err != nil {
		fmt.Printf("failed to marshal payload: %v\n", err)
		return
	}
	if n, err := conn.Write(jsonResponse); err != nil {
		fmt.Printf("failed to send PROVIDER_STATS from %s to address %s\n", p.id, conn.RemoteAddr().String())
		return
	} else {
		fmt.Println("n bytes written", n)
		return
	}
}

func (p *provider) getPeerAddressByID(transactionID string, peerID string) (string, error) {
	transaction, exists := p.transactions[transactionID]
	if !exists {
		return "", fmt.Errorf("transaction doesn't exist: %s", transactionID)
	}

	for _, peer := range transaction.peerList {
		if peer.ProviderID == peerID {
			return peer.Address, nil
		}
	}
	return "", fmt.Errorf("failed to find peer info: %s", peerID)
}
