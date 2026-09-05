// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rapidaai/pkg/commons"
)

func bridgeTestLogger() commons.Logger {
	l, _ := commons.NewApplicationLogger(commons.Level("debug"))
	return l
}

type bridgeAudioSink struct {
	frames chan []byte
}

func newBridgeAudioSink(size int) *bridgeAudioSink {
	return &bridgeAudioSink{frames: make(chan []byte, size)}
}

func (s *bridgeAudioSink) EnqueueAudio(audio []byte) error {
	select {
	case s.frames <- audio:
		return nil
	default:
		return ErrRTPOutputQueueFull
	}
}

// newTestRTPHandler creates an RTPHandler with pre-made channels and no real UDP socket.
func newTestRTPHandler() *RTPHandler {
	ctx, cancel := context.WithCancel(context.Background())
	h := &RTPHandler{
		audioInChan:  make(chan []byte, 100),
		audioOutChan: make(chan []byte, 100),
		flushAudioCh: make(chan struct{}, 1),
		ctx:          ctx,
		cancel:       cancel,
	}
	h.running.Store(true)
	return h
}

func bridgeTestConfig() *Config {
	return &Config{
		Server:            "127.0.0.1",
		Port:              5060,
		Username:          "testuser",
		Password:          "testpass",
		RTPPortRangeStart: 10000,
		RTPPortRangeEnd:   10020,
	}
}

// newBridgeTestSession creates a session with an in-memory RTP handler attached.
func newBridgeTestSession(t *testing.T, direction CallDirection, codec *Codec) (*Session, *RTPHandler) {
	t.Helper()
	s, err := NewSession(context.Background(),
		WithSessionConfig(bridgeTestConfig()),
		WithSessionDirection(direction),
		WithSessionCodec(codec),
	)
	require.NoError(t, err)
	s.SetState(CallStateConnected)
	rtp := newTestRTPHandler()
	s.SetRTPHandler(rtp)
	if codec != nil {
		s.SetNegotiatedCodec(codec.Name, int(codec.ClockRate))
	}
	return s, rtp
}

func bridgeTestServer() *Server {
	return &Server{logger: bridgeTestLogger()}
}

// =============================================================================
// transcodeG711 — codec pairs
// =============================================================================

func TestTranscodeG711_SameCodecPassthrough(t *testing.T) {
	t.Parallel()
	srv := bridgeTestServer()
	data := []byte{0xFF, 0x7F, 0x00, 0xAA}

	out := srv.transcodeG711(data, &CodecPCMU, &CodecPCMU)
	assert.Equal(t, data, out, "PCMU→PCMU should passthrough")

	out = srv.transcodeG711(data, &CodecPCMA, &CodecPCMA)
	assert.Equal(t, data, out, "PCMA→PCMA should passthrough")
}

func TestTranscodeG711_PCMAtoPCMU(t *testing.T) {
	t.Parallel()
	srv := bridgeTestServer()
	alaw := []byte{0xD5, 0xD5, 0xD5, 0xD5}
	out := srv.transcodeG711(alaw, &CodecPCMA, &CodecPCMU)
	require.Len(t, out, len(alaw))
	assert.NotEqual(t, alaw, out, "should be transcoded")
}

func TestTranscodeG711_PCMUtoPCMA(t *testing.T) {
	t.Parallel()
	srv := bridgeTestServer()
	ulaw := []byte{0xFF, 0xFF, 0xFF, 0xFF}
	out := srv.transcodeG711(ulaw, &CodecPCMU, &CodecPCMA)
	require.Len(t, out, len(ulaw))
	assert.NotEqual(t, ulaw, out, "should be transcoded")
}

func TestTranscodeG711_Roundtrip(t *testing.T) {
	t.Parallel()
	srv := bridgeTestServer()
	original := make([]byte, 160)
	for i := range original {
		original[i] = byte(i)
	}
	alaw := srv.transcodeG711(original, &CodecPCMU, &CodecPCMA)
	back := srv.transcodeG711(alaw, &CodecPCMA, &CodecPCMU)
	require.Len(t, back, len(original), "roundtrip must preserve length")
}

func TestTranscodeG711_UnknownCodecPassthrough(t *testing.T) {
	t.Parallel()
	srv := bridgeTestServer()
	data := []byte{0x01, 0x02}
	g722 := &Codec{Name: "G722", PayloadType: 9, ClockRate: 16000}
	out := srv.transcodeG711(data, g722, &CodecPCMU)
	assert.Equal(t, data, out, "unknown codec pair should passthrough")
}

func TestTranscodeG711_EmptyData(t *testing.T) {
	t.Parallel()
	srv := bridgeTestServer()
	out := srv.transcodeG711([]byte{}, &CodecPCMA, &CodecPCMU)
	assert.Empty(t, out)
}

// =============================================================================
// forwardBridgeAudio
// =============================================================================

func TestForwardBridgeAudio_PassthroughSameCodec(t *testing.T) {
	t.Parallel()
	srv := bridgeTestServer()

	src := make(chan []byte, 10)
	dst := newBridgeAudioSink(10)
	ctx, cancel := context.WithCancel(context.Background())

	go srv.forwardBridgeAudio(ctx, src, dst, false, &CodecPCMU, &CodecPCMU, nil)

	for i := 0; i < 5; i++ {
		src <- []byte{byte(i), byte(i + 1)}
	}
	for i := 0; i < 5; i++ {
		select {
		case frame := <-dst.frames:
			assert.Equal(t, []byte{byte(i), byte(i + 1)}, frame)
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("timeout waiting for frame %d", i)
		}
	}
	cancel()
}

func TestForwardBridgeAudio_TranscodesWhenNeeded(t *testing.T) {
	t.Parallel()
	srv := bridgeTestServer()

	src := make(chan []byte, 10)
	dst := newBridgeAudioSink(10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go srv.forwardBridgeAudio(ctx, src, dst, true, &CodecPCMA, &CodecPCMU, nil)

	alaw := []byte{0xD5, 0xD5}
	src <- alaw

	select {
	case frame := <-dst.frames:
		assert.NotEqual(t, alaw, frame, "should be transcoded")
		assert.Len(t, frame, len(alaw))
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for transcoded frame")
	}
}

func TestForwardBridgeAudio_ExitsOnContextCancel(t *testing.T) {
	t.Parallel()
	srv := bridgeTestServer()

	src := make(chan []byte, 10)
	dst := newBridgeAudioSink(10)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		srv.forwardBridgeAudio(ctx, src, dst, false, &CodecPCMU, &CodecPCMU, nil)
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("did not exit after context cancel")
	}
}

func TestForwardBridgeAudio_ExitsOnSrcClose(t *testing.T) {
	t.Parallel()
	srv := bridgeTestServer()

	src := make(chan []byte, 10)
	dst := newBridgeAudioSink(10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		srv.forwardBridgeAudio(ctx, src, dst, false, &CodecPCMU, &CodecPCMU, nil)
		close(done)
	}()

	close(src)

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("did not exit after src close")
	}
}

func TestForwardBridgeAudio_DropsFrameWhenDstFull(t *testing.T) {
	t.Parallel()
	srv := bridgeTestServer()

	src := make(chan []byte, 10)
	dst := newBridgeAudioSink(1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go srv.forwardBridgeAudio(ctx, src, dst, false, &CodecPCMU, &CodecPCMU, nil)

	// Fill dst
	src <- []byte{0x01}
	time.Sleep(10 * time.Millisecond)

	// Send more — should be dropped, not block
	for i := 0; i < 5; i++ {
		src <- []byte{byte(i + 2)}
	}

	select {
	case frame := <-dst.frames:
		assert.Equal(t, []byte{0x01}, frame)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("forwardBridgeAudio is blocked")
	}
}

func TestForwardBridgeAudio_DoesNotRecordDroppedFrame(t *testing.T) {
	t.Parallel()
	srv := bridgeTestServer()

	src := make(chan []byte, 1)
	dst := newBridgeAudioSink(1)
	recorded := make(chan []byte, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dst.frames <- []byte{0xff}
	go srv.forwardBridgeAudio(ctx, src, dst, false, &CodecPCMU, &CodecPCMU, func(audio []byte) {
		recorded <- audio
	})

	src <- []byte{0x01}
	require.Eventually(t, func() bool {
		return len(src) == 0
	}, 100*time.Millisecond, 5*time.Millisecond)

	select {
	case audio := <-recorded:
		t.Fatalf("dropped bridge RTP frame was recorded: %v", audio)
	default:
	}
}

// =============================================================================
// BridgeTransfer
// =============================================================================

func TestBridgeTransfer_NilInboundRTP_EndsBothSessions(t *testing.T) {
	t.Parallel()
	srv := bridgeTestServer()

	inbound := newInboundTestSession(t) // no RTP handler
	outbound, _ := newBridgeTestSession(t, CallDirectionOutbound, &CodecPCMU)

	_, err := srv.BridgeTransfer(context.Background(), inbound, outbound, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRTPNotInitialized)
	assert.True(t, inbound.IsEnded())
	assert.True(t, outbound.IsEnded())
}

func TestBridgeTransfer_NilOutboundRTP_EndsBothSessions(t *testing.T) {
	t.Parallel()
	srv := bridgeTestServer()

	inbound, _ := newBridgeTestSession(t, CallDirectionInbound, &CodecPCMU)
	outbound := newInboundTestSession(t) // no RTP handler

	_, err := srv.BridgeTransfer(context.Background(), inbound, outbound, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRTPNotInitialized)
	assert.True(t, inbound.IsEnded())
	assert.True(t, outbound.IsEnded())
}

type bridgeResult struct {
	reason BridgeEndReason
	err    error
}

func requireBridgeConnected(t *testing.T, inbound, outbound *Session) {
	t.Helper()
	require.Eventually(t, func() bool {
		return inbound.GetState() == CallStateBridgeConnected && outbound.GetState() == CallStateBridgeConnected
	}, time.Second, 10*time.Millisecond)
}

func TestBridgeTransfer_ContextCancellation(t *testing.T) {
	t.Parallel()
	srv := bridgeTestServer()

	inbound, _ := newBridgeTestSession(t, CallDirectionInbound, &CodecPCMU)
	outbound, _ := newBridgeTestSession(t, CallDirectionOutbound, &CodecPCMU)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan bridgeResult, 1)
	go func() {
		r, err := srv.BridgeTransfer(ctx, inbound, outbound, nil)
		done <- bridgeResult{r, err}
	}()

	requireBridgeConnected(t, inbound, outbound)
	cancel()

	select {
	case res := <-done:
		assert.NoError(t, res.err)
		assert.Equal(t, BridgeEndContext, res.reason)
	case <-time.After(1 * time.Second):
		t.Fatal("BridgeTransfer did not exit after context cancel")
	}
	assert.False(t, inbound.IsEnded(), "BridgeTransfer must NOT end the inbound session")
	assert.False(t, outbound.IsEnded(), "BridgeTransfer must NOT end the outbound session — caller owns it")
	assert.Equal(t, CallStateBridgeConnected, inbound.GetState())
	assert.Equal(t, CallStateBridgeConnected, outbound.GetState())
}

func TestBridgeTransfer_OwnsBridgeLifecycle(t *testing.T) {
	t.Parallel()
	srv := bridgeTestServer()

	inbound, _ := newBridgeTestSession(t, CallDirectionInbound, &CodecPCMU)
	outbound, _ := newBridgeTestSession(t, CallDirectionOutbound, &CodecPCMU)

	require.True(t, srv.TransitionCall(inbound, CallStateTransferring, LifecycleReasonTransferModeStarted))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan bridgeResult, 1)
	go func() {
		reason, err := srv.BridgeTransfer(ctx, inbound, outbound, nil)
		done <- bridgeResult{reason: reason, err: err}
	}()

	requireBridgeConnected(t, inbound, outbound)
	cancel()

	select {
	case result := <-done:
		assert.NoError(t, result.err)
		assert.Equal(t, BridgeEndContext, result.reason)
	case <-time.After(time.Second):
		t.Fatal("BridgeTransfer did not exit after context cancel")
	}
}

func TestBridgeTransfer_RejectsPreOwnedBridgeState(t *testing.T) {
	t.Parallel()
	srv := bridgeTestServer()

	inbound, _ := newBridgeTestSession(t, CallDirectionInbound, &CodecPCMU)
	outbound, _ := newBridgeTestSession(t, CallDirectionOutbound, &CodecPCMU)
	inbound.SetState(CallStateTransferring)
	inbound.SetState(CallStateBridgeConnected)

	_, err := srv.BridgeTransfer(context.Background(), inbound, outbound, nil)

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBridgeLifecycleRejected))
	assert.True(t, inbound.IsEnded())
	assert.True(t, outbound.IsEnded())
}

func TestBridgeTransfer_InboundByeEndsBridge(t *testing.T) {
	t.Parallel()
	srv := bridgeTestServer()

	inbound, _ := newBridgeTestSession(t, CallDirectionInbound, &CodecPCMU)
	outbound, _ := newBridgeTestSession(t, CallDirectionOutbound, &CodecPCMU)

	done := make(chan bridgeResult, 1)
	go func() {
		r, err := srv.BridgeTransfer(context.Background(), inbound, outbound, nil)
		done <- bridgeResult{r, err}
	}()

	requireBridgeConnected(t, inbound, outbound)
	inbound.NotifyBye()

	select {
	case res := <-done:
		assert.NoError(t, res.err)
		assert.Equal(t, BridgeEndInboundBye, res.reason)
	case <-time.After(1 * time.Second):
		t.Fatal("BridgeTransfer did not exit after inbound BYE")
	}
	assert.False(t, outbound.IsEnded(), "outbound lifecycle is owned by caller")
}

func TestBridgeTransfer_OutboundByeEndsBridge(t *testing.T) {
	t.Parallel()
	srv := bridgeTestServer()

	inbound, _ := newBridgeTestSession(t, CallDirectionInbound, &CodecPCMU)
	outbound, _ := newBridgeTestSession(t, CallDirectionOutbound, &CodecPCMU)

	done := make(chan bridgeResult, 1)
	go func() {
		r, err := srv.BridgeTransfer(context.Background(), inbound, outbound, nil)
		done <- bridgeResult{r, err}
	}()

	requireBridgeConnected(t, inbound, outbound)
	outbound.NotifyBye()

	select {
	case res := <-done:
		assert.NoError(t, res.err)
		assert.Equal(t, BridgeEndOutboundBye, res.reason)
	case <-time.After(1 * time.Second):
		t.Fatal("BridgeTransfer did not exit after outbound BYE")
	}
	assert.False(t, inbound.IsEnded(), "BridgeTransfer must NOT end the inbound session")
	assert.False(t, outbound.IsEnded(), "BridgeTransfer must NOT end the outbound session — caller owns it")
}

func TestBridgeTransfer_SessionEndTerminatesBridge(t *testing.T) {
	t.Parallel()
	srv := bridgeTestServer()

	inbound, _ := newBridgeTestSession(t, CallDirectionInbound, &CodecPCMU)
	outbound, _ := newBridgeTestSession(t, CallDirectionOutbound, &CodecPCMU)

	done := make(chan bridgeResult, 1)
	go func() {
		r, err := srv.BridgeTransfer(context.Background(), inbound, outbound, nil)
		done <- bridgeResult{r, err}
	}()

	requireBridgeConnected(t, inbound, outbound)
	inbound.End()

	select {
	case res := <-done:
		assert.NoError(t, res.err)
		assert.Equal(t, BridgeEndInboundBye, res.reason)
	case <-time.After(1 * time.Second):
		t.Fatal("BridgeTransfer did not exit after inbound End()")
	}
	assert.False(t, outbound.IsEnded(), "outbound lifecycle is owned by caller")
}

func TestBridgeTransfer_AudioForwardsBidirectionally(t *testing.T) {
	t.Parallel()
	srv := bridgeTestServer()

	inbound, inRTP := newBridgeTestSession(t, CallDirectionInbound, &CodecPCMU)
	outbound, outRTP := newBridgeTestSession(t, CallDirectionOutbound, &CodecPCMU)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan bridgeResult, 1)
	go func() {
		r, err := srv.BridgeTransfer(ctx, inbound, outbound, nil)
		done <- bridgeResult{r, err}
	}()

	// outbound → inbound (inbound→outbound is handled by the telephony media port)
	outRTP.audioInChan <- []byte{0x03, 0x04}
	select {
	case frame := <-inRTP.audioOutChan:
		assert.Equal(t, []byte{0x03, 0x04}, frame)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("audio not forwarded outbound→inbound")
	}

	cancel()
	<-done
}

func TestBridgeTransfer_TranscodesAcrossCodecs(t *testing.T) {
	t.Parallel()
	srv := bridgeTestServer()

	inbound, inRTP := newBridgeTestSession(t, CallDirectionInbound, &CodecPCMA)
	outbound, outRTP := newBridgeTestSession(t, CallDirectionOutbound, &CodecPCMU)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan bridgeResult, 1)
	go func() {
		r, err := srv.BridgeTransfer(ctx, inbound, outbound, nil)
		done <- bridgeResult{r, err}
	}()

	// µ-law from outbound → A-law on inbound
	ulaw := []byte{0xFF, 0xFF, 0xFF, 0xFF}
	outRTP.audioInChan <- ulaw
	select {
	case frame := <-inRTP.audioOutChan:
		assert.Len(t, frame, len(ulaw))
		assert.NotEqual(t, ulaw, frame, "should be transcoded PCMU→PCMA")
	case <-time.After(200 * time.Millisecond):
		t.Fatal("transcoded audio not forwarded")
	}

	cancel()
	<-done
}

func TestBridgeTransfer_AlreadyEndedSessions(t *testing.T) {
	t.Parallel()
	srv := bridgeTestServer()

	inbound, _ := newBridgeTestSession(t, CallDirectionInbound, &CodecPCMU)
	outbound, _ := newBridgeTestSession(t, CallDirectionOutbound, &CodecPCMU)

	inbound.End()
	outbound.End()

	// RTP handler is nil after End() clears it
	_, err := srv.BridgeTransfer(context.Background(), inbound, outbound, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRTPNotInitialized)
}

// =============================================================================
// BridgeTransfer — graceful transfer contract (outbound-only teardown)
// =============================================================================

func TestBridgeTransfer_OnlyEndsOutboundSession(t *testing.T) {
	t.Parallel()
	srv := bridgeTestServer()

	inbound, _ := newBridgeTestSession(t, CallDirectionInbound, &CodecPCMU)
	outbound, _ := newBridgeTestSession(t, CallDirectionOutbound, &CodecPCMU)

	done := make(chan bridgeResult, 1)
	go func() {
		r, err := srv.BridgeTransfer(context.Background(), inbound, outbound, nil)
		done <- bridgeResult{r, err}
	}()

	requireBridgeConnected(t, inbound, outbound)

	// Simulate the transfer target ending the call through session lifecycle.
	outbound.End()

	select {
	case res := <-done:
		assert.NoError(t, res.err)
		assert.Equal(t, BridgeEndOutboundBye, res.reason)
	case <-time.After(1 * time.Second):
		t.Fatal("BridgeTransfer did not exit after outbound end")
	}

	assert.True(t, outbound.IsEnded(), "outbound must be ended")
	assert.False(t, inbound.IsEnded(), "inbound must NOT be ended — caller owns its lifecycle")
}

func TestBridgeTransfer_InboundBye_DoesNotEndEitherSession(t *testing.T) {
	t.Parallel()
	srv := bridgeTestServer()

	inbound, _ := newBridgeTestSession(t, CallDirectionInbound, &CodecPCMU)
	outbound, _ := newBridgeTestSession(t, CallDirectionOutbound, &CodecPCMU)

	done := make(chan bridgeResult, 1)
	go func() {
		r, err := srv.BridgeTransfer(context.Background(), inbound, outbound, nil)
		done <- bridgeResult{r, err}
	}()

	requireBridgeConnected(t, inbound, outbound)
	inbound.NotifyBye()

	select {
	case res := <-done:
		assert.NoError(t, res.err)
		assert.Equal(t, BridgeEndInboundBye, res.reason)
	case <-time.After(1 * time.Second):
		t.Fatal("BridgeTransfer did not exit after inbound BYE")
	}

	// Both session lifecycles are owned by the caller (executeTransfer); BridgeTransfer
	// no longer ends either side on normal exit.
	assert.False(t, outbound.IsEnded(), "outbound lifecycle is owned by caller")
	assert.False(t, inbound.IsEnded(), "inbound must NOT be ended — NotifyBye is not End()")
}

func TestForwardBridgeAudio_Passthrough_10Frames(t *testing.T) {
	t.Parallel()
	srv := bridgeTestServer()

	src := make(chan []byte, 20)
	dst := newBridgeAudioSink(20)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go srv.forwardBridgeAudio(ctx, src, dst, false, &CodecPCMU, &CodecPCMU, nil)

	const frameCount = 10
	for i := 0; i < frameCount; i++ {
		src <- []byte{byte(i), byte(i * 2)}
	}

	for i := 0; i < frameCount; i++ {
		select {
		case frame := <-dst.frames:
			assert.Equal(t, []byte{byte(i), byte(i * 2)}, frame, "frame %d mismatch", i)
		case <-time.After(200 * time.Millisecond):
			t.Fatalf("timeout waiting for frame %d of %d", i, frameCount)
		}
	}
}

func TestForwardBridgeAudio_ContextCancel_NoHang(t *testing.T) {
	t.Parallel()
	srv := bridgeTestServer()

	src := make(chan []byte) // unbuffered — blocks if forwardBridgeAudio tries to read
	dst := newBridgeAudioSink(10)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		srv.forwardBridgeAudio(ctx, src, dst, false, &CodecPCMU, &CodecPCMU, nil)
		close(done)
	}()

	// Cancel immediately — forwardBridgeAudio must not block
	cancel()

	select {
	case <-done:
		// success
	case <-time.After(200 * time.Millisecond):
		t.Fatal("forwardBridgeAudio did not exit promptly after context cancel")
	}

	// dst should be empty — no frames were sent
	select {
	case <-dst.frames:
		t.Fatal("unexpected frame on dst after cancel")
	default:
		// expected
	}
}

// =============================================================================
// codecName
// =============================================================================

func TestCodecName_Nil(t *testing.T) {
	srv := bridgeTestServer()
	assert.Equal(t, "PCMU", srv.codecName(nil))
}

func TestCodecName_PCMA(t *testing.T) {
	srv := bridgeTestServer()
	assert.Equal(t, "PCMA", srv.codecName(&CodecPCMA))
}

// =============================================================================
// outboundMedia cleanup
// =============================================================================

func TestOutboundMediaStop_StopsRTPHandler(t *testing.T) {
	t.Parallel()
	rtp := newTestRTPHandler()
	require.True(t, rtp.running.Load())

	media := &outboundMedia{
		rtpHandler: rtp,
	}
	media.Stop()
	assert.False(t, rtp.running.Load(), "RTP handler should be stopped")
}

func TestOutboundMediaStop_DoesNotStopSessionAdoptedRTPHandler(t *testing.T) {
	t.Parallel()
	session := newInboundTestSession(t)
	rtp := newTestRTPHandler()
	session.SetRTPHandler(rtp)

	media := &outboundMedia{
		session:    session,
		rtpHandler: rtp,
	}
	media.Stop()

	assert.True(t, rtp.running.Load(), "adopted RTP handler should be stopped by Session.End")
	session.End()
	assert.False(t, rtp.running.Load())
}

func TestOutboundMediaStop_NilRTPHandler(t *testing.T) {
	t.Parallel()
	media := &outboundMedia{
		rtpHandler: nil,
	}
	// Should not panic
	media.Stop()
}

func TestOutboundMediaStop_DoubleStop(t *testing.T) {
	t.Parallel()
	rtp := newTestRTPHandler()
	media := &outboundMedia{
		rtpHandler: rtp,
	}
	media.Stop()
	media.Stop() // second call should not panic
	assert.False(t, rtp.running.Load())
}
