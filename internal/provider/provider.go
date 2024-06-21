package provider

import (
	"encoding/json"
	"fmt"
	"net"
	"time"
	"wifi-trade-consensus/internal/pkg/events"
	"wifi-trade-consensus/internal/pkg/payload"

	"github.com/google/uuid"
)

type PayloadMeta = payload.Meta

type buyPayload struct {
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

type options struct {
	address string
}

type Provider struct {
	id              uuid.UUID
	address         string
	price           float64
	peerScoreMatrix map[string]map[string]int
	transactions
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

func New(option options) error {
	return nil
}

// Creates a new listener, this is a blocking function so wrapping the function
// call in a goroutine is required.
func (p *Provider) NewListener(option options) error {
	l, err := net.Listen("tcp", option.address)
	if err != nil {
		return fmt.Errorf("failed to create new listener: %w", err)
	}
	defer l.Close()

	for {
		// Wait for a connection
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("failed to accept new connection: %w", err)
		}
		// Concurrently handle the new connections
		go func(c net.Conn) {
			defer c.Close()

			payloadMeta := payload.Meta{}
			d := json.NewDecoder(c)
			err := d.Decode(&payloadMeta)
			if err != nil {
				fmt.Printf("failed to decode payload meta from %s: %w\n", c.RemoteAddr().String(), err)
				return
			}
			fmt.Printf("received payload meta from %s: %v\n", c.RemoteAddr(), payloadMeta)

			switch payloadMeta.PayloadType {
			case events.BEACON:

			// Handle BUY event
			case events.BUY:
				buyPayload := buyPayload{}
				if err := d.Decode(&buyPayload); err != nil {
					fmt.Printf("failed to decode BUY payload from %s: %w\n", c.RemoteAddr().String(), err)
					return
				}
				fmt.Printf("received BUY payload from %s: %v\n", c.RemoteAddr().String(), buyPayload)
				p.handleBuyEvent(buyPayload)

			// Handle REQUEST_VOTE event
			case events.REQUEST_VOTE:
				requestVotePayload := requestVotePayload{}
				if err := d.Decode(&requestVotePayload); err != nil {
					fmt.Printf("failed to decode REQUEST_VOTE payload from %s: %w\n", c.RemoteAddr().String(), err)
					return
				}
				fmt.Printf("received REQUEST_VOTE payload from %s: %v\n", c.RemoteAddr().String(), requestVotePayload)
				p.handleRequestVote(requestVotePayload)

			// Handle REPLY_VOTE event
			case events.REPLY_VOTE:
				replyVotePayload := replyVotePayload{}
				if err := d.Decode(&replyVotePayload); err != nil {
					fmt.Printf("failed to decode REPLY_VOTE payload from %s: %w\n", c.RemoteAddr().String(), err)
					return
				}
				fmt.Printf("received REPLY_VOTE payload from %s: %v\n", c.RemoteAddr().String(), replyVotePayload)
				p.handleReplyVote(replyVotePayload)

			// Handle unknown events
			default:
				fmt.Printf("failed to determine event type: %v", payloadMeta)
				return
			}
		}(conn)
	}
}

func NewBeaconEmitter(peerList peers, interval int) {
	// Run emitter concurrently
	go func() {
		for {
			// Wait for beacon interval
			time.Sleep(time.Millisecond * time.Duration(interval))
			for _, peer := range peerList {
				conn, _ := net.Dial("tcp", peer.address)
				// Send beacon to each peer concurrently
				go func() {
					fmt.Fprint(conn, "test\n")

				}()
			}
		}
	}()
}
