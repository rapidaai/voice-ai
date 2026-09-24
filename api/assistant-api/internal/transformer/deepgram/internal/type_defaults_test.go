// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package deepgram_internal

import "testing"

func TestSpeechToTextEndpointingDefault(t *testing.T) {
	if STTDefaultEndpointing != "500" {
		t.Fatalf("STTDefaultEndpointing = %q, want 500", STTDefaultEndpointing)
	}
}
