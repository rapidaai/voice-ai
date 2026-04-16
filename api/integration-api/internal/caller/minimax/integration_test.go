//go:build integration

// Rapida -- Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.

package internal_minimax_callers

import (
	"context"
	"testing"
	"time"

	testutil "github.com/rapidaai/api/integration-api/internal/caller/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const providerName = "minimax"

// TestIntegration_ChatCompletion verifies non-streaming chat completion: send a
// simple prompt and assert the assistant responds with content and metrics.
func TestIntegration_ChatCompletion(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.ChatProvider(t, providerName)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cred := testutil.BuildCredential(pcfg.Credential)
	caller := NewLargeLanguageCaller(testutil.NewTestLogger(), cred)
	opts := testutil.BuildChatOptions(pcfg)

	msg, metrics, err := caller.GetChatCompletion(ctx, testutil.SimpleMessages(), opts)
	require.NoError(t, err, "GetChatCompletion should succeed")
	require.NotNil(t, msg, "response message should not be nil")

	contents := msg.GetAssistant().GetContents()
	assert.NotEmpty(t, contents, "assistant should return content")
	assert.NotEmpty(t, metrics, "metrics should be returned")
	testutil.AssertHasMetric(t, metrics, "TIME_TAKEN")
	t.Logf("provider=%s response=%q", providerName, contents)
}

// TestIntegration_StreamChatCompletion verifies streaming chat completion: send a
// simple prompt and assert tokens are streamed, metrics are returned.
func TestIntegration_StreamChatCompletion(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.ChatProvider(t, providerName)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cred := testutil.BuildCredential(pcfg.Credential)
	caller := NewLargeLanguageCaller(testutil.NewTestLogger(), cred)
	opts := testutil.BuildChatOptions(pcfg)

	collector := &testutil.StreamCollector{}
	err := caller.StreamChatCompletion(ctx, testutil.SimpleMessages(), opts,
		collector.OnStream, collector.OnMetrics, collector.OnError)
	require.NoError(t, err, "StreamChatCompletion should succeed")
	collector.AssertStream(t)
	t.Logf("provider=%s stream_count=%d", providerName, collector.StreamCount)
}

// TestIntegration_VerifyCredential verifies that valid credentials pass
// the provider's credential verification endpoint without error.
func TestIntegration_VerifyCredential(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.VerifyProvider(t, providerName)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cred := testutil.BuildCredential(pcfg.Credential)
	verifier := NewVerifyCredentialCaller(testutil.NewTestLogger(), cred)
	_, err := verifier.CredentialVerifier(ctx, testutil.BuildVerifyOptions(pcfg))
	require.NoError(t, err, "CredentialVerifier should succeed with valid credentials")
	t.Logf("provider=%s credential_verification=ok", providerName)
}
