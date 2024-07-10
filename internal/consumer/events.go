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
			c.transactions[transactionID].providerList[idx].Price = payload.Price
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

	// Calculate FFSfinal and determine winner
	FFSfinal, winner := calculateFFSfinal(transaction)

	fmt.Println("FFSfinal:", FFSfinal)

	// Send START_FLOW event to all peers concurrently
	for _, provider := range transaction.providerList {
		go func(provider providerInfo) {
			conn, err := net.Dial("tcp", provider.Address)
			if err != nil {
				fmt.Printf("failed to dial provider: %v\n", err)
			}
			defer conn.Close()

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

			_, err = conn.Write(jsonPayload)
			if err != nil {
				fmt.Printf("failed to send START_FLOW from %s to %s: %v\n", c.address, provider.Address, err)
			}
		}(provider)
	}

	fmt.Println("sending iperf3 streams (forward/reverse) to winner:", winner)

	// Get provider iperf3 server ip and port
	winnerIP := strings.Split(winner.Address, ":")[0]

	fmt.Println("winner ip:", winnerIP)
	fmt.Println("base server port:", winner.Iperf3BaseServerPort)
	upChannel := make(chan *iperf3.Results)
	go func(upChannel chan *iperf3.Results) {
		iperf3Res, err := iperf3.StartStream(winnerIP, winner.Iperf3BaseServerPort, winner.Iperf3ServerCount,
			"1G", payload.TransactionID.String())
		if err != nil {
			fmt.Println("failed to send stream to winner:", err)
			upChannel <- nil
		}
		upChannel <- iperf3Res
	}(upChannel)

	downChannel := make(chan *iperf3.Results)
	go func(downChannel chan *iperf3.Results) {
		iperf3Res, err := iperf3.StartReverseStream(winnerIP, winner.Iperf3BaseServerPort, winner.Iperf3ServerCount,
			"1G", payload.TransactionID.String())
		if err != nil {
			fmt.Println("failed to send reverse stream to winner:", err)
			downChannel <- nil
		}
		downChannel <- iperf3Res
	}(downChannel)

	uplinkBitsPerSecond := 0.0
	downlinkBitsPerSecond := 0.0
	uplinkResults := <-upChannel
	downlinkResults := <-downChannel
	fmt.Println("received downlink results:", downlinkResults)
	if uplinkResults != nil {
		uplinkBitsPerSecond = uplinkResults.End.SumSent.BitsPerSecond
	}
	if downlinkResults != nil {
		downlinkBitsPerSecond = downlinkResults.End.SumSent.BitsPerSecond
	}

	// Calculate upload and downlink speeds
	actualUplink := uplinkBitsPerSecond / 8 / 1000000
	actualDownlink := downlinkBitsPerSecond / 8 / 1000000

	// Record metric, to be written into output file for data analysis
	for idx, provider := range providerList {
		if provider.ProviderID == winner.ProviderID {
			transaction.FlowMetrics.Price = transaction.providerList[idx].Price
		}
	}
	transaction.FlowMetrics.PriceConsumer = transaction.qosRequirements.PriceConsumer
	transaction.FlowMetrics.AverageUplinkSpeed = actualUplink
	transaction.FlowMetrics.AverageDownlinkSpeed = actualDownlink
	transaction.FlowMetrics.ProviderInfo = winner
	c.transactions[transactionID] = transaction

	uplinkRequirement := c.qosRequirements.UplinkSpeedConsumer
	downlinkRequirement := c.qosRequirements.DownlinkSpeedConsumer
	consumerRating := calculateConsumerRating(actualUplink, actualDownlink, uplinkRequirement, downlinkRequirement)

	// Send TRANSACTION_END to all providers, including rating for current transaction
	for _, provider := range transaction.providerList {
		go func(provider providerInfo) {
			conn, err := net.Dial("tcp", provider.Address)
			if err != nil {
				fmt.Printf("failed to dial provider: %v\n", err)
			}
			defer conn.Close()

			transactionEndPayload := transactionEndPayload{
				PayloadMeta: PayloadMeta{
					PayloadType:   events.TRANSACTION_END,
					TransactionID: payload.TransactionID,
					OriginID:      c.id,
					OriginAddress: c.address,
				},
				Rating:        consumerRating,
				UplinkSpeed:   actualUplink,
				DownlinkSpeed: actualDownlink,
			}

			jsonPayload, err := json.Marshal(transactionEndPayload)
			if err != nil {
				fmt.Printf("failed to marshal payload: %v\n", err)
				return
			}

			_, err = conn.Write(jsonPayload)
			if err != nil {
				fmt.Printf("failed to send TRANSACTION_END from %s to %s: %v\n", c.address, provider.Address, err)
			}
		}(provider)
	}
}

// func (c *consumer) sendTransactionEnd(provider Provider) {

// }
