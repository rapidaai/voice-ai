// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package utils

import (
	"math"
	"testing"
	"time"
)

func TestGetDuration(t *testing.T) {
	tests := []struct {
		name string
		in   interface{}
		want *time.Duration
	}{
		{name: "string seconds", in: "12", want: durationPtr(12 * time.Second)},
		{name: "decimal seconds", in: "1.5", want: durationPtr(1500 * time.Millisecond)},
		{name: "zero string", in: "0", want: durationPtr(0)},
		{name: "float seconds", in: float64(2), want: durationPtr(2 * time.Second)},
		{name: "int seconds", in: 3, want: durationPtr(3 * time.Second)},
		{name: "uint64 overflow", in: uint64(math.MaxInt64), want: nil},
		{name: "blank string", in: " ", want: nil},
		{name: "invalid string", in: "bad", want: nil},
		{name: "unsupported", in: true, want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetDuration(tt.in)
			if tt.want == nil {
				if got != nil {
					t.Fatalf("expected nil, got %s", got.String())
				}
				return
			}
			if got == nil {
				t.Fatalf("expected %s, got nil", tt.want.String())
			}
			if *got != *tt.want {
				t.Fatalf("expected %s, got %s", tt.want.String(), got.String())
			}
		})
	}
}

func TestOption_GetDuration(t *testing.T) {
	opts := Option{
		"duration": "12",
		"zero":     "0",
		"bad":      "bad",
	}

	if got, err := opts.GetDuration("duration"); err != nil || got != 12*time.Second {
		t.Fatalf("expected 12s, got %v err=%v", got, err)
	}
	if got, err := opts.GetDuration("zero"); err != nil || got != 0 {
		t.Fatalf("expected 0s, got %v err=%v", got, err)
	}
	if _, err := opts.GetDuration("bad"); err == nil {
		t.Fatal("expected error for bad duration")
	}
	if _, err := opts.GetDuration("missing"); err == nil {
		t.Fatal("expected error for missing duration")
	}
}

func TestOption_GetDurationFallback(t *testing.T) {
	opts := Option{
		"CallDuration": "17",
	}

	duration, err := opts.GetDuration("ConversationDuration")
	if err != nil {
		duration, err = opts.GetDuration("Duration")
	}
	if err != nil {
		duration, err = opts.GetDuration("CallDuration")
	}

	if err != nil || duration != 17*time.Second {
		t.Fatalf("expected fallback duration 17s, got %v err=%v", duration, err)
	}
}

func durationPtr(duration time.Duration) *time.Duration {
	return &duration
}
