package trigger

import (
	"encoding/json"
	"fmt"
	"net"
	"time"
	"wifi-trade-consensus/internal/pkg/events"
	"wifi-trade-consensus/internal/pkg/payload"

	"github.com/spf13/viper"
)

type PayloadMeta payload.Meta

type providers []providerInfo

type providerInfo struct {
	ProviderID           string  `mapstructure:"provider_id" json:"provider_id"`
	Address              string  `mapstructure:"address" json:"address"`
	Iperf3BaseServerPort string  `json:"iperf3_base_server_port"`
	Iperf3ServerCount    int     `json:"iperf3_server_count"`
	Price                float64 `json:"price"`
}

type qosRequirements struct {
	PriceConsumer         float64 `mapstructure:"price" json:"price"`       // consumer price requirement
	UplinkSpeedConsumer   float64 `mapstructure:"uplink" json:"uplink"`     // consumer uplink speed requirement
	DownlinkSpeedConsumer float64 `mapstructure:"downlink" json:"downlink"` // consumer downlink speed requirement
	Mu                    float64 `mapstructure:"mu" json:"mu"`             // uplink weight
	Delta                 float64 `mapstructure:"delta" json:"delta"`       // downlink weight
	Epsilon               float64 `mapstructure:"epsilon" json:"epsilon"`   // price range multiplier limit
}

type buyPayload struct {
	PayloadMeta
	ProviderList providers `json:"provider_list"`
	qosRequirements
}

type options struct {
	ConsumerAddress        string    `mapstructure:"consumer_address"`
	BuyEventCount          int       `mapstructure:"buy_event_count"`
	BuyEventIntervalMean   float64   `mapstructure:"buy_event_interval_mean"` // seconds
	BuyEventIntervalStdDev float64   `mapstructure:"buy_event_interval_std_dev"`
	UplinkMean             float64   `mapstructure:"uplink_mean"`
	UplinkStdDev           float64   `mapstructure:"uplink_std_dev"`
	DownlinkMean           float64   `mapstructure:"downlink_mean"`
	DownlinkStdDev         float64   `mapstructure:"downlink_std_dev"`
	PriceMean              float64   `mapstructure:"price_mean"`
	PriceStdDev            float64   `mapstructure:"price_std_dev"`
	MuMean                 float64   `mapstructure:"mu_mean"` // uplink weight
	MuStdDev               float64   `mapstructure:"mu_std_dev"`
	DeltaMean              float64   `mapstructure:"delta_mean"` // downlink weight
	DeltaStdDev            float64   `mapstructure:"delta_std_dev"`
	EpsilonMean            float64   `mapstructure:"epsilon_mean"` // price range multiplier limit
	EpsilonStdDev          float64   `mapstructure:"epsilon_std_dev"`
	ProviderList           providers `mapstructure:"provider_list"`
}

type trigger struct {
	options
}

func NewOptionsFromConfigFile() (*options, error) {
	options := options{}

	viper.SetConfigName("config") // Name of config file (without extension)
	viper.SetConfigType("json")   // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath(".")      // Path to look for the config file in
	viper.AddConfigPath("cmd/trigger")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// config not found, ignore err
			return nil, fmt.Errorf("config file not found")
		} else {
			// other errors, ignore err
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	err := viper.Unmarshal(&options)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal options config file: %w", err)
	}

	return &options, nil
}

func New(opt *options) trigger {
	return trigger{
		*opt,
	}
}

func (t *trigger) Start() {
	for i := 0; i < t.BuyEventCount; i++ {
		interval := getRandomizedVal(t.BuyEventIntervalMean, t.BuyEventIntervalStdDev)
		time.Sleep(time.Second * time.Duration(interval))

		conn, err := net.Dial("tcp", t.ConsumerAddress)
		if err != nil {
			fmt.Println("failed to dial consumer:", err)
		}

		buyPayload := buyPayload{
			PayloadMeta: PayloadMeta{
				PayloadType: events.TRIGGER_BUY,
			},
			ProviderList: t.ProviderList,
			qosRequirements: qosRequirements{
				PriceConsumer:         getRandomizedVal(t.PriceMean, t.PriceStdDev),
				UplinkSpeedConsumer:   getRandomizedVal(t.UplinkMean, t.UplinkStdDev),
				DownlinkSpeedConsumer: getRandomizedVal(t.DownlinkMean, t.DownlinkStdDev),
				Mu:                    getRandomizedVal(t.MuMean, t.MuStdDev),
				Delta:                 getRandomizedVal(t.DeltaMean, t.DeltaStdDev),
				Epsilon:               getRandomizedVal(t.EpsilonMean, t.EpsilonStdDev),
			},
		}

		jsonPayload, err := json.Marshal(buyPayload)
		if err != nil {
			fmt.Println("failed to marshal buy payload:", err)
		}

		if _, err = conn.Write(jsonPayload); err != nil {
			fmt.Println("failed to send TRIGGER_BUY event to consumer:", err)
		} else {
			fmt.Println("sent TRIGGER_BUY to consumer:", string(jsonPayload))
		}
		conn.Close()
	}
}
