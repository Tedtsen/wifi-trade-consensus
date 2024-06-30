package provider

import (
	"fmt"
	"net"
	"time"

	"github.com/spf13/viper"
)

type beaconSettings struct {
	peers    peers
	interval int // ms
}

func NewBeaconEmitter(beaconSettings beaconSettings) {
	// Run emitter concurrently
	go func() {
		for {
			// Wait for beacon interval
			time.Sleep(time.Millisecond * time.Duration(beaconSettings.interval))
			for _, peer := range beaconSettings.peers {
				// fmt.Println("sending beacon to:", peer.address)
				conn, err := net.Dial("tcp", peer.address)
				if err != nil {
					// fmt.Printf("failed to send beacon to %s: %v\n", peer.address, err)
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

func NewBeaconSettings(addresses []string, interval int) beaconSettings {
	peers := peers{}
	for _, address := range addresses {
		peers = append(peers, peerInfo{
			providerID: "mock-id",
			address:    address,
		})
	}

	beaconSettings := beaconSettings{
		peers:    peers,
		interval: interval,
	}

	return beaconSettings
}

func NewBeaconSettingsFromConfigFile() (*beaconSettings, error) {

	viper.SetConfigName("beacon_config") // Name of config file (without extension)
	viper.SetConfigType("json")          // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath(".")             // Path to look for the config file in

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
		Addresses []string `mapstructure:"addresses"`
		Interval  int      `mapstructure:"interval"`
	}{}
	err := viper.Unmarshal(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal params config file: %w", err)
	}

	peers := peers{}
	for idx, address := range config.Addresses {
		peers = append(peers, peerInfo{
			providerID: "mock-id" + fmt.Sprint(idx),
			address:    address,
		})
	}

	beaconSettings := beaconSettings{
		peers:    peers,
		interval: config.Interval,
	}

	return &beaconSettings, nil
}
