package main

import (
	"fmt"
	"wifi-trade-consensus/internal/provider"
)

func main() {
	// Run beacon emitter concurrently, with mock peer list
	// go provider.NewBeaconEmitter(provider.NewMockPeerList([]string{
	// 	"localhost:8888",
	// }), 110)
	beaconSettings, err := provider.NewBeaconSettingsFromConfigFile()
	if err != nil {
		panic(fmt.Errorf("failed to read beacon settings from config file: %w", err))
	}
	fmt.Println("beacon settings loaded:", *beaconSettings)

	// params := provider.NewParams(1000, 0.5, 0.5, 0.5, 3, 0)
	// options := provider.NewOptions("localhost:8080", 0.0000007, 30, 50, params)
	options, err := provider.NewParamsOptionsFromConfigFile()
	if err != nil {
		panic(fmt.Errorf("failed to read params and options from config file: %w", err))
	}
	fmt.Println("options loaded:", *options)

	// Begin beacon broadcast
	go provider.NewBeaconEmitter(*beaconSettings)

	// Create new listener for provider
	p := provider.New(*options)
	if err := p.NewListener(); err != nil {
		fmt.Printf("failed to create new listener: %v", err)
	}
}
