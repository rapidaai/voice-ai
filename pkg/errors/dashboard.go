// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package errors

// DashboardAnalyticsRequestInvalidError preserves a trusted downstream validation message.
type DashboardAnalyticsRequestInvalidError struct {
	Message string
	Cause   error
}

func (dashboardError DashboardAnalyticsRequestInvalidError) Error() string {
	return dashboardError.Message
}

func (dashboardError DashboardAnalyticsRequestInvalidError) Unwrap() error {
	return dashboardError.Cause
}

// DashboardUnavailableError reports that neither Dashboard section could be loaded.
type DashboardUnavailableError struct {
	Cause error
}

func (DashboardUnavailableError) Error() string {
	return "dashboard is unavailable"
}

func (dashboardError DashboardUnavailableError) Unwrap() error {
	return dashboardError.Cause
}
