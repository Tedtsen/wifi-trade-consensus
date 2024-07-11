package provider

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"
	"wifi-trade-consensus/internal/pkg/events"

	"github.com/spf13/viper"
)

type beaconSettings struct {
	peers                      peers
	interval                   int // ms
	mockChannelUtilizationRate int
	mockRSSI                   int
}

func (p *provider) NewBeaconEmitter(beaconSettings beaconSettings) {
	for {
		// Wait for beacon interval
		time.Sleep(time.Millisecond * time.Duration(beaconSettings.interval))
		for _, peer := range beaconSettings.peers {
			// fmt.Println("sending beacon to:", peer.address)
			conn, err := net.Dial("tcp", peer.Address)
			if err != nil {
				fmt.Printf("failed to send beacon to %s: %v\n", peer.Address, err)
				continue
			}

			p.channelUtilizationRate = calculateChannelUtilizationRate(p.activeFlowCount)

			payload := beaconPayload{
				PayloadMeta: PayloadMeta{
					PayloadType:   events.BEACON,
					OriginID:      p.id,
					OriginAddress: p.address,
				},
				ChannelUtilizationRate: p.channelUtilizationRate,
				RSSI:                   beaconSettings.mockRSSI,
			}

			jsonPayload, err := json.Marshal(payload)
			if err != nil {
				fmt.Println("failed to marshal beacon payload:", err)
				continue
			}

			// Send beacon to each peer concurrently
			go func(conn net.Conn) {
				if _, err := conn.Write(jsonPayload); err != nil {
					fmt.Println("failed to send beacon:", err)
				}
				conn.Close()
			}(conn)
		}
	}
}

func NewBeaconSettings(addresses []string, interval int, mockChannelUtil int, mockRSSI int) beaconSettings {
	peers := peers{}
	for _, address := range addresses {
		peers = append(peers, peerInfo{
			ProviderID: "mock-id",
			Address:    address,
		})
	}

	beaconSettings := beaconSettings{
		peers:                      peers,
		interval:                   interval,
		mockChannelUtilizationRate: mockChannelUtil,
		mockRSSI:                   mockRSSI,
	}

	return beaconSettings
}

func NewBeaconSettingsFromConfigFile() (*beaconSettings, error) {

	// Name of config file (without extension)
	// Get docker container's env variable
	if nodeNum := os.Getenv("node_num"); nodeNum != "" {
		viper.SetConfigName("beacon_config" + nodeNum)
	} else {
		viper.SetConfigName("beacon_config")
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

	config := struct {
		Addresses                  []string `mapstructure:"addresses"`
		Interval                   int      `mapstructure:"interval"`
		MockChannelUtilizationRate int      `mapstructure:"mock_channel_utilization_rate"`
		MockRSSI                   int      `mapstructure:"mock_rssi"`
	}{}
	err := viper.Unmarshal(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal params config file: %w", err)
	}

	peers := peers{}
	for idx, address := range config.Addresses {
		peers = append(peers, peerInfo{
			ProviderID: "mock-id" + fmt.Sprint(idx),
			Address:    address,
		})
	}

	beaconSettings := beaconSettings{
		peers:                      peers,
		interval:                   config.Interval,
		mockChannelUtilizationRate: config.MockChannelUtilizationRate,
		mockRSSI:                   config.MockRSSI,
	}

	return &beaconSettings, nil
}
