package iperf3

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

const app = "iperf3"

type results struct {
	End struct {
		SumSent struct {
			BitsPerSecond float64 `json:"bits_per_second"`
		} `json:"sum_sent"`
		SumReceived struct {
			BitsPerSecond float64 `json:"bits_per_second"`
		} `json:"sum_received"`
	} `json:"end"`
}

func StartServer(port string) (*exec.Cmd, error) {
	args := []string{
		"-s", // type: server
		"-p", port,
		"-J", // JSON output
	}

	cmd := exec.Command(app, args...)

	buf := &bytes.Buffer{}
	cmd.Stdout = buf

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to run iperf3 cmd: %w", err)
	}
	fmt.Printf("iperf3 start server output: %s\n", buf.String())
	return cmd, nil
}

func StopServer(cmd *exec.Cmd) error {
	err := cmd.Process.Kill()
	if err != nil {
		return err
	}
	return nil
}

func StartStream(ip string, port string, size string) (*results, error) {
	args := []string{
		"-c", ip,
		"-p", port,
		"-n", size,
		"-J", // JSON output
	}

	cmd := exec.Command(app, args...)

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run iperf3 cmd: %w", err)
	}

	results := results{}
	if err = json.Unmarshal(out, &results); err != nil {
		return nil, fmt.Errorf("failed to unmarshal json results: %w", err)
	}

	jsonResults, _ := json.MarshalIndent(results, "\t", "\t")
	fmt.Printf("iperf3 stream output: %s\n", string(jsonResults))
	return &results, nil
}
