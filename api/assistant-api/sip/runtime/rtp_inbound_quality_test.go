package sip_runtime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRTPInboundQualitySnapshotScoresWindow(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name         string
		record       func(*rtpInboundQuality)
		wantQuality  string
		wantScore    uint8
		wantLossRate float64
		wantDropRate float64
	}{
		{
			name: "excellent when all received audio is delivered",
			record: func(quality *rtpInboundQuality) {
				quality.recordReceived(now, 100)
				quality.recordDelivered(now, 100)
			},
			wantQuality: rtpInboundQualityExcellent,
			wantScore:   100,
		},
		{
			name: "good when loss crosses warning threshold",
			record: func(quality *rtpInboundQuality) {
				quality.recordReceived(now, 100)
				quality.recordDelivered(now, 94)
				quality.recordLost(now, 6)
			},
			wantQuality:  rtpInboundQualityGood,
			wantScore:    80,
			wantLossRate: 6.0 / 106.0,
		},
		{
			name: "poor when queue drops cross poor threshold",
			record: func(quality *rtpInboundQuality) {
				quality.recordReceived(now, 10)
				quality.recordDelivered(now, 8)
				quality.recordAudioInputDropped(now, 2)
			},
			wantQuality:  rtpInboundQualityPoor,
			wantScore:    50,
			wantLossRate: 2.0 / 10.0,
			wantDropRate: 2.0 / 10.0,
		},
		{
			name: "lost when nothing can be delivered",
			record: func(quality *rtpInboundQuality) {
				quality.recordReceived(now, 1)
				quality.recordAudioInputDropped(now, 1)
			},
			wantQuality:  rtpInboundQualityLost,
			wantLossRate: 1,
			wantDropRate: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var quality rtpInboundQuality
			test.record(&quality)

			snapshot := quality.snapshot(now.Add(time.Second))

			assert.Equal(t, test.wantQuality, snapshot.quality)
			assert.Equal(t, test.wantScore, snapshot.score)
			assert.InDelta(t, test.wantLossRate, snapshot.lossRate, 0.0001)
			assert.InDelta(t, test.wantDropRate, snapshot.dropRate, 0.0001)
		})
	}
}

func TestRTPInboundQualitySnapshotMarksDryWindowLost(t *testing.T) {
	now := time.Now()
	var quality rtpInboundQuality
	quality.recordReceived(now, 1)
	quality.recordDelivered(now, 1)

	snapshot := quality.snapshot(now.Add(rtpInboundQualityWindow + time.Millisecond))

	assert.Equal(t, rtpInboundQualityLost, snapshot.quality)
	assert.Equal(t, uint8(0), snapshot.score)
	assert.Zero(t, snapshot.packetsReceived)
	assert.Zero(t, snapshot.packetsDelivered)
}
