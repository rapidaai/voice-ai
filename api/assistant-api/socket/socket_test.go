// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package assistant_socket

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"strings"
	"testing"

	rapida_client "github.com/rapidaai/pkg/clients/rapida"
)

func TestAudioSocketEngineRetainsRapidaClient(t *testing.T) {
	client := &rapida_client.RapidaClient{}
	engine := &audioSocketEngine{rapidaClient: client}

	if engine.rapidaClient != client {
		t.Fatal("audioSocketEngine did not retain RapidaClient")
	}
}

func TestReadContextID_HappyPath(t *testing.T) {
	engine := &audioSocketEngine{}

	// Build a valid UUID frame: type=0x01, length=16, payload=16 bytes
	uuid := "550e8400-e29b-41d4-a716-446655440000"
	uuidHex := strings.ReplaceAll(uuid, "-", "")
	payload, err := hex.DecodeString(uuidHex)
	if err != nil {
		t.Fatalf("failed to decode UUID hex: %v", err)
	}

	var buf bytes.Buffer
	buf.WriteByte(0x01) // frame type UUID
	lenBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(lenBytes, uint16(len(payload)))
	buf.Write(lenBytes)
	buf.Write(payload)

	reader := bufio.NewReader(&buf)
	got, err := engine.readContextID(reader)
	if err != nil {
		t.Fatalf("readContextID returned error: %v", err)
	}
	if got != uuid {
		t.Errorf("readContextID = %q, want %q", got, uuid)
	}
}

func TestReadContextID_WrongFrameType(t *testing.T) {
	engine := &audioSocketEngine{}

	var buf bytes.Buffer
	buf.WriteByte(0x10) // wrong frame type
	lenBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(lenBytes, 16)
	buf.Write(lenBytes)
	buf.Write(make([]byte, 16))

	reader := bufio.NewReader(&buf)
	_, err := engine.readContextID(reader)
	if err == nil {
		t.Fatal("expected error for wrong frame type, got nil")
	}
	if !strings.Contains(err.Error(), "expected UUID frame") {
		t.Errorf("error should mention expected UUID frame, got: %v", err)
	}
}

func TestReadContextID_InvalidPayloadLength(t *testing.T) {
	engine := &audioSocketEngine{}

	var buf bytes.Buffer
	buf.WriteByte(0x01)
	lenBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(lenBytes, 10) // wrong payload length (not 16)
	buf.Write(lenBytes)
	buf.Write(make([]byte, 10))

	reader := bufio.NewReader(&buf)
	_, err := engine.readContextID(reader)
	if err == nil {
		t.Fatal("expected error for invalid payload length, got nil")
	}
	if !strings.Contains(err.Error(), "invalid UUID payload length") {
		t.Errorf("error should mention invalid UUID payload length, got: %v", err)
	}
}

func TestReadContextID_EmptyReader(t *testing.T) {
	engine := &audioSocketEngine{}

	var buf bytes.Buffer
	reader := bufio.NewReader(&buf)
	_, err := engine.readContextID(reader)
	if err == nil {
		t.Fatal("expected error for empty reader, got nil")
	}
}

func TestReadContextID_TruncatedLength(t *testing.T) {
	engine := &audioSocketEngine{}

	var buf bytes.Buffer
	buf.WriteByte(0x01) // frame type
	buf.WriteByte(0x00) // only 1 byte of length (need 2)

	reader := bufio.NewReader(&buf)
	_, err := engine.readContextID(reader)
	if err == nil {
		t.Fatal("expected error for truncated length, got nil")
	}
}

func TestReadContextID_TruncatedPayload(t *testing.T) {
	engine := &audioSocketEngine{}

	var buf bytes.Buffer
	buf.WriteByte(0x01)
	lenBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(lenBytes, 16)
	buf.Write(lenBytes)
	buf.Write(make([]byte, 8)) // only 8 bytes, need 16

	reader := bufio.NewReader(&buf)
	_, err := engine.readContextID(reader)
	if err == nil {
		t.Fatal("expected error for truncated payload, got nil")
	}
}
