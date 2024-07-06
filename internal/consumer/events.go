package consumer

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"
	"wifi-trade-consensus/internal/pkg/events"
	"wifi-trade-consensus/internal/pkg/iperf3"

	"github.com/google/uuid"
)

func (c *consumer) triggerBuyEvent(triggerBuyPayload buyPayload) {
	// Send BUY concurrrently to all providers in list
	providerList := triggerBuyPayload.ProviderList
	qosRequirements := triggerBuyPayload.qosRequirements
	transactionID := uuid.New()

	// Init new transaction record
	c.transactions[transactionID.String()] = transaction{
		transactionID:   transactionID,
		transactionTime: time.Now().UnixMilli(),
		consumerID:      c.id,
		consumerAddress: c.address,
		providerList:    providerList,
		providerCount:   len(providerList),
		allFFS:          make(allFFS),
		qosRequirements: qosRequirements,
	}

	for _, provider := range providerList {
		go func(provider providerInfo) {
			conn, err := net.Dial("tcp", provider.Address)
			if err != nil {
				fmt.Printf("failed to dial provider: %v\n", err)
				return
			} else {
				fmt.Println("sending BUY to", provider.Address)
			}
			defer conn.Close()

			payload := buyPayload{
				PayloadMeta: PayloadMeta{
					PayloadType:   events.BUY,
					TransactionID: transactionID,
					OriginID:      c.id,
					OriginAddress: c.address,
				},
				ProviderList:    providerList,
				qosRequirements: qosRequirements,
			}

			jsonPayload, err := json.Marshal(payload)
			if err != nil {
				fmt.Printf("failed to marshal payload: %v\n", err)
				return
			}

			_, err = conn.Write(jsonPayload)
			if err != nil {
				fmt.Printf("failed to send BUY from %s to %s: %v\n", c.address, provider.Address, err)
			}
		}(provider)
	}
}

func (c *consumer) handleInformVote(payload informVotePayload) {
	// TODO: check logic and steps
	transactionID := payload.TransactionID.String()

	providerCount := c.transactions[transactionID].providerCount
	providerList := c.transactions[transactionID].providerList
	allFFS := c.transactions[transactionID].allFFS

	c.mutex.Lock()
	allFFS[payload.OriginID] = payload.FFSnew
	for idx, provider := range providerList {
		if provider.ProviderID == payload.OriginID {
			c.transactions[transactionID].providerList[idx] = payload.providerInfo
		}
	}

	if len(allFFS) < providerCount {
		fmt.Printf("received inform vote for transaction %s, but allFFS count is not enough, have %d want %d\n",
			transactionID, len(allFFS), providerCount)
		// TODO: create new goroutine with timeout
		c.mutex.Unlock()
		return
	} else {
		fmt.Printf("received all inform votes for transaction %s, have %d want %d\n",
			transactionID, len(allFFS), providerCount)
	}
	c.mutex.Unlock()

	transaction, exists := c.transactions[transactionID]
	if !exists {
		fmt.Printf("transaction doesn't exist: %s", transactionID)
		return
	}

	FFSfinal := FFS{}
	winner := providerInfo{}
	highestFF := -10.0
	for _, targetProvider := range providerList {
		divisor := 0.0
		for _, scorerProvider := range providerList {
			if targetProvider.ProviderID == scorerProvider.ProviderID {
				continue
			}

			FFS, exists := transaction.allFFS[scorerProvider.ProviderID]
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

	fmt.Println("FFSfinal:", FFSfinal)

	// Send START_FLOW event to all peers concurrently
	for _, provider := range transaction.providerList {
		go func(provider providerInfo) {
			conn, err := net.Dial("tcp", provider.Address)
			if err != nil {
				fmt.Printf("failed to dial provider: %v\n", err)
			}

			payload := startFlowPayload{
				PayloadMeta: PayloadMeta{
					PayloadType:   events.START_FLOW,
					TransactionID: transaction.transactionID,
					OriginID:      c.id,
					OriginAddress: c.address,
				},
				Winner: winner,
			}

			jsonPayload, err := json.Marshal(payload)
			if err != nil {
				fmt.Printf("failed to marshal payload: %v\n", err)
				return
			}

			_, err = fmt.Fprint(conn, jsonPayload)
			if err != nil {
				fmt.Printf("failed to send BUY from %s to %s: %v\n", c.address, provider.Address, err)
			}
		}(provider)
	}

	fmt.Println("sending iperf3 stream to winner:", winner)

	// Get provider iperf3 server ip and port
	winnerIP := strings.Split(winner.Address, ":")[0]

	go func() {
		_, err := iperf3.StartStream(winnerIP, winner.Iperf3BaseServerPort, winner.Iperf3ServerCount, "10G")
		if err != nil {
			fmt.Println("failed to send stream to winner:", err)
		}
	}()
}

// func (c *consumer) sendTransactionEnd(provider Provider) {

// }
