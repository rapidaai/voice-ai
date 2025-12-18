// Copyright (c) Rapida
// Author: Prashant <prashant@rapida.ai>
//
// Licensed under the Rapida internal use license.
// This file is part of Rapida's proprietary software.
// Unauthorized copying, modification, or redistribution is strictly prohibited.
package internal_adapter_request_generic

import (
	"context"
	"time"

	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
)

/*
 * Metadata Management for Talking Conversations
 * ---------------------------------------------
 * These methods provide functionality to manage metadata associated with
 * a talking conversation. Metadata can be used to store additional
 * information about the conversation that may be useful for processing,
 * analysis, or integration with other systems.
 *
 * GetMetadata(): Retrieves the entire metadata map.
 * AddMetadata(): Adds a single key-value pair to the metadata.
 * SetMetadata(): Replaces the entire metadata map with a new one.
 *
 * Note: Proper use of these methods ensures consistent handling of
 * conversation metadata across the application.
 */
func (tc *GenericRequestor) GetMetadata() map[string]interface{} {
	return tc.metadata
}

func (tc *GenericRequestor) AddMetadata(
	auth types.SimplePrinciple,
	k string, v interface{}) {
	vl, ok := tc.metadata[k]
	if ok && vl == v {
		return
	}
	tc.metadata[k] = v
	utils.Go(context.Background(), func() {
		start := time.Now()
		tc.conversationService.
			ApplyConversationMetadata(
				context.Background(),
				auth, tc.assistant.Id, tc.assistantConversation.Id,
				[]*types.Metadata{types.NewMetadata(k, v)})
		tc.logger.Benchmark("GenericRequestor.AddMetadata", time.Since(start))
	})
}

func (tc *GenericRequestor) SetMetadata(
	auth types.SimplePrinciple,
	mt map[string]interface{}) {

	modified := make(map[string]interface{})
	for k, v := range mt {
		vl, ok := tc.metadata[k]
		if ok && vl == v {
			continue
		}
		tc.metadata[k] = v
		modified[k] = v
	}
	utils.Go(context.Background(), func() {
		start := time.Now()
		tc.conversationService.
			ApplyConversationMetadata(
				context.Background(),
				auth, tc.assistant.Id, tc.assistantConversation.Id, types.NewMetadataList(modified))
		tc.logger.Benchmark("GenericRequestor.SetMetadata", time.Since(start))
	})

}
