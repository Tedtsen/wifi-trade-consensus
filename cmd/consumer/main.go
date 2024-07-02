package main

import (
	"fmt"
	"wifi-trade-consensus/internal/consumer"
)

func main() {
	opt, err := consumer.NewOptionsFromConfigFile()
	if err != nil {
		fmt.Println("failed to read options from config file:", err)
		return
	}

	consumer := consumer.New(*opt)

	if err := consumer.NewIperf3Server(); err != nil {
		fmt.Println("failed to create iperf3 server:", err)
		return
	}

	if err := consumer.NewListener(); err != nil {
		fmt.Println("failed to create new listener:", err)
		return
	}
}
