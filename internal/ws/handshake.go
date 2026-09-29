package ws

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
)

const websocketGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

func Upgrade(rwc net.Conn) (*bufio.Reader, error) {
	br := bufio.NewReader(rwc)

	req, err := http.ReadRequest(br)
	if err != nil {
		return nil, fmt.Errorf("read request: %w", err)
	}

	if err := validateUpgrade(req); err != nil {
		writeHTTPError(rwc, 400, err.Error())
		return nil, err
	}

	key := req.Header.Get("Sec-WebSocket-Key")
	acceptKey := computeAcceptKey(key)

	response := fmt.Sprintf(
		"HTTP/1.1 101 Switching Protocols\r\n"+
			"Upgrade: websocket\r\n"+
			"Connection: Upgrade\r\n"+
			"Sec-WebSocket-Accept: %s\r\n"+
			"\r\n",
		acceptKey,
	)

	if _, err := rwc.Write([]byte(response)); err != nil {
		return nil, fmt.Errorf("write 101: %w", err)
	}
	return br, nil
}

func validateUpgrade(req *http.Request) error {
	if req.Method != http.MethodGet {
		return errors.New("method must be GET")
	}
	if !req.ProtoAtLeast(1, 1) {
		return errors.New("HTTP version must be >= 1.1")
	}
	if !strings.EqualFold(req.Header.Get("Upgrade"), "websocket") {
		return errors.New("missing Upgrade: websocket")
	}
	if !headerContains(req.Header, "Connection", "upgrade") {
		return errors.New("missing Connection: Upgrade")
	}
	key := req.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		return errors.New("missing Sec-WebSocket-Key")
	}
	decoded, err := base64.StdEncoding.DecodeString(key)
	if err != nil || len(decoded) != 16 {
		return errors.New("invalid Sec-WebSocket-Key (must be 16 bytes base64)")
	}
	if req.Header.Get("Sec-WebSocket-Version") != "13" {
		return errors.New("Sec-WebSocket-Version must be 13")
	}
	return nil
}

func computeAcceptKey(key string) string {
	h := sha1.New()
	h.Write([]byte(key))
	h.Write([]byte(websocketGUID))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
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

func writeHTTPError(w net.Conn, code int, msg string) {
	resp := fmt.Sprintf(
		"HTTP/1.1 %d %s\r\nContent-Type: text/plain\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s",
		code, http.StatusText(code), len(msg), msg,
	)
	w.Write([]byte(resp))
}
