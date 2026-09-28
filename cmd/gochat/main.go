package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
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

	fmt.Printf("[http] %s %s %s\n", req.Method, req.URL.Path, req.Proto)
	for name, values := range req.Header {
		fmt.Printf("[http] %s: %s\n", name, values)
	}

	body := "Hello World\n"
	response := fmt.Sprintf(
		"HTTP/1.1 200 OK\r\n"+
			"Content-Type: text/plain\r\n"+
			"Content-Length: %d\r\n"+
			"Connection: close\r\n"+
			"\r\n"+
			"%s",
		len(body), body,
	)
	conn.Write([]byte(response))
}
