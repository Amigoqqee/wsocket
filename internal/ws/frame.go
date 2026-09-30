package ws

import (
	"encoding/binary"
	"fmt"
	"io"
)

const DefaultMaxMessageSize int64 = 64 * 1024

const (
	OpContinuation byte = 0x0
	OpText         byte = 0x1
	OpBinary       byte = 0x2
	OpClose        byte = 0x8
	OpPing         byte = 0x9
	OpPong         byte = 0xA
)

type Frame struct {
	FIN     bool
	Opcode  byte
	Masked  bool
	MaskKey [4]byte
	Payload []byte
}

func (f *Frame) IsControl() bool {
	return f.Opcode >= OpClose
}

func ReadFrame(r io.Reader, maxPayloadSize int64, requireMask bool) (*Frame, error) {
	//1-2 byte
	header := make([]byte, 2)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}

	f := &Frame{}

	//Byte 0
	f.FIN = header[0]&0x80 != 0
	rsv := header[0] & 0x70
	f.Opcode = header[0] & 0x0F

	if rsv != 0 {
		return nil, fmt.Errorf("non-zero RSV bits")
	}

	//Byte 1
	f.Masked = header[1]&0x80 != 0
	if requireMask && !f.Masked {
		return nil, fmt.Errorf("client frame must be masked")
	}

	payloadLen := uint64(header[1] & 0x7F)
	switch {
	case payloadLen == 126:
		ext := make([]byte, 2)
		if _, err := io.ReadFull(r, ext); err != nil {
			return nil, fmt.Errorf("read ext16: %w", err)
		}
		payloadLen = uint64(binary.BigEndian.Uint16(ext))
	case payloadLen == 127:
		ext := make([]byte, 8)
		if _, err := io.ReadFull(r, ext); err != nil {
			return nil, fmt.Errorf("read ext64: %w", err)
		}
		payloadLen = binary.BigEndian.Uint64(ext)
		if payloadLen>>63 != 0 {
			return nil, fmt.Errorf("payload length MSB must be 0")
		}
	}

	if f.Masked {
		if _, err := io.ReadFull(r, f.MaskKey[:]); err != nil {
			return nil, fmt.Errorf("read mask: %w", err)
		}
	}

	if payloadLen > 0 {
		if maxPayloadSize > 0 && payloadLen > uint64(maxPayloadSize) {
			return nil, fmt.Errorf("payload %d exceeds max %d", payloadLen, maxPayloadSize)
		}
		f.Payload = make([]byte, payloadLen)
		if _, err := io.ReadFull(r, f.Payload); err != nil {
			return nil, fmt.Errorf("read payload: %w", err)
		}
		if f.Masked {
			maskPayload(f.Payload, f.MaskKey)
		}
	}
	return f, nil
}

func WriteFrame(w io.Writer, f *Frame) error {
	b0 := f.Opcode
	if f.FIN {
		b0 |= 0x80
	}

	payloadLen := len(f.Payload)
	var header []byte
	switch {
	case payloadLen <= 125:
		header = []byte{b0, byte(payloadLen)}
	case payloadLen <= 65535:
		header = make([]byte, 4)
		header[0] = b0
		header[1] = 126
		binary.BigEndian.PutUint16(header[2:4], uint16(payloadLen))
	default:
		header = make([]byte, 10)
		header[0] = b0
		header[1] = 127
		binary.BigEndian.PutUint64(header[2:10], uint64(payloadLen))
	}

	if _, err := w.Write(header); err != nil {
		return err
	}
	if payloadLen > 0 {
		if _, err := w.Write(f.Payload); err != nil {
			return err
		}
	}
	return nil
}

func maskPayload(payload []byte, key [4]byte) {
	for i := range payload {
		payload[i] ^= key[i%4]
	}
}

func NewTextFrame(data []byte) *Frame {
    return &Frame{FIN: true, Opcode: OpText, Payload: data}
}
func NewBinaryFrame(data []byte) *Frame {
    return &Frame{FIN: true, Opcode: OpBinary, Payload: data}
}
func NewPingFrame(data []byte) *Frame {
    return &Frame{FIN: true, Opcode: OpPing, Payload: data}
}
func NewPongFrame(data []byte) *Frame {
    return &Frame{FIN: true, Opcode: OpPong, Payload: data}
}