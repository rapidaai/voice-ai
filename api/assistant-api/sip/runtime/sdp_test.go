package sip_runtime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSDPIncludesRTCPWhenConfigured(t *testing.T) {
	sdp := (&Server{}).GenerateSDP(&SDPConfig{
		SessionID:   "1",
		SessionName: "call",
		LocalIP:     "127.0.0.1",
		RTPPort:     19000,
		RTCPPort:    19001,
		Codecs:      []Codec{CodecPCMU},
		PTime:       20,
	})

	assert.Contains(t, sdp, "a=rtcp:19001 IN IP4 127.0.0.1\r\n")
}

func TestGenerateSDPOmitsRTCPWhenUnavailable(t *testing.T) {
	sdp := (&Server{}).GenerateSDP(&SDPConfig{
		SessionID:   "1",
		SessionName: "call",
		LocalIP:     "127.0.0.1",
		RTPPort:     19000,
		Codecs:      []Codec{CodecPCMU},
		PTime:       20,
	})

	assert.NotContains(t, sdp, "a=rtcp:")
}

func TestParseSDPParsesRTCPAttribute(t *testing.T) {
	info, err := (&Server{}).ParseSDP([]byte(
		"v=0\r\n" +
			"o=carrier 1 1 IN IP4 127.0.0.1\r\n" +
			"s=call\r\n" +
			"c=IN IP4 127.0.0.1\r\n" +
			"t=0 0\r\n" +
			"m=audio 19000 RTP/AVP 0 101\r\n" +
			"a=rtcp:19001 IN IP4 198.51.100.10\r\n" +
			"a=rtpmap:0 PCMU/8000\r\n" +
			"a=rtpmap:101 telephone-event/8000\r\n" +
			"a=sendrecv\r\n",
	))

	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1", info.ConnectionIP)
	assert.Equal(t, 19000, info.AudioPort)
	assert.Equal(t, "198.51.100.10", info.RTCPIP)
	assert.Equal(t, 19001, info.RTCPPort)
}

func TestParseSDPDefaultsRTCPIPToConnectionIP(t *testing.T) {
	info, err := (&Server{}).ParseSDP([]byte(
		"v=0\r\n" +
			"o=carrier 1 1 IN IP4 127.0.0.1\r\n" +
			"s=call\r\n" +
			"c=IN IP4 127.0.0.1\r\n" +
			"t=0 0\r\n" +
			"m=audio 19000 RTP/AVP 0 101\r\n" +
			"a=rtcp:19001\r\n" +
			"a=rtpmap:0 PCMU/8000\r\n" +
			"a=sendrecv\r\n",
	))

	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1", info.RTCPIP)
	assert.Equal(t, 19001, info.RTCPPort)
}

func TestParseSDPParsesPTime(t *testing.T) {
	info, err := (&Server{}).ParseSDP([]byte(
		"v=0\r\n" +
			"o=carrier 1 1 IN IP4 127.0.0.1\r\n" +
			"s=call\r\n" +
			"c=IN IP4 127.0.0.1\r\n" +
			"t=0 0\r\n" +
			"m=audio 19000 RTP/AVP 0 101\r\n" +
			"a=ptime:30\r\n" +
			"a=rtpmap:0 PCMU/8000\r\n" +
			"a=sendrecv\r\n",
	))

	require.NoError(t, err)
	assert.Equal(t, 30, info.PTime)
	assert.Equal(t, 30*time.Millisecond, info.PacketizationDuration())
}

func TestParseSDPDefaultsPTimeWhenOmitted(t *testing.T) {
	info, err := (&Server{}).ParseSDP([]byte(
		"v=0\r\n" +
			"o=carrier 1 1 IN IP4 127.0.0.1\r\n" +
			"s=call\r\n" +
			"c=IN IP4 127.0.0.1\r\n" +
			"t=0 0\r\n" +
			"m=audio 19000 RTP/AVP 0 101\r\n" +
			"a=rtpmap:0 PCMU/8000\r\n" +
			"a=sendrecv\r\n",
	))

	require.NoError(t, err)
	assert.Equal(t, sdpDefaultPTimeMS, info.PTime)
	assert.Equal(t, rtpDefaultPacketizationTime, info.PacketizationDuration())
}

func TestParseSDPDefaultsPTimeWhenMalformed(t *testing.T) {
	info, err := (&Server{}).ParseSDP([]byte(
		"v=0\r\n" +
			"o=carrier 1 1 IN IP4 127.0.0.1\r\n" +
			"s=call\r\n" +
			"c=IN IP4 127.0.0.1\r\n" +
			"t=0 0\r\n" +
			"m=audio 19000 RTP/AVP 0 101\r\n" +
			"a=ptime:not-a-number\r\n" +
			"a=rtpmap:0 PCMU/8000\r\n" +
			"a=sendrecv\r\n",
	))

	require.NoError(t, err)
	assert.Equal(t, sdpDefaultPTimeMS, info.PTime)
	assert.Equal(t, rtpDefaultPacketizationTime, info.PacketizationDuration())
}
