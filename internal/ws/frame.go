package ws

import (
	"fmt"
	"io"
)

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

func ReadFrame(r io.Reader) (*Frame, error) {
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
	payloadLen := uint64(header[1] & 0x7F)

	if f.Masked {
		if _, err := io.ReadFull(r, f.MaskKey[:]); err != nil {
			return nil, fmt.Errorf("read mask: %w", err)
		}
	}

	if payloadLen > 0 {
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
	header := []byte{b0, byte(payloadLen)}

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
