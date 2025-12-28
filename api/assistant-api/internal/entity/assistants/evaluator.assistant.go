// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_assistant_entity

// type AssistantEvaluatorOption struct {
// 	gorm_model.Audited
// 	gorm_model.Mutable
// 	gorm_model.Metadata
// 	AssistantEvaluatorId uint64 `json:"AssistantEvaluatorId" gorm:"type:bigint;size:20"`
// }

// type AssistantEvaluator struct {
// 	gorm_model.Audited
// 	gorm_model.Mutable
// 	AssistantId uint64                      `json:"assistantId" gorm:"type:bigint;size:20"`
// 	Stage       string                      `json:"stage" gorm:"type:string;size:20"`
// 	Type        string                      `json:"type" gorm:"type:string;size:20"`
// 	Name        string                      `json:"name" gorm:"type:string;size:20"`
// 	Options     []*AssistantEvaluatorOption `json:"options"  gorm:"foreignKey:AssistantEvaluatorId"`
// }

// func (a *AssistantEvaluator) GetName() string {
// 	return a.Name
// }
// func (a *AssistantEvaluator) GetStage() string {
// 	return a.Stage
// }

// func (a *AssistantEvaluator) GetType() string {
// 	return a.Type
// }

// func (a *AssistantEvaluator) GetOptions() map[string]interface{} {
// 	opts := map[string]interface{}{}
// 	if a.Options != nil {
// 		for _, v := range a.Options {
// 			opts[v.Key] = v.Value
// 		}
// 	}
// 	return opts
// }
