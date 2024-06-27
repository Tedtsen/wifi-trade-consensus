package provider

import (
	"encoding/json"
	"fmt"
	"net"
	"time"
	"wifi-trade-consensus/internal/pkg/events"
	"wifi-trade-consensus/internal/pkg/payload"

	"github.com/google/uuid"
	"github.com/spf13/viper"
)

type PayloadMeta = payload.Meta

type beaconPayload struct {
	PayloadMeta
	ChannelUtilizationRate int `json:"channel_utilization_rate"` // 0-255
	RSSI                   int `json:"signal_strength"`          // Mocking field Received Signal Strength Indicator 0-255
}

type customerQOS struct {
	PriceConsumer         float64 `json:"price_consumer"`    // consumer price requirement
	UplinkSpeedConsumer   float64 `json:"uplink_consumer"`   // consumer uplink speed requirement
	DownlinkSpeedConsumer float64 `json:"downlink_consumer"` // consumer downlink speed requirement
	Mu                    float64 `json:"mu"`                // uplink weight
	Delta                 float64 `json:"delta"`             // downlink weight
	Epsilon               float64 `json:"epsilon"`           // price range multiplier limit
}

type buyPayload struct {
	PayloadMeta
	PeerList peers `json:"peer_list"`
	customerQOS
}

type requestVotePayload struct {
	PayloadMeta
	CandidateID uuid.UUID
	Price       float64
}

type allFFS map[string]FFS // index: provider id

type FFS map[string]float64 // index: provider id

type transactionEndPayload struct {
	PayloadMeta
	// TODO
}

type replyVotePayload struct {
	PayloadMeta
	FFS FFS `json:"FFS"`
}

type informVotePayload struct {
	PayloadMeta
	FFSnew FFS `json:"FFS_new"`
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

type transaction struct {
	transactionID   uuid.UUID
	transactionTime int64
	consumerID      uuid.UUID
	consumerAddress string
	peerList        peers
	peerCount       int
	allFFS          allFFS
	customerQOS     customerQOS
}

type transactions map[string]transaction

type params struct {
	BeaconTLimit  int64   `mapstructure:"beacon_t_limit"`  // 0 < beaconTLimit (ms) < 1000
	KUptime       float64 `mapstructure:"k_uptime"`        // 0 < kUptime < 1
	KLoad         float64 `mapstructure:"k_load"`          // 0 < kLoad < 1
	KStrength     float64 `mapstructure:"k_strength"`      // 0 < kStrength < 1
	Tau           float64 `mapstructure:"tau"`             // z-score threshold
	DefaultPeerFF float64 `mapstructure:"default_peer_ff"` // -1 < defaultPeerFF < 1
}

type options struct {
	Address       string  `mapstructure:"address"`
	Price         float64 `mapstructure:"price"`
	UplinkSpeed   float64 `mapstructure:"uplink_speed"`
	DownlinkSpeed float64 `mapstructure:"downlink_speed"`
	Params        params  `mapstructure:"params"`
}

type Provider struct {
	id              uuid.UUID
	address         string
	price           float64
	uplinkSpeed     float64
	downlinkSpeed   float64
	params          params
	peerScoreMatrix peerScoreMatrix
	transactions    transactions
}

// func NewParamsFromConfig() (*params, error) {
// 	params := params{}

// 	viper.SetConfigName("config") // Name of config file (without extension)
// 	viper.SetConfigType("json")   // REQUIRED if the config file does not have the extension in the name
// 	viper.AddConfigPath(".")      // Path to look for the config file in

// 	if err := viper.ReadInConfig(); err != nil {
// 		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
// 			// config not found, ignore err
// 			return nil, fmt.Errorf("config file not found")
// 		} else {
// 			// other errors, ignore err
// 			return nil, fmt.Errorf("failed to read config file: %w", err)
// 		}
// 	}

// 	err := viper.Unmarshal(&params)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to unmarshal config file: %w", err)
// 	}

// 	return &params, nil
// }

func NewParamsOptionsFromConfig() (*options, error) {
	params := params{}
	options := options{}

	viper.SetConfigName("config") // Name of config file (without extension)
	viper.SetConfigType("json")   // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath(".")      // Path to look for the config file in

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// config not found, ignore err
			return nil, fmt.Errorf("config file not found")
		} else {
			// other errors, ignore err
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	err := viper.Unmarshal(&params)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal params config file: %w", err)
	}

	err = viper.Unmarshal(&options)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal options config file: %w", err)
	}

	options.Params = params

	return &options, nil

}

func NewParams(beaconTLimit int64, kUptime float64, kLoad float64, kStrength float64, tau float64, defaultPeerFF float64) params {
	return params{
		BeaconTLimit:  beaconTLimit,
		KUptime:       kUptime,
		KLoad:         kLoad,
		KStrength:     kStrength,
		Tau:           tau,
		DefaultPeerFF: defaultPeerFF,
	}
}

func NewOptions(address string, price float64, uplinkSpeed float64, downlinkSpeed float64, params params) options {
	return options{
		Address:       address,
		Price:         price,
		UplinkSpeed:   uplinkSpeed,
		DownlinkSpeed: downlinkSpeed,
		Params:        params,
	}
}

func New(opt options) Provider {
	return Provider{
		id:            uuid.New(),
		address:       opt.Address,
		price:         opt.Price,
		uplinkSpeed:   opt.UplinkSpeed,
		downlinkSpeed: opt.DownlinkSpeed,
		params:        opt.Params,
	}
}

// Creates a new listener, this is a blocking function so wrapping the function
// call in a goroutine is required.
func (p *Provider) NewListener() error {
	l, err := net.Listen("tcp", p.address)
	if err != nil {
		return fmt.Errorf("failed to create new listener: %w", err)
	}
	defer l.Close()

	for {
		// Wait for a connection
		fmt.Println("listening for new connection at", p.address)
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("failed to accept new connection:", err)
		}
		// Concurrently handle the new connections
		go func(c net.Conn) {
			defer c.Close()

			payloadMeta := payload.Meta{}
			d := json.NewDecoder(c)
			err := d.Decode(&payloadMeta)
			if err != nil {
				fmt.Printf("failed to decode payload meta from %s: %v\n", c.RemoteAddr().String(), err)
				return
			}
			fmt.Printf("received payload meta from %s: %v\n", c.RemoteAddr(), payloadMeta)

			switch payloadMeta.PayloadType {

			// Handle BEACON event
			case events.BEACON:
				beaconPayload := beaconPayload{}
				if err := d.Decode(&beaconPayload); err != nil {
					fmt.Printf("failed to decode BEACON payload from %s: %v\n", c.RemoteAddr().String(), err)
					return
				}
				fmt.Printf("received BEACON payload from %s: %v\n", c.RemoteAddr().String(), beaconPayload)
				p.handleBeaconPayload(beaconPayload)

			// Handle BUY event
			case events.BUY:
				buyPayload := buyPayload{}
				if err := d.Decode(&buyPayload); err != nil {
					fmt.Printf("failed to decode BUY payload from %s: %v\n", c.RemoteAddr().String(), err)
					return
				}
				fmt.Printf("received BUY payload from %s: %v\n", c.RemoteAddr().String(), buyPayload)
				p.handleBuyEvent(buyPayload)

			// Handle REQUEST_VOTE event
			case events.REQUEST_VOTE:
				requestVotePayload := requestVotePayload{}
				if err := d.Decode(&requestVotePayload); err != nil {
					fmt.Printf("failed to decode REQUEST_VOTE payload from %s: %v\n", c.RemoteAddr().String(), err)
					return
				}
				fmt.Printf("received REQUEST_VOTE payload from %s: %v\n", c.RemoteAddr().String(), requestVotePayload)
				p.handleRequestVote(requestVotePayload)

			// Handle REPLY_VOTE event
			case events.REPLY_VOTE:
				replyVotePayload := replyVotePayload{}
				if err := d.Decode(&replyVotePayload); err != nil {
					fmt.Printf("failed to decode REPLY_VOTE payload from %s: %v\n", c.RemoteAddr().String(), err)
					return
				}
				fmt.Printf("received REPLY_VOTE payload from %s: %v\n", c.RemoteAddr().String(), replyVotePayload)
				p.handleReplyVote(replyVotePayload)

			// Handle TRANSACTION_END event
			case events.TRANSACTION_END:
				transactionEndPayload := transactionEndPayload{}
				if err := d.Decode(&transactionEndPayload); err != nil {
					fmt.Printf("failed to decode TRANSACTION_END payload from %s: %v\n", c.RemoteAddr().String(), err)
					return
				}
				fmt.Printf("received TRANSACTION_END payload from %s: %v\n", c.RemoteAddr().String(), transactionEndPayload)
				p.handleTransactionEnd(transactionEndPayload)

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
				// fmt.Println("sending beacon to:", peer.address)
				conn, err := net.Dial("tcp", peer.address)
				if err != nil {
					fmt.Printf("failed to send beacon to %s: %v\n", peer.address, err)
					continue
				}

				// Send beacon to each peer concurrently
				go func() {
					fmt.Fprint(conn, "test\n")
				}()
			}
		}
	}()
}

func NewMockPeerList(addresses []string) peers {
	peers := peers{}
	for _, address := range addresses {
		peers = append(peers, peerInfo{
			providerID: "mock-id",
			address:    address,
		})
	}
	return peers
}
