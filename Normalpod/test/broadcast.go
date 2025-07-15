package main

import (
	"fmt"
	"net"
	"time"
)

func main() {
	addr, err := net.ResolveUDPAddr("udp", "255.255.255.255:9876")
	if err != nil {
		fmt.Printf("Error resolving address: %v\n", err)
		return
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		fmt.Printf("Error dialing UDP: %v\n", err)
		return
	}
	defer conn.Close()

	for {
		message := []byte("Hello from broadcaster")
		_, err := conn.Write(message)
		if err != nil {
			fmt.Printf("Error sending message: %v\n", err)
		} else {
			fmt.Println("Broadcast message sent")
		}
		time.Sleep(5 * time.Second) // 每5秒发送一次
	}
}
