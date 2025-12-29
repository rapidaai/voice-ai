// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package commons

type Response struct {
	Code    int         `json:"code"`
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
}

type ErrorMessage struct {
	Code    int   `json:"code"`
	Message error `json:"message"`
}

type HealthCheck struct {
	Healthy bool `json:"healthy"`
}
