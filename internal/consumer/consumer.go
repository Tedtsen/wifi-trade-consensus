package consumer

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
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
	Winner providerInfo `json:"winner"`
}

type buyPayload struct {
	PayloadMeta
	ProviderList providers `json:"provider_list"`
	qosRequirements
}

type informVotePayload struct {
	PayloadMeta
	providerInfo
	FFSnew FFS     `json:"FFS_new"`
	Price  float64 `json:"price"`
}

type consumer struct {
	id                   string
	address              string
	transactions         transactions
	qosRequirements      qosRequirements
	iperf3BaseServerPort string
	iperf3ServerCount    int
	iperf3Cmds           []*exec.Cmd
	mutex                sync.Mutex
	outputDir            string
}

type transactions map[string]transaction

type transaction struct {
	transactionID   uuid.UUID
	transactionTime int64
	consumerID      string
	consumerAddress string
	providerList    providers
	providerCount   int
	qosRequirements qosRequirements
	allFFS          allFFS
	FlowMetrics     flowMetrics `json:"flow_metrics"`
}

type providers []providerInfo

type providerInfo struct {
	ProviderID           string  `json:"provider_id"`
	Address              string  `json:"address"`
	Iperf3BaseServerPort string  `json:"iperf3_base_server_port"`
	Iperf3ServerCount    int     `json:"iperf3_server_count"`
	Price                float64 `json:"price"`
}

// mapstructure tags are for config file mapping
// json tags are for tcp body mapping
type options struct {
	ID                   string          `mapstructure:"id" json:"id"`
	Address              string          `mapstructure:"address" json:"address"`
	Iperf3BaseServerPort string          `mapstructure:"iperf3_base_server_port" json:"iperf3_base_server_port"`
	Iperf3ServerCount    int             `mapstructure:"iperf3_server_count" json:"iperf3_server_count"`
	QOSRequirements      qosRequirements `mapstructure:"params" json:"params"`
	OutputDir            string          `mapstructure:"output_dir" json:"output_dir"`
}

type qosRequirements struct {
	PriceConsumer         float64 `mapstructure:"price" json:"price"`       // consumer price requirement
	UplinkSpeedConsumer   float64 `mapstructure:"uplink" json:"uplink"`     // consumer uplink speed requirement
	DownlinkSpeedConsumer float64 `mapstructure:"downlink" json:"downlink"` // consumer downlink speed requirement
	Mu                    float64 `mapstructure:"mu" json:"mu"`             // uplink weight
	Delta                 float64 `mapstructure:"delta" json:"delta"`       // downlink weight
	Epsilon               float64 `mapstructure:"epsilon" json:"epsilon"`   // price range multiplier limit
}

type flowMetrics struct {
	ProviderInfo         providerInfo `json:"provider_info"`
	Price                float64      `json:"price"`
	PriceConsumer        float64      `json:"price_consumer"`
	AverageUplinkSpeed   float64      `json:"average_uplink"`
	AverageDownlinkSpeed float64      `json:"average_downlink"`
}

func NewOptionsFromConfigFile() (*options, error) {
	options := options{}
	qosRequirements := qosRequirements{}

	viper.SetConfigName("config") // Name of config file (without extension)
	viper.SetConfigType("json")   // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath(".")      // Path to look for the config file in
	viper.AddConfigPath("cmd/consumer")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// config not found, ignore err
			return nil, fmt.Errorf("config file not found")
		} else {
			// other errors, ignore err
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	err := viper.Unmarshal(&qosRequirements)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal qosRequirements config file: %w", err)
	}

	err = viper.Unmarshal(&options)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal options config file: %w", err)
	}

	options.QOSRequirements = qosRequirements

	return &options, nil
}

func New(opt options) consumer {
	consumer := consumer{
		id:                   opt.ID,
		address:              opt.Address,
		transactions:         make(transactions),
		qosRequirements:      opt.QOSRequirements,
		iperf3BaseServerPort: opt.Iperf3BaseServerPort,
		iperf3ServerCount:    opt.Iperf3ServerCount,
		outputDir:            opt.OutputDir,
	}

	// Register cleanup for interrupt signal i.e. Ctrl^c
	channel := make(chan os.Signal)
	signal.Notify(channel, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-channel
		if err := consumer.persistResults(); err != nil {
			fmt.Println("failed to persist results:", err)
		}
		if err := consumer.cleanup(); err != nil {
			fmt.Println("failed to cleanup:", err)
		}
		fmt.Println("preparing to shutdown...")
		time.Sleep(time.Second * 3)
		os.Exit(1)
	}()

	return consumer
}

func (c *consumer) NewListener() error {
	l, err := net.Listen("tcp", c.address)
	if err != nil {
		return fmt.Errorf("failed to create new listener: %w", err)
	}
	defer l.Close()

	for {
		// Wait for a connection
		fmt.Println("listening for new connection at", c.address)
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
			fmt.Printf("received payload meta from %s: %v\n", conn.RemoteAddr(), payloadMeta)

			switch payloadMeta.PayloadType {

			// Handle TRIGGER_BUY event
			case events.TRIGGER_BUY:
				buyPayload := buyPayload{}
				if err := json.Unmarshal(data, &buyPayload); err != nil {
					fmt.Printf("failed to unmarshal TRIGGER_BUY payload from %s: %v\n", conn.RemoteAddr().String(), err)
					return
				}
				fmt.Printf("received TRIGGER_BUY payload from %s: %#v\n", conn.RemoteAddr().String(), buyPayload)
				c.triggerBuyEvent(buyPayload)

			// Handle INFORM_VOTE event
			case events.INFORM_VOTE:
				informVotePayload := informVotePayload{}
				if err := json.Unmarshal(data, &informVotePayload); err != nil {
					fmt.Printf("failed to unmarshal INFORM_VOTE payload from %s: %v\n", conn.RemoteAddr().String(), err)
					return
				}
				fmt.Printf("received INFORM_VOTE payload from %s: %v\n", conn.RemoteAddr().String(), informVotePayload)
				c.handleInformVote(informVotePayload)

			// Handle unknown events
			default:
				fmt.Printf("failed to determine event type: %v", payloadMeta)
				return
			}
		}(conn)
	}
}

func (c *consumer) NewIperf3Server() error {
	cmds, err := iperf3.StartServers(c.iperf3BaseServerPort, c.iperf3ServerCount)
	if err != nil {
		return fmt.Errorf("failed to start iperf3 server: %w", err)
	}
	c.iperf3Cmds = cmds

	return nil
}

func (c *consumer) cleanup() error {
	fmt.Println("running cleanup...")
	for _, cmd := range c.iperf3Cmds {
		if err := iperf3.StopServer(cmd); err != nil {
			return fmt.Errorf("failed to stop iperf3 server: %w", err)
		}

	}
	fmt.Println("cleanup ran")
	return nil
}

func (c *consumer) persistResults() error {
	fmt.Println("persisting results to file...")
	err := os.MkdirAll(c.outputDir, 0777)
	if err != nil {
		return fmt.Errorf("failed to make new dir: %w", err)
	}

	jsonResult, err := json.Marshal(c.transactions)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction results: %w", err)
	}

	timeString := time.Now().Format("2006-01-02--15-04-05") // Golang weird time format constants
	filename := filepath.Join(c.outputDir, "consumer_transactions--"+timeString)
	err = os.WriteFile(filename, jsonResult, 0777)
	if err != nil {
		return fmt.Errorf("failed to write transactions to file: %w", err)
	}

	fmt.Println("results persisted to:", filename)
	return nil
}
