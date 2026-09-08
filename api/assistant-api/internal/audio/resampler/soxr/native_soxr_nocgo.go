//go:build !cgo

// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package resampler_soxr

import "errors"

type nativePCM16Resampler struct{}

func newNativePCM16Resampler(uint32, uint32) (*nativePCM16Resampler, error) {
	return nil, errors.New("native SOXR requires CGO")
}

func (resampler *nativePCM16Resampler) Resample([]byte) ([]byte, error) {
	return nil, errors.New("native SOXR requires CGO")
}

func (resampler *nativePCM16Resampler) Close() {}
