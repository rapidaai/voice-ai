// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_registration

// Pipeline is the base interface for every step in the registration pipeline.
// Mirrors the typed Pipeline pattern of sip/runtime/pipeline.go: concrete types
// carry their own payload, the manager's dispatch() switches on type to
// invoke the matching handler, and handlers return the next Pipeline (or nil
// to stop) — chains form without explicit wiring.
type Pipeline interface {
	DID() string
}

// ClaimOwnershipPipeline is the entry point of the chain: per the contract
// "Check if owner is not there -> create owner". Hands off to RegisterPipeline
// on a fresh claim or self-owned refresh; returns nil on peer-owned or claim
// error.
type ClaimOwnershipPipeline struct {
	Record *Record
}

func (p ClaimOwnershipPipeline) DID() string { return p.Record.DID }

// RegisterPipeline performs the actual SIP REGISTER via the registration
// client. Hands off to MarkActivePipeline on success; on terminal/transient
// failure returns nil (the handler writes its own status before stopping).
type RegisterPipeline struct {
	Record *Record
}

func (p RegisterPipeline) DID() string { return p.Record.DID }

// MarkActivePipeline writes sip_status='active' (and clears error/retry
// counters) to the deployment options. Skips the DB write if the DID was
// already locally active — the renewal loop still owns the binding and DB
// state is unchanged.
type MarkActivePipeline struct {
	Record *Record
}

func (p MarkActivePipeline) DID() string { return p.Record.DID }
