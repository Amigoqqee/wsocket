package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
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

	br := bufio.NewReader(conn)

	req, err := http.ReadRequest(br)
	if err != nil {
		fmt.Printf("[http] parse error: %v\n", err)
		return
	}

	if !isWebSocketUpgrade(req) {
		body := "Not a WebSocket request\n"
		fmt.Fprintf(conn,
			"HTTP/1.1 400 Bad Request\r\n"+
				"Content-Length: %d\r\n"+
				"Connection: close\r\n"+
				"\r\n"+
				"%s", len(body), body)
		return
	}

	fmt.Printf("[ws] upgrade request from %s\n", conn.RemoteAddr())

	conn.Write([]byte(
		"HTTP/1.1 101 Switching Protocols\r\n" +
			"Upgrade: websocket\r\n" +
			"Connection: Upgrade\r\n" +
			"\r\n",
	))

	fmt.Printf("[ws] upgraded: %s\n", conn.RemoteAddr())

	buf := make([]byte, 4096)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			return
		}
		fmt.Printf("[ws] raw bytes: % x\n", buf[:n])
	}
}

func isWebSocketUpgrade(req *http.Request) bool {
	if req.Method != http.MethodGet {
		return false
	}
	if !strings.EqualFold(req.Header.Get("Upgrade"), "websocket") {
		return false
	}
	if !headerContains(req.Header, "Connection", "upgrade") {
		return false
	}
	return true
}

func headerContains(h http.Header, key, value string) bool {
	for _, v := range h[http.CanonicalHeaderKey(key)] {
		for _, s := range strings.Split(v, ",") {
			if strings.EqualFold(strings.TrimSpace(s), value) {
				return true
			}
		}
	}
	return false
}
