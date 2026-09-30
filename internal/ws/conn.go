package ws

import (
	"bufio"
	"fmt"
	"net"
	"sync"
	"time"
	"unicode/utf8"
)

type Conn struct {
	rwc        net.Conn
	br         *bufio.Reader
	writeMu    sync.Mutex
	maxMsgSize int64

	OnPong func(data []byte)
}

func NewConn(rwc net.Conn, br *bufio.Reader, maxMsgSize int64) *Conn {
	return &Conn{
		rwc:        rwc,
		br:         br,
		maxMsgSize: maxMsgSize,
	}
}

func (c *Conn) ReadMessage() (opcode byte, payload []byte, err error) {
	for {
		frame, err := ReadFrame(c.br, c.maxMsgSize, true)
		if err != nil {
			return 0, nil, err
		}

		if frame.IsControl() {
			if err := c.handleControl(frame); err != nil {
				return 0, nil, err
			}
			continue
		}

		if frame.Opcode == OpText && !utf8.Valid(frame.Payload) {
			c.Close(CloseInvalidPayload, "invalid UTF-8")
			return 0, nil, fmt.Errorf("invalid UTF-8")
		}

		return frame.Opcode, frame.Payload, nil
	}
}

func (c *Conn) handleControl(f *Frame) error {
	switch f.Opcode {
	case OpPing:
		return c.writeFrame(NewPongFrame(f.Payload))
	case OpPong:
		if c.OnPong != nil {
			c.OnPong(f.Payload)
		}
		return nil
	case OpClose:
		code, reason, _ := ParseClosePayload(f.Payload)
		c.writeFrame(NewCloseFrame(code, ""))
		return &CloseError{Code: code, Reason: reason}
	default:
		return fmt.Errorf("unknown control opcode: 0x%x", f.Opcode)
	}
}

func (c *Conn) WriteText(data []byte) error {
	return c.writeFrame(NewTextFrame(data))
}
func (c *Conn) WriteBinary(data []byte) error {
	return c.writeFrame(NewBinaryFrame(data))
}
func (c *Conn) WritePing(data []byte) error {
	return c.writeFrame(NewPingFrame(data))
}
func (c *Conn) Close(code uint16, reason string) error {
	return c.writeFrame(NewCloseFrame(code, reason))
}
func (c *Conn) ForceClose() error {
	return c.rwc.Close()
}
func (c *Conn) writeFrame(f *Frame) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return WriteFrame(c.rwc, f)
}
func (c *Conn) RemoteAddr() net.Addr              { return c.rwc.RemoteAddr() }
func (c *Conn) SetReadDeadline(t time.Time) error { return c.rwc.SetReadDeadline(t) }

const (
	CloseNormal         uint16 = 1000
	CloseGoingAway      uint16 = 1001
	CloseProtocolError  uint16 = 1002
	CloseUnsupported    uint16 = 1003
	CloseInvalidPayload uint16 = 1007
	CloseMessageTooBig  uint16 = 1009
)

type CloseError struct {
	Code   uint16
	Reason string
}

func (e *CloseError) Error() string {
	return fmt.Sprintf("websocket closed: %d %s", e.Code, e.Reason)
}
func IsCloseError(err error) bool {
	_, ok := err.(*CloseError)
	return ok
}
