package main

import (
	"fmt"
	"log"
	"net"

	"github.com/Amigoqqee/wsocket/internal/ws"
)

func main() {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	fmt.Println("server listening on :8080")

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept: %v", err)
			continue
		}

		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()

	br, err := ws.Upgrade(conn)
	if err != nil {
		fmt.Printf("[ws] upgrade failed: %v\n", err)
		return
	}

	fmt.Printf("[ws] connected: %s\n", conn.RemoteAddr())

	buf := make([]byte, 4096)
	for {
		n, err := br.Read(buf)
		if err != nil {
			fmt.Printf("[ws] disconnected: %s\n", conn.RemoteAddr())
			return
		}
		fmt.Printf("[ws] raw frame bytes: %x\n", buf[:n])
	}
}
