package consumer

import (
	"encoding/json"
	"fmt"
	"net"
	"wifi-trade-consensus/internal/pkg/events"

	"github.com/google/uuid"
)

func (c *consumer) triggerBuyEvent(requirements qosRequirements, providerList providers) {
	// Send BUY concurrrently to all providers in list
	for _, provider := range providerList {
		go func(provider providerInfo) {
			conn, err := net.Dial("tcp", provider.address)
			if err != nil {
				fmt.Printf("failed to dial provider: %v\n", err)
			}

			payload := buyPayload{
				PayloadMeta: PayloadMeta{
					PayloadType:   events.BUY,
					TransactionID: uuid.New(),
					OriginID:      c.id,
					OriginAddress: c.address,
				},
				ProviderList:    providerList,
				qosRequirements: requirements,
			}

			jsonPayload, err := json.Marshal(payload)
			if err != nil {
				fmt.Printf("failed to marshal payload: %v\n", err)
				return
			}

			_, err = fmt.Fprint(conn, jsonPayload)
			if err != nil {
				fmt.Printf("failed to send BUY from %s to %s: %v\n", c.address, provider.address, err)
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

	if len(allFFS) < providerCount {
		// TODO: create new goroutine with timeout
		return
	}

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
			if targetProvider.providerID == scorerProvider.providerID {
				continue
			}

			FFS, exists := transaction.allFFS[scorerProvider.providerID]
			if !exists {
				fmt.Println("FFS not found for scorer provider:", scorerProvider.providerID)
				continue
			}

			FF, exists := FFS[targetProvider.providerID]
			if !exists {
				fmt.Println("FF not found for targer provider:", targetProvider.providerID)
				continue
			}

			_, exists = FFSfinal[targetProvider.providerID]
			if !exists {
				FFSfinal[targetProvider.providerID] = FF
				divisor += 1
			} else {
				FFSfinal[targetProvider.providerID] += FF
				divisor += 1
			}
		}
		FFSfinal[targetProvider.providerID] /= divisor

		if FFSfinal[targetProvider.providerID] > highestFF {
			highestFF = FFSfinal[targetProvider.providerID]
			winner = targetProvider
		}
	}

	// Send START_FLOW event to all peers concurrently
	for _, provider := range transaction.providerList {
		go func(provider providerInfo) {
			conn, err := net.Dial("tcp", provider.address)
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
				fmt.Printf("failed to send BUY from %s to %s: %v\n", c.address, provider.address, err)
			}
		}(provider)
	}
}

// func (c *consumer) sendTransactionEnd(provider Provider) {

// }
