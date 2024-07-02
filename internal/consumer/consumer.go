package consumer

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
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
	FFSnew FFS `json:"FFS_new"`
}

type consumer struct {
	id               uuid.UUID
	address          string
	transactions     transactions
	qosRequirements  qosRequirements
	iperf3ServerPort string
	iperf3Cmd        *exec.Cmd
}

type transactions map[string]transaction

type transaction struct {
	transactionID   uuid.UUID
	transactionTime int64
	consumerID      uuid.UUID
	consumerAddress string
	providerList    providers
	providerCount   int
	qosRequirements qosRequirements
	allFFS          allFFS
}

type providers []providerInfo

type providerInfo struct {
	providerID string
	address    string
}

type options struct {
	address          string          `mapstructure:"address"`
	iperf3ServerPort string          `mapstructure:"iperf3_server_port"`
	qosRequirements  qosRequirements `mapstructure:"params"`
}

type qosRequirements struct {
	PriceConsumer         float64 `mapstructure:"price"`    // consumer price requirement
	UplinkSpeedConsumer   float64 `mapstructure:"uplink"`   // consumer uplink speed requirement
	DownlinkSpeedConsumer float64 `mapstructure:"downlink"` // consumer downlink speed requirement
	Mu                    float64 `mapstructure:"mu"`       // uplink weight
	Delta                 float64 `mapstructure:"delta"`    // downlink weight
	Epsilon               float64 `mapstructure:"epsilon"`  // price range multiplier limit
}

func NewOptionsFromConfigFile() (*options, error) {
	options := options{}
	qosRequirements := qosRequirements{}

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

	err := viper.Unmarshal(&qosRequirements)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal qosRequirements config file: %w", err)
	}

	err = viper.Unmarshal(&options)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal options config file: %w", err)
	}

	options.qosRequirements = qosRequirements

	return &options, nil
}

func New(opt options) consumer {
	consumer := consumer{
		id:               uuid.New(),
		address:          opt.address,
		qosRequirements:  opt.qosRequirements,
		iperf3ServerPort: opt.iperf3ServerPort,
	}

	// Register cleanup for interrupt signal i.e. Ctrl^c
	channel := make(chan os.Signal)
	signal.Notify(channel, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-channel
		if err := consumer.cleanup(); err != nil {
			fmt.Println("failed to cleanup:", err)
		}
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

			payloadMeta := payload.Meta{}
			d := json.NewDecoder(conn)
			err := d.Decode(&payloadMeta)
			if err != nil {
				fmt.Printf("failed to decode payload meta from %s: %v\n", conn.RemoteAddr().String(), err)
				return
			}
			fmt.Printf("received payload meta from %s: %v\n", conn.RemoteAddr(), payloadMeta)

			switch payloadMeta.PayloadType {

			// Handle INFORM_VOTE event
			case events.INFORM_VOTE:
				informVotePayload := informVotePayload{}
				if err := d.Decode(&informVotePayload); err != nil {
					fmt.Printf("failed to decode INFORM_VOTE payload from %s: %v\n", conn.RemoteAddr().String(), err)
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
	cmd, err := iperf3.StartServer(c.iperf3ServerPort)
	if err != nil {
		return fmt.Errorf("failed to start iperf3 server: %w", err)
	}
	c.iperf3Cmd = cmd

	return nil
}

func (c *consumer) cleanup() error {
	if err := iperf3.StopServer(c.iperf3Cmd); err != nil {
		return fmt.Errorf("failed to stop iperf3 server: %w", err)
	}
	fmt.Println("cleanup ran, preparing to shutdown...")
	time.Sleep(time.Second * 3)
	return nil
}
