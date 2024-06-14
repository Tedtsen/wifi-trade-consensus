package provider

import (
	"bufio"
	"fmt"
	"net"
	"time"
)

type PayloadType string

const (
	Beacon int = iota
	Buy
	Traffic
)

type Payload struct {
	PayloadType string `json:"type"`
	Size        int    `json:"size"`
	Utilization int    `json:"utilization"`
}

type Options struct {
	address string
}

func New(option Options) error {

}

// Creates a new listener, this is a blocking function so wrapping the function
// call in a goroutine is required.
func NewListener(option Options) error {
	l, err := net.Listen("tcp", option.address)
	if err != nil {
		return fmt.Errorf("failed to create new listener: %w", err)
	}
	defer l.Close()

	for {
		// Wait for a connection
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("failed to accept new connection: %w", err)
		}
		// Concurrently handle the new connections
		go func(c net.Conn) {
			command, _ := bufio.NewReader(c).ReadString('\n')
			body, _ := bufio.NewReader(c).ReadString('\n')
			switch command {
			case "beacon":
			case "":
			default:
			}

			// Shut down the connection.
			c.Close()
		}(conn)
	}
}

type PeerInfo struct {
	address string
}

type PeerList []PeerInfo

func NewBeaconEmitter(peerList PeerList, interval int) {
	// Run emitter concurrently
	go func() {
		for {
			// Wait for beacon interval
			time.Sleep(time.Millisecond * time.Duration(interval))
			for _, peer := range peerList {
				conn, _ := net.Dial("tcp", peer.address)
				// Send beacon to each peer concurrently
				go func() {
					fmt.Fprint(conn, "test\n")

				}()

			}
		}
	}()
}
