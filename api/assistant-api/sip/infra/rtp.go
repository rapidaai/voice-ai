// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_infra

import (
	"context"
	"net"

	internal_core "github.com/rapidaai/api/assistant-api/sip/internal/core"
)

func NewRTPHandler(ctx context.Context, config *RTPConfig) (*RTPHandler, error) {
	inner, err := internal_core.NewRTPHandler(ctx, config)
	if err != nil {
		return nil, err
	}
	return wrapRTPHandler(inner), nil
}

func wrapRTPHandler(inner *internal_core.RTPHandler) *RTPHandler {
	if inner == nil {
		return nil
	}
	return &RTPHandler{inner: inner}
}

func (h *RTPHandler) unwrap() *internal_core.RTPHandler {
	if h == nil {
		return nil
	}
	return h.inner
}

func (h *RTPHandler) Start() {
	if h == nil {
		return
	}
	if h.inner != nil {
		h.inner.Start()
	}
}

func (h *RTPHandler) Stop() error {
	if h == nil {
		return nil
	}
	if h.inner != nil {
		return h.inner.Stop()
	}
	return nil
}

func (h *RTPHandler) IsRunning() bool {
	if h == nil {
		return false
	}
	if h.inner != nil {
		return h.inner.IsRunning()
	}
	return true
}

func (h *RTPHandler) SetRemoteAddr(ip string, port int) {
	if h == nil {
		return
	}
	if h.inner != nil {
		h.inner.SetRemoteAddr(ip, port)
	}
}

func (h *RTPHandler) GetRemoteAddr() *net.UDPAddr {
	if h == nil {
		return nil
	}
	if h.inner != nil {
		return h.inner.GetRemoteAddr()
	}
	return nil
}

func (h *RTPHandler) LocalAddr() (string, int) {
	if h == nil {
		return "", 0
	}
	if h.inner != nil {
		return h.inner.LocalAddr()
	}
	return "", 0
}

func (h *RTPHandler) AudioIn() <-chan []byte {
	if h == nil {
		return nil
	}
	if h.inner != nil {
		return h.inner.AudioIn()
	}
	if h.audioInChan == nil {
		h.audioInChan = make(chan []byte, 100)
	}
	return h.audioInChan
}

func (h *RTPHandler) EnqueueAudio(audio []byte) error {
	if h == nil {
		return ErrRTPNotInitialized
	}
	if len(audio) == 0 {
		return nil
	}
	if h.inner != nil {
		return h.inner.EnqueueAudio(audio)
	}
	if h.audioOutChan == nil {
		h.audioOutChan = make(chan []byte, 100)
	}
	select {
	case h.audioOutChan <- audio:
		return nil
	default:
		return ErrRTPOutputQueueFull
	}
}

func (h *RTPHandler) FlushAudioOut() {
	if h == nil {
		return
	}
	if h.inner != nil {
		h.inner.FlushAudioOut()
		return
	}
	if h.flushAudioCh == nil {
		h.flushAudioCh = make(chan struct{}, 1)
	}
	select {
	case h.flushAudioCh <- struct{}{}:
	default:
	}
}

func (h *RTPHandler) SetFallbackAudioSource(source RTPFallbackAudioSource) {
	if h == nil {
		return
	}
	if h.inner != nil {
		h.inner.SetFallbackAudioSource(source)
		return
	}
	h.fallbackSource = source
}

func (h *RTPHandler) ClearFallbackAudioSource() {
	h.SetFallbackAudioSource(nil)
}

func (h *RTPHandler) GetCodec() *Codec {
	if h == nil {
		return nil
	}
	if h.inner != nil {
		return codecFromCore(h.inner.GetCodec())
	}
	return h.codec
}

func (h *RTPHandler) SetCodec(codec *Codec) {
	if h == nil {
		return
	}
	if h.inner != nil {
		h.inner.SetCodec(codecToCore(codec))
		return
	}
	h.codec = codec
}

func (h *RTPHandler) GetStats() (sent, received uint64) {
	if h == nil {
		return 0, 0
	}
	if h.inner != nil {
		return h.inner.GetStats()
	}
	return 0, 0
}

func (h *RTPHandler) GetDetailedStats() RTPStats {
	if h == nil {
		return RTPStats{}
	}
	if h.inner != nil {
		return h.inner.GetDetailedStats()
	}
	return RTPStats{}
}
