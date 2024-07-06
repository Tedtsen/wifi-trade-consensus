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
		fmt.Println("failed to read beacon settings from config file:", err)
		return
	}
	fmt.Println("beacon settings loaded:", *beaconSettings)

	// params := provider.NewParams(1000, 0.5, 0.5, 0.5, 3, 0)
	// options := provider.NewOptions("localhost:8080", 0.0000007, 30, 50, params)
	options, err := provider.NewParamsOptionsFromConfigFile()
	if err != nil {
		fmt.Println("failed to read params and options from config file:", err)
		return
	}
	fmt.Println("options loaded:", *options)

	p := provider.New(*options)

	// Begin beacon broadcast
	go p.NewBeaconEmitter(*beaconSettings)

	// Create new iperf3 server
	if err := p.NewIperf3Server(); err != nil {
		fmt.Println("failed to create iperf3 server:", err)
		return
	}

	// Create new listener for provider
	if err := p.NewListener(); err != nil {
		fmt.Println("failed to create new listener:", err)
		return
	}
}
