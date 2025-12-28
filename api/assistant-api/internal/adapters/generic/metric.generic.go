// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_adapter_request_generic

import (
	"time"

	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
)

// for the conversation metrics
// for adding another metrics
// -----------------------------------------------------------------------------
// Metrics Management
// -----------------------------------------------------------------------------
//
// The following methods are responsible for managing metrics associated with
// the GenericRequestor. Metrics provide valuable insights into the
// conversation's performance, usage, and other relevant statistics.
//
// GetMetrics retrieves the current set of metrics associated with this
// conversation. It returns a slice of Metric pointers, allowing the caller
// to access and analyze various aspects of the conversation's performance.
//
// AddMetrics allows for the addition of new metrics to the conversation.
// This method can be used to update or extend the existing set of metrics
// with new data points or measurements.
//
// These methods play a crucial role in monitoring and analyzing the behavior
// and performance of the GenericRequestor, enabling data-driven
// improvements and optimizations.
//
// -----------------------------------------------------------------------------

func (tc *GenericRequestor) GetMetrics() []*types.Metric {
	return tc.metrics
}

func (tc *GenericRequestor) AddMetrics(
	auth types.SimplePrinciple,
	metrics ...*types.Metric) {
	tc.metrics = append(tc.metrics, metrics...)
	utils.Go(tc.ctx, func() {
		start := time.Now()
		_, err := tc.conversationService.ApplyConversationMetrics(
			tc.ctx,
			auth,
			tc.assistant.Id,
			tc.assistantConversation.Id,
			metrics,
		)
		tc.logger.Benchmark("GenericRequestor.AddMetrics", time.Since(start))
		if err != nil {
			tc.logger.Errorf("unable to flush metrics for conversation %+v", err)
		}
	})
}

func (tc *GenericRequestor) AddMetric(
	auth types.SimplePrinciple,
	metric *types.Metric) {
	tc.metrics = append(tc.metrics, metric)
	utils.Go(tc.ctx, func() {
		start := time.Now()
		_, err := tc.conversationService.ApplyConversationMetrics(
			tc.ctx,
			auth,
			tc.assistant.Id,
			tc.assistantConversation.Id,
			[]*types.Metric{metric},
		)
		tc.logger.Benchmark("GenericRequestor.AddMetric", time.Since(start))
		if err != nil {
			tc.logger.Errorf("unable to flush metrics for conversation %+v", err)
		}

	})

}
