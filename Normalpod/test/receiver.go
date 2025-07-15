package main

import (
	"fmt"
	"net"
)

func main() {
	addr, err := net.ResolveUDPAddr("udp", ":9876")
	if err != nil {
		fmt.Printf("Error resolving address: %v\n", err)
		return
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Printf("Error listening on UDP: %v\n", err)
		return
	}
	defer conn.Close()

	buf := make([]byte, 1024)
	for {
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Printf("Error receiving message: %v\n", err)
			continue
		}
		fmt.Printf("Received message from %v: %s\n", addr, string(buf[:n]))
	}
}
