// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package gorm_generator

import (
	"errors"
	"sync/atomic"
	"time"
)

var lastTime int64
var lastSeq uint32

// AtomicResolver define as atomic sequence resolver, base on standard sync/atomic.
func AtomicResolver(ms int64) (uint16, error) {
	var last int64
	var seq, localSeq uint32

	for {
		last = atomic.LoadInt64(&lastTime)
		localSeq = atomic.LoadUint32(&lastSeq)
		if last > ms {
			return MaxSequence, nil
		}

		if last == ms {
			seq = uint32(MaxSequence) & (localSeq + 1)
			if seq == 0 {
				return MaxSequence, nil
			}
		}

		if atomic.CompareAndSwapInt64(&lastTime, last, ms) && atomic.CompareAndSwapUint32(&lastSeq, localSeq, seq) {
			return uint16(seq), nil
		}
	}
}

// These constants are the bit lengths of snowflake ID parts.
const (
	TimestampLength uint8  = 41
	MachineIDLength uint8  = 10
	SequenceLength  uint8  = 12
	MaxSequence     uint16 = 1<<SequenceLength - 1
	MaxTimestamp    uint64 = 1<<TimestampLength - 1
	MaxMachineID    uint16 = 1<<MachineIDLength - 1

	machineIDMoveLength = SequenceLength
	timestampMoveLength = MachineIDLength + SequenceLength
)

// SequenceResolver the snowflake sequence resolver.
//
// When you want to use the snowflake algorithm to generate unique ID, You must ensure: The sequence-number generated in the same millisecond of the same node is unique.
// Based on this, we create this interface provide following resolver:
//
//	AtomicResolver : base sync/atomic (by default).
type SequenceResolver func(ms int64) (uint16, error)

// default start time is 2008-11-10 23:00:00 UTC, why ? In the playground the time begins at 2009-11-10 23:00:00 UTC.
// It can run on golang playground.
// default machineID is 0
// default resolver is AtomicResolver
var (
	resolver  SequenceResolver
	machineID uint64 = 0
	startTime        = time.Date(2008, 11, 10, 23, 0, 0, 0, time.UTC)
)

// ID use ID to generate snowflake id, and it will ignore error. if you want error info, you need use NextID method.
// This function is thread safe.
func ID() uint64 {
	id, _ := NextID()
	return id
}

// NextID use NextID to generate snowflake id and return an error.
// This function is thread safe.
func NextID() (uint64, error) {
	c := currentMillis()
	seqResolver := callSequenceResolver()
	seq, err := seqResolver(c)

	if err != nil {
		return 0, err
	}

	for seq >= MaxSequence {
		c = waitForNextMillis(c)
		seq, err = seqResolver(c)
		if err != nil {
			return 0, err
		}
	}

	df := elapsedTime(c, startTime)
	if df < 0 || uint64(df) > MaxTimestamp {
		return 0, errors.New("the maximum life cycle of the snowflake algorithm is 2^41-1(millis), please check start-time")
	}

	id := (uint64(df) << uint64(timestampMoveLength)) | (machineID << uint64(machineIDMoveLength)) | uint64(seq)
	return id, nil
}

// SetMachineID specify the machine ID. It will panic when machined > max limit for 2^10-1.
// This function is thread-unsafe, recommended you call him in the main function.
func SetMachineID(m uint16) {
	if m > MaxMachineID {
		panic("The machineID cannot be greater than 1023")
	}
	machineID = uint64(m)
}

// SetSequenceResolver set a custom sequence resolver.
// This function is thread-unsafe, recommended you call him in the main function.
func SetSequenceResolver(seq SequenceResolver) {
	if seq != nil {
		resolver = seq
	}
}

// SID snowflake id
type SID struct {
	Sequence  uint64
	MachineID uint64
	Timestamp uint64
	ID        uint64
}

// GenerateTime snowflake generate at, return a UTC time.
func (id *SID) GenerateTime() time.Time {
	ms := startTime.UTC().UnixNano()/1e6 + int64(id.Timestamp)

	return time.Unix(0, ms*int64(time.Millisecond)).UTC()
}

// ParseID parse snowflake it to SID struct.
func ParseID(id uint64) SID {
	t := id >> uint64(SequenceLength+MachineIDLength)
	sequence := id & uint64(MaxSequence)
	mID := (id & (uint64(MaxMachineID) << SequenceLength)) >> SequenceLength

	return SID{
		ID:        id,
		Sequence:  sequence,
		MachineID: mID,
		Timestamp: t,
	}
}

//--------------------------------------------------------------------
// private function defined.
//--------------------------------------------------------------------

func waitForNextMillis(last int64) int64 {
	now := currentMillis()
	for now == last {
		now = currentMillis()
	}
	return now
}

func callSequenceResolver() SequenceResolver {
	if resolver == nil {
		return AtomicResolver
	}

	return resolver
}

func elapsedTime(noms int64, s time.Time) int64 {
	return noms - s.UTC().UnixNano()/1e6
}

// currentMillis get current millisecond.
func currentMillis() int64 {
	return time.Now().UTC().UnixNano() / 1e6
}
