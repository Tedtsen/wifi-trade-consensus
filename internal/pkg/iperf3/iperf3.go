package iperf3

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

const app = "iperf3"

type streamDirection int

const (
	FORWARD streamDirection = iota
	REVERSE
)

type Results struct {
	End struct {
		SumSent struct {
			BitsPerSecond float64 `json:"bits_per_second"`
		} `json:"sum_sent"`
		SumReceived struct {
			BitsPerSecond float64 `json:"bits_per_second"`
		} `json:"sum_received"`
	} `json:"end"`
}

func StartServers(basePort string, serverCount int) ([]*exec.Cmd, error) {
	cmds := []*exec.Cmd{}
	for i := 0; i < serverCount; i++ {

		port, err := strconv.Atoi(basePort)
		if err != nil {
			return nil, fmt.Errorf("failed to convert basePort to int: %w", err)
		}
		port += i

		cmd, err := startServer(fmt.Sprint(port))
		if err != nil {
			return nil, fmt.Errorf("failed to start server: %w", err)
		}
		cmds = append(cmds, cmd)
	}
	return cmds, nil
}

func startServer(port string) (*exec.Cmd, error) {
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
	fmt.Println("iperf3 server started at localhost port:", port)
	return cmd, nil
}

func StopServer(cmd *exec.Cmd) error {
	err := cmd.Process.Kill()
	if err != nil {
		return err
	}
	return nil
}

// Mock Upload
func StartStream(ip string, basePort string, serverCount int, size string, title string) (*Results, error) {

	for i := 0; i < serverCount; i += 1 {
		port, err := strconv.Atoi(basePort)
		if err != nil {
			return nil, fmt.Errorf("failed to convert basePort to int: %w", err)
		}
		port += i

		res, err := tryStartStream(ip, fmt.Sprint(port), size, title, FORWARD)
		if err != nil {
			fmt.Printf("failed to start stream with port %s: %v\n", fmt.Sprint(port), err)
			continue
		}
		return res, nil

	}
	return nil, fmt.Errorf("failed to send stream, no port available")
}

// Mock Download
func StartReverseStream(ip string, basePort string, serverCount int, size string, title string) (*Results, error) {
	for i := 0; i < serverCount; i += 1 {
		port, err := strconv.Atoi(basePort)
		if err != nil {
			return nil, fmt.Errorf("failed to convert basePort to int: %w", err)
		}
		port += i

		res, err := tryStartStream(ip, fmt.Sprint(port), size, title, REVERSE)
		if err != nil {
			fmt.Printf("failed to start reverse stream with port %s: %v\n", fmt.Sprint(port), err)
			continue
		}
		return res, nil

	}
	return nil, fmt.Errorf("failed to send stream, no port available")

}

func tryStartStream(ip string, port string, size string, title string, direction streamDirection) (*Results, error) {
	args := []string{
		"-c", ip,
		"-p", port,
		"-n", size,
		"-T", title,
		"-J", // JSON output
	}

	if direction == REVERSE {
		args = append(args, "-R")
	}

	cmd := exec.Command(app, args...)

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run iperf3 cmd: %w", err)
	}

	results := Results{}
	if err = json.Unmarshal(out, &results); err != nil {
		return nil, fmt.Errorf("failed to unmarshal json results: %w", err)
	}

	jsonResults, _ := json.MarshalIndent(results, "\t", "\t")
	fmt.Printf("iperf3 stream output: %s\n", string(jsonResults))
	return &results, nil
}
