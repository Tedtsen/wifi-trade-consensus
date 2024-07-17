package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
	"wifi-trade-consensus/internal/pkg/events"
	"wifi-trade-consensus/internal/pkg/iperf3"
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
	PriceConsumer         float64 `json:"price"`    // consumer price requirement
	UplinkSpeedConsumer   float64 `json:"uplink"`   // consumer uplink speed requirement
	DownlinkSpeedConsumer float64 `json:"downlink"` // consumer downlink speed requirement
	Mu                    float64 `json:"mu"`       // uplink weight
	Delta                 float64 `json:"delta"`    // downlink weight
	Epsilon               float64 `json:"epsilon"`  // price range multiplier limit
}

type buyPayload struct {
	PayloadMeta
	PeerList peers `json:"provider_list"`
	customerQOS
}

type requestVotePayload struct {
	PayloadMeta
	CandidateID string  `json:"candidate_id"`
	Price       float64 `json:"price"`
}

type allFFS map[string]FFS // index: provider id

type FFS map[string]float64 // index: provider id

type transactionEndPayload struct {
	PayloadMeta
	Rating        float64 `json:"rating"`
	UplinkSpeed   float64 `json:"uplink_speed"`
	DownlinkSpeed float64 `json:"downlink_speed"`
}

type startFlowPayload struct {
	PayloadMeta
	Winner peerInfo `json:"winner"`
}

type replyVotePayload struct {
	PayloadMeta
	FFS FFS `json:"FFS"`
}

type informVotePayload struct {
	PayloadMeta
	peerInfo
	FFSnew FFS     `json:"FFS_new"`
	Price  float64 `json:"price"`
}

type informWinnerPayload struct {
	PayloadMeta
	winnerID uuid.UUID
}

type peerInfo struct {
	ProviderID           string `json:"provider_id"`
	Address              string `json:"address"`
	Iperf3BaseServerPort string `json:"iperf3_base_server_port"`
	Iperf3ServerCount    int    `json:"iperf3_server_count"`
}

type peers []peerInfo

type transaction struct {
	transactionID   uuid.UUID
	transactionTime int64
	consumerID      string
	consumerAddress string
	peerList        peers
	peerCount       int
	allFFS          allFFS
	customerQOS     customerQOS
	// Flow details
	winner        peerInfo
	flowStartTime int
	flowEndTime   int
}

type transactions map[string]transaction

type params struct {
	BeaconTLimit  int64   `mapstructure:"beacon_t_limit"`  // 0 < beaconTLimit (ms) < 1000
	KUptime       float64 `mapstructure:"k_uptime"`        // 0 < kUptime < 1
	KLoad         float64 `mapstructure:"k_load"`          // 0 < kLoad < 1
	KStrength     float64 `mapstructure:"k_strength"`      // 0 < kStrength < 1
	Tau           float64 `mapstructure:"tau"`             // z-score threshold
	Gamma         float64 `mapstructure:"gamma"`           // 0 < gamma < 1
	DefaultPeerFF float64 `mapstructure:"default_peer_ff"` // -1 < defaultPeerFF < 1
}

type options struct {
	ID                   string  `mapstructure:"id"`
	Address              string  `mapstructure:"address"`
	Iperf3BaseServerPort string  `mapstructure:"iperf3_base_server_port"`
	Iperf3ServerCount    int     `mapstructure:"iperf3_server_count"`
	Price                float64 `mapstructure:"price"`
	UplinkSpeed          float64 `mapstructure:"uplink_speed"`
	DownlinkSpeed        float64 `mapstructure:"downlink_speed"`
	Params               params  `mapstructure:"params"`
	// peer-score default values
	DefaultPeerUplinkSpeed      float64 `mapstructure:"default_peer_uplink_speed"`
	DefaultPeerDownlinkSpeed    float64 `mapstructure:"default_peer_downlink_speed"`
	DefaultPeerLastPrice        float64 `mapstructure:"default_peer_last_price"`
	DefaultPeerConsumerFeedback float64 `mapstructure:"default_peer_consumer_feedback"`
}

type provider struct {
	id                   string
	address              string
	price                float64
	uplinkSpeed          float64
	downlinkSpeed        float64
	params               params
	peerScoreMatrix      peerScoreMatrix
	transactions         transactions
	iperf3BaseServerPort string
	iperf3ServerCount    int
	iperf3Cmds           []*exec.Cmd
	mutex                sync.Mutex
	activeFlowCount      int
	// peer-score default values
	defaultPeerUplinkSpeed      float64
	defaultPeerDownlinkSpeed    float64
	defaultPeerLastPrice        float64
	defaultPeerConsumerFeedback float64
	// Beacon attributes
	channelUtilizationRate int // 0-255
	isFaulty               bool
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

func NewParamsOptionsFromConfigFile() (*options, error) {
	params := params{}
	options := options{}

	// Name of config file (without extension)
	// Get docker container's env variable
	if nodeNum := os.Getenv("node_num"); nodeNum != "" {
		viper.SetConfigName("config" + nodeNum)
	} else {
		viper.SetConfigName("config")
	}

	viper.SetConfigType("json") // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath(".")    // Path to look for the config file in
	viper.AddConfigPath("cmd/provider")

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

func New(opt options) provider {
	val := os.Getenv("is_faulty")
	isFaulty, err := strconv.ParseBool(val)
	if err != nil {
		fmt.Println("failed to parse environment variable is_faulty:", err)
		isFaulty = false
	}

	provider := provider{
		id:                   opt.ID,
		address:              opt.Address,
		price:                opt.Price,
		uplinkSpeed:          opt.UplinkSpeed,
		downlinkSpeed:        opt.DownlinkSpeed,
		params:               opt.Params,
		peerScoreMatrix:      make(peerScoreMatrix),
		transactions:         make(transactions),
		iperf3BaseServerPort: opt.Iperf3BaseServerPort,
		iperf3ServerCount:    opt.Iperf3ServerCount,
		activeFlowCount:      0,
		// peer-score default values
		defaultPeerUplinkSpeed:      opt.DefaultPeerUplinkSpeed,
		defaultPeerDownlinkSpeed:    opt.DefaultPeerDownlinkSpeed,
		defaultPeerLastPrice:        opt.DefaultPeerLastPrice,
		defaultPeerConsumerFeedback: opt.DefaultPeerConsumerFeedback,
		isFaulty:                    isFaulty,
	}

	// Register cleanup for interrupt signal i.e. Ctrl^c
	channel := make(chan os.Signal)
	signal.Notify(channel, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-channel
		if err := provider.cleanup(); err != nil {
			fmt.Println("failed to cleanup:", err)
		}
		os.Exit(1)
	}()

	return provider
}

// Creates a new listener, this is a blocking function so wrapping the function
// call in a goroutine is required.
func (p *provider) NewListener() error {
	l, err := net.Listen("tcp", p.address)
	if err != nil {
		return fmt.Errorf("failed to listen tcp address: %w", err)
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
		go func(conn net.Conn) {
			defer conn.Close()

			data, err := io.ReadAll(conn)
			if err != nil {
				fmt.Println("failed to read connection data:", err)
			}

			payloadMeta := payload.Meta{}
			err = json.Unmarshal(data, &payloadMeta)
			if err != nil {
				fmt.Printf("failed to unmarshal payload meta from %s: %v\n", conn.RemoteAddr().String(), err)
				return
			}
			// fmt.Printf("received payload meta from %s: %v\n", conn.RemoteAddr(), payloadMeta)

			switch payloadMeta.PayloadType {

			// Handle BEACON event
			case events.BEACON:
				beaconPayload := beaconPayload{}
				if err := json.Unmarshal(data, &beaconPayload); err != nil {
					fmt.Printf("failed to unmarshal BEACON payload from %s: %v\n", conn.RemoteAddr().String(), err)
					return
				}
				// fmt.Printf("received BEACON payload from %s: %v\n", conn.RemoteAddr().String(), beaconPayload)
				p.handleBeaconPayload(beaconPayload)

			// Handle BUY event
			case events.BUY:
				buyPayload := buyPayload{}
				if err := json.Unmarshal(data, &buyPayload); err != nil {
					fmt.Printf("failed to unmarshal BUY payload from %s: %v\n", conn.RemoteAddr().String(), err)
					return
				}
				fmt.Printf("received BUY payload from %s: %v\n", conn.RemoteAddr().String(), buyPayload)
				p.handleBuyEvent(buyPayload)

			// Handle REQUEST_VOTE event
			case events.REQUEST_VOTE:
				requestVotePayload := requestVotePayload{}
				if err := json.Unmarshal(data, &requestVotePayload); err != nil {
					fmt.Printf("failed to unmarshal REQUEST_VOTE payload from %s: %v\n", conn.RemoteAddr().String(), err)
					return
				}
				fmt.Printf("received REQUEST_VOTE payload from %s: %v\n", conn.RemoteAddr().String(), requestVotePayload)
				p.handleRequestVote(requestVotePayload)

			// Handle REPLY_VOTE event
			case events.REPLY_VOTE:
				replyVotePayload := replyVotePayload{}
				if err := json.Unmarshal(data, &replyVotePayload); err != nil {
					fmt.Printf("failed to unmarshal REPLY_VOTE payload from %s: %v\n", conn.RemoteAddr().String(), err)
					return
				}
				fmt.Printf("received REPLY_VOTE payload from %s: %v\n", conn.RemoteAddr().String(), replyVotePayload)
				p.handleReplyVote(replyVotePayload)

			// Handle START_FLOW event
			case events.START_FLOW:
				startFlowPayload := startFlowPayload{}
				if err := json.Unmarshal(data, &startFlowPayload); err != nil {
					fmt.Printf("failed to unmarshal START_FLOW payload from %s: %v\n", conn.RemoteAddr().String(), err)
					return
				}
				fmt.Printf("received START_FLOW payload from %s: %v\n", conn.RemoteAddr().String(), startFlowPayload)
				p.handleStartFlow(startFlowPayload)

			// Handle TRANSACTION_END event
			case events.TRANSACTION_END:
				transactionEndPayload := transactionEndPayload{}
				if err := json.Unmarshal(data, &transactionEndPayload); err != nil {
					fmt.Printf("failed to unmarshal TRANSACTION_END payload from %s: %v\n", conn.RemoteAddr().String(), err)
					return
				}
				fmt.Printf("received TRANSACTION_END payload from %s: %v\n", conn.RemoteAddr().String(), transactionEndPayload)
				p.handleTransactionEnd(transactionEndPayload)

			// Handle GET_PROVIDER_STATS debug event
			case events.GET_PROVIDER_STATS:
				fmt.Printf("received GET_PROVIDER_STATS from %s\n", conn.RemoteAddr().String())
				p.handleGetProviderStats(conn)

			// Handle unknown events
			default:
				fmt.Println("failed to determine event type:", payloadMeta)
				return
			}
		}(conn)
	}
}

func (p *provider) NewIperf3Server() error {
	cmds, err := iperf3.StartServers(p.iperf3BaseServerPort, p.iperf3ServerCount)
	if err != nil {
		return fmt.Errorf("failed to start iperf3 server: %w", err)
	}
	p.iperf3Cmds = cmds

	return nil
}

func (p *provider) cleanup() error {
	for _, cmd := range p.iperf3Cmds {
		if err := iperf3.StopServer(cmd); err != nil {
			return fmt.Errorf("failed to stop iperf3 server: %w", err)
		}
	}

	fmt.Println("cleanup ran, preparing to shutdown...")
	time.Sleep(time.Second * 3)
	return nil
}
