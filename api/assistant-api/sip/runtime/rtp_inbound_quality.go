package sip_runtime

import (
	"sync"
	"time"
)

const (
	rtpInboundQualityWindow       = 5 * time.Second
	rtpInboundQualityGoodLossRate = 0.05
	rtpInboundQualityPoorLossRate = 0.12

	rtpInboundQualityUnknown   = "unknown"
	rtpInboundQualityExcellent = "excellent"
	rtpInboundQualityGood      = "good"
	rtpInboundQualityPoor      = "poor"
	rtpInboundQualityLost      = "lost"
)

type rtpInboundQuality struct {
	mu sync.Mutex

	window         rtpInboundQualityWindowStats
	lastDelivered  time.Time
	lastPacketSeen time.Time
}

type rtpInboundQualityWindowStats struct {
	startedAt         time.Time
	packetsReceived   uint64
	packetsDelivered  uint64
	packetsLost       uint64
	packetsDropped    uint64
	audioInputDropped uint64
}

type rtpInboundQualitySnapshot struct {
	quality           string
	score             uint8
	window            time.Duration
	packetsReceived   uint64
	packetsDelivered  uint64
	packetsLost       uint64
	packetsDropped    uint64
	audioInputDropped uint64
	lossRate          float64
	dropRate          float64
	deliveryRate      float64
}

func (q *rtpInboundQuality) recordReceived(now time.Time, count uint64) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.prepareWindowLocked(now)
	q.window.packetsReceived += count
	q.lastPacketSeen = now
}

func (q *rtpInboundQuality) recordDelivered(now time.Time, count uint64) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.prepareWindowLocked(now)
	q.window.packetsDelivered += count
	q.lastDelivered = now
}

func (q *rtpInboundQuality) recordLost(now time.Time, count uint64) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.prepareWindowLocked(now)
	q.window.packetsLost += count
}

func (q *rtpInboundQuality) recordDropped(now time.Time, count uint64) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.prepareWindowLocked(now)
	q.window.packetsDropped += count
}

func (q *rtpInboundQuality) recordAudioInputDropped(now time.Time, count uint64) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.prepareWindowLocked(now)
	q.window.audioInputDropped += count
}

func (q *rtpInboundQuality) snapshot(now time.Time) rtpInboundQualitySnapshot {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.prepareWindowLocked(now)

	window := now.Sub(q.window.startedAt)
	if q.window.startedAt.IsZero() || window < 0 {
		window = 0
	}
	if window > rtpInboundQualityWindow {
		window = rtpInboundQualityWindow
	}

	expectedPackets := q.window.packetsReceived + q.window.packetsLost + q.window.packetsDropped
	impairedPackets := q.window.packetsLost + q.window.packetsDropped + q.window.audioInputDropped
	deliveryAttempts := q.window.packetsDelivered + q.window.audioInputDropped

	snapshot := rtpInboundQualitySnapshot{
		window:            window,
		packetsReceived:   q.window.packetsReceived,
		packetsDelivered:  q.window.packetsDelivered,
		packetsLost:       q.window.packetsLost,
		packetsDropped:    q.window.packetsDropped,
		audioInputDropped: q.window.audioInputDropped,
		lossRate:          rtpRatio(impairedPackets, expectedPackets),
		dropRate:          rtpRatio(q.window.audioInputDropped, deliveryAttempts),
		deliveryRate:      rtpRatio(q.window.packetsDelivered, deliveryAttempts),
	}

	switch {
	case expectedPackets == 0 && impairedPackets > 0:
		snapshot.quality = rtpInboundQualityLost
		snapshot.score = 0
	case expectedPackets == 0 && !q.lastDelivered.IsZero() && now.Sub(q.lastDelivered) > rtpInboundQualityWindow:
		snapshot.quality = rtpInboundQualityLost
		snapshot.score = 0
	case expectedPackets == 0 && !q.lastPacketSeen.IsZero() && now.Sub(q.lastPacketSeen) > rtpInboundQualityWindow:
		snapshot.quality = rtpInboundQualityLost
		snapshot.score = 0
	case expectedPackets == 0:
		snapshot.quality = rtpInboundQualityUnknown
		snapshot.score = 100
	case q.window.packetsDelivered == 0 && impairedPackets > 0:
		snapshot.quality = rtpInboundQualityLost
		snapshot.score = 0
	case snapshot.lossRate >= rtpInboundQualityPoorLossRate || snapshot.dropRate >= rtpInboundQualityPoorLossRate:
		snapshot.quality = rtpInboundQualityPoor
		snapshot.score = 50
	case snapshot.lossRate >= rtpInboundQualityGoodLossRate || snapshot.dropRate >= rtpInboundQualityGoodLossRate:
		snapshot.quality = rtpInboundQualityGood
		snapshot.score = 80
	default:
		snapshot.quality = rtpInboundQualityExcellent
		snapshot.score = 100
	}

	return snapshot
}

func (q *rtpInboundQuality) prepareWindowLocked(now time.Time) {
	if q.window.startedAt.IsZero() {
		q.window.startedAt = now
		return
	}
	if now.Sub(q.window.startedAt) < rtpInboundQualityWindow {
		return
	}
	q.window = rtpInboundQualityWindowStats{startedAt: now}
}

func rtpRatio(value uint64, total uint64) float64 {
	if total == 0 {
		return 0
	}
	return float64(value) / float64(total)
}
