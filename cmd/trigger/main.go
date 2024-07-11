package main

import (
	"fmt"
	"wifi-trade-consensus/internal/trigger"
)

func main() {
	opt, err := trigger.NewOptionsFromConfigFile()
	if err != nil {
		fmt.Println("failed to read options from config file:", err)
	}

	t := trigger.New(opt)
	t.Start()
}
