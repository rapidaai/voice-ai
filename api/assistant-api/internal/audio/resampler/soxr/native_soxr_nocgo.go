//go:build !cgo

// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package resampler_soxr

type nativePCM16Resampler struct{}

func newNativePCM16Resampler(uint32, uint32, nativeSOXRQuality) (*nativePCM16Resampler, error) {
	return nil, ErrNativeSOXRUnavailable
}

func (resampler *nativePCM16Resampler) Resample([]byte) ([]byte, error) {
	return nil, ErrNativeSOXRUnavailable
}

func (resampler *nativePCM16Resampler) WriteTo([]byte, func([]byte) error) error {
	return ErrNativeSOXRUnavailable
}

func (resampler *nativePCM16Resampler) FlushTo(func([]byte) error) error {
	return ErrNativeSOXRUnavailable
}

func (resampler *nativePCM16Resampler) Close() {}
