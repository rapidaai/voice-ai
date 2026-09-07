// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_type

import "testing"

type sipRTPBridgeTargetStub struct{}

func (sipRTPBridgeTargetStub) WriteAudio([]byte) error { return nil }

func TestSIPRTPBridgeTargetWritesCompleteFrame(t *testing.T) {
	var target SIPRTPBridgeTarget = sipRTPBridgeTargetStub{}
	if err := target.WriteAudio([]byte{0xff}); err != nil {
		t.Fatalf("WriteAudio() error = %v", err)
	}
}
