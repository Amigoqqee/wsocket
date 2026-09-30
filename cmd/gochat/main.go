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

	for {
		frame, err := ws.ReadFrame(br)
		if err != nil {
			fmt.Printf("[ws] read: %v\n", err)
			return
		}

		fmt.Printf("[ws] FIN=%v opcode=0x%x len=%d payload=%q\n",
			frame.FIN, frame.Opcode, len(frame.Payload), string(frame.Payload))

		if frame.Opcode == ws.OpText {
			echo := &ws.Frame{
				FIN:     true,
				Opcode:  ws.OpText,
				Payload: frame.Payload,
			}
			ws.WriteFrame(conn, echo)
		}
	}
}
