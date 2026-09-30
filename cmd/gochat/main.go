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

func handleConn(rawConn net.Conn) {
	br, err := ws.Upgrade(rawConn)
	if err != nil {
		rawConn.Close()
		return
	}

	conn := ws.NewConn(rawConn, br, ws.DefaultMaxMessageSize)
	defer conn.ForceClose()

	for {
		opcode, data, err := conn.ReadMessage()
		if err != nil {
			if !ws.IsCloseError(err) {
				fmt.Printf("[ws] error: %v\n", err)
			}
			return
		}
		if opcode == ws.OpText {
			conn.WriteText(data)
		}
	}
}
