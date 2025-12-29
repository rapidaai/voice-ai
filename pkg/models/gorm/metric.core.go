// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package gorm_models

type Metric struct {
	Name        string `json:"name" gorm:"type:text"`
	Value       string `json:"value" gorm:"type:text"`
	Description string `json:"description" gorm:"type:text"`
}

func NewMetric(k string, v string, description string) *Metric {
	md := &Metric{
		Name:        k,
		Value:       v,
		Description: description,
	}
	return md
}
