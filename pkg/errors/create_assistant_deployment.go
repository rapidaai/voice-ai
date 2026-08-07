// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package errors

import "net/http"

const (
	CreateAssistantDebuggerDeploymentInvalidRequestCode         ErrorCode = 1004001
	CreateAssistantDebuggerDeploymentUnauthenticatedCode        ErrorCode = 1004002
	CreateAssistantDebuggerDeploymentMissingAuthScopeCode       ErrorCode = 1004003
	CreateAssistantDebuggerDeploymentInvalidAssistantIDCode     ErrorCode = 1004004
	CreateAssistantDebuggerDeploymentCreateDeploymentCode       ErrorCode = 1004005
	CreateAssistantDebuggerDeploymentInvalidAudioProviderCode   ErrorCode = 1004006
	CreateAssistantDebuggerDeploymentInvalidIdealTimeoutCode    ErrorCode = 1004007
	CreateAssistantDebuggerDeploymentInvalidTimeoutBackoffCode  ErrorCode = 1004008
	CreateAssistantDebuggerDeploymentInvalidSessionDurationCode ErrorCode = 1004009
	CreateAssistantDebuggerDeploymentInvalidUnclearTimeoutCode  ErrorCode = 1004010

	CreateAssistantPhoneDeploymentInvalidRequestCode         ErrorCode = 1005001
	CreateAssistantPhoneDeploymentUnauthenticatedCode        ErrorCode = 1005002
	CreateAssistantPhoneDeploymentMissingAuthScopeCode       ErrorCode = 1005003
	CreateAssistantPhoneDeploymentInvalidAssistantIDCode     ErrorCode = 1005004
	CreateAssistantPhoneDeploymentCreateDeploymentCode       ErrorCode = 1005005
	CreateAssistantPhoneDeploymentInvalidAudioProviderCode   ErrorCode = 1005006
	CreateAssistantPhoneDeploymentInvalidIdealTimeoutCode    ErrorCode = 1005007
	CreateAssistantPhoneDeploymentInvalidTimeoutBackoffCode  ErrorCode = 1005008
	CreateAssistantPhoneDeploymentInvalidSessionDurationCode ErrorCode = 1005009
	CreateAssistantPhoneDeploymentMissingPhoneProviderCode   ErrorCode = 1005010
	CreateAssistantPhoneDeploymentInvalidUnclearTimeoutCode  ErrorCode = 1005011

	CreateAssistantApiDeploymentInvalidRequestCode         ErrorCode = 1006001
	CreateAssistantApiDeploymentUnauthenticatedCode        ErrorCode = 1006002
	CreateAssistantApiDeploymentMissingAuthScopeCode       ErrorCode = 1006003
	CreateAssistantApiDeploymentInvalidAssistantIDCode     ErrorCode = 1006004
	CreateAssistantApiDeploymentCreateDeploymentCode       ErrorCode = 1006005
	CreateAssistantApiDeploymentInvalidAudioProviderCode   ErrorCode = 1006006
	CreateAssistantApiDeploymentInvalidIdealTimeoutCode    ErrorCode = 1006007
	CreateAssistantApiDeploymentInvalidTimeoutBackoffCode  ErrorCode = 1006008
	CreateAssistantApiDeploymentInvalidSessionDurationCode ErrorCode = 1006009
	CreateAssistantApiDeploymentInvalidUnclearTimeoutCode  ErrorCode = 1006010

	CreateAssistantWebpluginDeploymentInvalidRequestCode         ErrorCode = 1007001
	CreateAssistantWebpluginDeploymentUnauthenticatedCode        ErrorCode = 1007002
	CreateAssistantWebpluginDeploymentMissingAuthScopeCode       ErrorCode = 1007003
	CreateAssistantWebpluginDeploymentInvalidAssistantIDCode     ErrorCode = 1007004
	CreateAssistantWebpluginDeploymentCreateDeploymentCode       ErrorCode = 1007005
	CreateAssistantWebpluginDeploymentInvalidAudioProviderCode   ErrorCode = 1007006
	CreateAssistantWebpluginDeploymentInvalidIdealTimeoutCode    ErrorCode = 1007007
	CreateAssistantWebpluginDeploymentInvalidTimeoutBackoffCode  ErrorCode = 1007008
	CreateAssistantWebpluginDeploymentInvalidSessionDurationCode ErrorCode = 1007009
	CreateAssistantWebpluginDeploymentInvalidUnclearTimeoutCode  ErrorCode = 1007010

	CreateAssistantWhatsappDeploymentInvalidRequestCode         ErrorCode = 1008001
	CreateAssistantWhatsappDeploymentUnauthenticatedCode        ErrorCode = 1008002
	CreateAssistantWhatsappDeploymentMissingAuthScopeCode       ErrorCode = 1008003
	CreateAssistantWhatsappDeploymentInvalidAssistantIDCode     ErrorCode = 1008004
	CreateAssistantWhatsappDeploymentCreateDeploymentCode       ErrorCode = 1008005
	CreateAssistantWhatsappDeploymentInvalidIdealTimeoutCode    ErrorCode = 1008006
	CreateAssistantWhatsappDeploymentInvalidTimeoutBackoffCode  ErrorCode = 1008007
	CreateAssistantWhatsappDeploymentInvalidSessionDurationCode ErrorCode = 1008008
	CreateAssistantWhatsappDeploymentMissingProviderCode        ErrorCode = 1008009
	CreateAssistantWhatsappDeploymentInvalidUnclearTimeoutCode  ErrorCode = 1008010

	GetAssistantDebuggerDeploymentUnauthenticatedCode    ErrorCode = 1009001
	GetAssistantDebuggerDeploymentMissingAuthScopeCode   ErrorCode = 1009002
	GetAssistantDebuggerDeploymentInvalidAssistantIDCode ErrorCode = 1009003
	GetAssistantDebuggerDeploymentGetDeploymentCode      ErrorCode = 1009004

	GetAssistantPhoneDeploymentUnauthenticatedCode    ErrorCode = 1010001
	GetAssistantPhoneDeploymentMissingAuthScopeCode   ErrorCode = 1010002
	GetAssistantPhoneDeploymentInvalidAssistantIDCode ErrorCode = 1010003
	GetAssistantPhoneDeploymentGetDeploymentCode      ErrorCode = 1010004

	GetAssistantApiDeploymentUnauthenticatedCode    ErrorCode = 1011001
	GetAssistantApiDeploymentMissingAuthScopeCode   ErrorCode = 1011002
	GetAssistantApiDeploymentInvalidAssistantIDCode ErrorCode = 1011003
	GetAssistantApiDeploymentGetDeploymentCode      ErrorCode = 1011004

	GetAssistantWebpluginDeploymentUnauthenticatedCode    ErrorCode = 1012001
	GetAssistantWebpluginDeploymentMissingAuthScopeCode   ErrorCode = 1012002
	GetAssistantWebpluginDeploymentInvalidAssistantIDCode ErrorCode = 1012003
	GetAssistantWebpluginDeploymentGetDeploymentCode      ErrorCode = 1012004

	GetAssistantWhatsappDeploymentUnauthenticatedCode    ErrorCode = 1013001
	GetAssistantWhatsappDeploymentMissingAuthScopeCode   ErrorCode = 1013002
	GetAssistantWhatsappDeploymentInvalidAssistantIDCode ErrorCode = 1013003
	GetAssistantWhatsappDeploymentGetDeploymentCode      ErrorCode = 1013004

	GetAllAssistantDebuggerDeploymentUnauthenticatedCode    ErrorCode = 1014001
	GetAllAssistantDebuggerDeploymentMissingAuthScopeCode   ErrorCode = 1014002
	GetAllAssistantDebuggerDeploymentInvalidAssistantIDCode ErrorCode = 1014003
	GetAllAssistantDebuggerDeploymentInvalidRequestCode     ErrorCode = 1014004
	GetAllAssistantDebuggerDeploymentGetDeploymentCode      ErrorCode = 1014005

	GetAllAssistantPhoneDeploymentUnauthenticatedCode    ErrorCode = 1015001
	GetAllAssistantPhoneDeploymentMissingAuthScopeCode   ErrorCode = 1015002
	GetAllAssistantPhoneDeploymentInvalidAssistantIDCode ErrorCode = 1015003
	GetAllAssistantPhoneDeploymentInvalidRequestCode     ErrorCode = 1015004
	GetAllAssistantPhoneDeploymentGetDeploymentCode      ErrorCode = 1015005

	GetAllAssistantApiDeploymentUnauthenticatedCode    ErrorCode = 1016001
	GetAllAssistantApiDeploymentMissingAuthScopeCode   ErrorCode = 1016002
	GetAllAssistantApiDeploymentInvalidAssistantIDCode ErrorCode = 1016003
	GetAllAssistantApiDeploymentInvalidRequestCode     ErrorCode = 1016004
	GetAllAssistantApiDeploymentGetDeploymentCode      ErrorCode = 1016005

	GetAllAssistantWebpluginDeploymentUnauthenticatedCode    ErrorCode = 1017001
	GetAllAssistantWebpluginDeploymentMissingAuthScopeCode   ErrorCode = 1017002
	GetAllAssistantWebpluginDeploymentInvalidAssistantIDCode ErrorCode = 1017003
	GetAllAssistantWebpluginDeploymentInvalidRequestCode     ErrorCode = 1017004
	GetAllAssistantWebpluginDeploymentGetDeploymentCode      ErrorCode = 1017005

	GetAllAssistantWhatsappDeploymentUnauthenticatedCode    ErrorCode = 1018001
	GetAllAssistantWhatsappDeploymentMissingAuthScopeCode   ErrorCode = 1018002
	GetAllAssistantWhatsappDeploymentInvalidAssistantIDCode ErrorCode = 1018003
	GetAllAssistantWhatsappDeploymentInvalidRequestCode     ErrorCode = 1018004
	GetAllAssistantWhatsappDeploymentGetDeploymentCode      ErrorCode = 1018005
)

var (
	CreateAssistantDebuggerDeploymentInvalidRequest = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateAssistantDebuggerDeploymentInvalidRequestCode,
		Error:          "invalid request",
		ErrorMessage:   "Invalid request.",
	}
	CreateAssistantDebuggerDeploymentUnauthenticated = PlatformError{
		HTTPStatusCode: http.StatusUnauthorized,
		Code:           CreateAssistantDebuggerDeploymentUnauthenticatedCode,
		Error:          "unauthenticated request",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	CreateAssistantDebuggerDeploymentMissingAuthScope = PlatformError{
		HTTPStatusCode: http.StatusForbidden,
		Code:           CreateAssistantDebuggerDeploymentMissingAuthScopeCode,
		Error:          "missing authentication scope",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	CreateAssistantDebuggerDeploymentInvalidAssistantID = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateAssistantDebuggerDeploymentInvalidAssistantIDCode,
		Error:          "invalid assistant_id parameter",
		ErrorMessage:   "Please provide a valid assistantId parameter.",
	}
	CreateAssistantDebuggerDeploymentCreateDeployment = PlatformError{
		HTTPStatusCode: http.StatusInternalServerError,
		Code:           CreateAssistantDebuggerDeploymentCreateDeploymentCode,
		Error:          "unable to create assistant debugger deployment",
		ErrorMessage:   "Unable to create assistant debugger deployment, please try again later.",
	}
	CreateAssistantDebuggerDeploymentInvalidAudioProvider = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateAssistantDebuggerDeploymentInvalidAudioProviderCode,
		Error:          "invalid audio provider parameter",
		ErrorMessage:   "Please provide a valid audioProvider parameter.",
	}
	CreateAssistantDebuggerDeploymentInvalidIdealTimeout = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateAssistantDebuggerDeploymentInvalidIdealTimeoutCode,
		Error:          "invalid ideal_timeout parameter",
		ErrorMessage:   "Please provide idealTimeout between 5 and 120 seconds.",
	}
	CreateAssistantDebuggerDeploymentInvalidUnclearTimeout = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateAssistantDebuggerDeploymentInvalidUnclearTimeoutCode,
		Error:          "invalid unclear_input_timeout parameter",
		ErrorMessage:   "Please provide unclearInputTimeout between 2 and 10 seconds.",
	}
	CreateAssistantDebuggerDeploymentInvalidTimeoutBackoff = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateAssistantDebuggerDeploymentInvalidTimeoutBackoffCode,
		Error:          "invalid ideal_timeout_backoff parameter",
		ErrorMessage:   "Please provide idealTimeoutBackoff between 0 and 5 times.",
	}
	CreateAssistantDebuggerDeploymentInvalidSessionDuration = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateAssistantDebuggerDeploymentInvalidSessionDurationCode,
		Error:          "invalid max_session_duration parameter",
		ErrorMessage:   "Please provide maxSessionDuration between 180 and 600 seconds.",
	}
	CreateAssistantPhoneDeploymentInvalidRequest = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateAssistantPhoneDeploymentInvalidRequestCode,
		Error:          "invalid request",
		ErrorMessage:   "Invalid request.",
	}
	CreateAssistantPhoneDeploymentUnauthenticated = PlatformError{
		HTTPStatusCode: http.StatusUnauthorized,
		Code:           CreateAssistantPhoneDeploymentUnauthenticatedCode,
		Error:          "unauthenticated request",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	CreateAssistantPhoneDeploymentMissingAuthScope = PlatformError{
		HTTPStatusCode: http.StatusForbidden,
		Code:           CreateAssistantPhoneDeploymentMissingAuthScopeCode,
		Error:          "missing authentication scope",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	CreateAssistantPhoneDeploymentInvalidAssistantID = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateAssistantPhoneDeploymentInvalidAssistantIDCode,
		Error:          "invalid assistant_id parameter",
		ErrorMessage:   "Please provide a valid assistantId parameter.",
	}
	CreateAssistantPhoneDeploymentCreateDeployment = PlatformError{
		HTTPStatusCode: http.StatusInternalServerError,
		Code:           CreateAssistantPhoneDeploymentCreateDeploymentCode,
		Error:          "unable to create assistant phone deployment",
		ErrorMessage:   "Unable to create assistant phone deployment, please try again later.",
	}
	CreateAssistantPhoneDeploymentInvalidAudioProvider = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateAssistantPhoneDeploymentInvalidAudioProviderCode,
		Error:          "invalid audio provider parameter",
		ErrorMessage:   "Please provide a valid audioProvider parameter.",
	}
	CreateAssistantPhoneDeploymentInvalidIdealTimeout = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateAssistantPhoneDeploymentInvalidIdealTimeoutCode,
		Error:          "invalid ideal_timeout parameter",
		ErrorMessage:   "Please provide idealTimeout between 5 and 120 seconds.",
	}
	CreateAssistantPhoneDeploymentInvalidUnclearTimeout = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateAssistantPhoneDeploymentInvalidUnclearTimeoutCode,
		Error:          "invalid unclear_input_timeout parameter",
		ErrorMessage:   "Please provide unclearInputTimeout between 2 and 10 seconds.",
	}
	CreateAssistantPhoneDeploymentInvalidTimeoutBackoff = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateAssistantPhoneDeploymentInvalidTimeoutBackoffCode,
		Error:          "invalid ideal_timeout_backoff parameter",
		ErrorMessage:   "Please provide idealTimeoutBackoff between 0 and 5 times.",
	}
	CreateAssistantPhoneDeploymentInvalidSessionDuration = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateAssistantPhoneDeploymentInvalidSessionDurationCode,
		Error:          "invalid max_session_duration parameter",
		ErrorMessage:   "Please provide maxSessionDuration between 180 and 600 seconds.",
	}
	CreateAssistantPhoneDeploymentMissingPhoneProvider = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateAssistantPhoneDeploymentMissingPhoneProviderCode,
		Error:          "missing phone_provider_name parameter",
		ErrorMessage:   "Please provide the required phoneProviderName parameter.",
	}
	CreateAssistantApiDeploymentInvalidRequest = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateAssistantApiDeploymentInvalidRequestCode,
		Error:          "invalid request",
		ErrorMessage:   "Invalid request.",
	}
	CreateAssistantApiDeploymentUnauthenticated = PlatformError{
		HTTPStatusCode: http.StatusUnauthorized,
		Code:           CreateAssistantApiDeploymentUnauthenticatedCode,
		Error:          "unauthenticated request",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	CreateAssistantApiDeploymentMissingAuthScope = PlatformError{
		HTTPStatusCode: http.StatusForbidden,
		Code:           CreateAssistantApiDeploymentMissingAuthScopeCode,
		Error:          "missing authentication scope",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	CreateAssistantApiDeploymentInvalidAssistantID = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateAssistantApiDeploymentInvalidAssistantIDCode,
		Error:          "invalid assistant_id parameter",
		ErrorMessage:   "Please provide a valid assistantId parameter.",
	}
	CreateAssistantApiDeploymentCreateDeployment = PlatformError{
		HTTPStatusCode: http.StatusInternalServerError,
		Code:           CreateAssistantApiDeploymentCreateDeploymentCode,
		Error:          "unable to create assistant api deployment",
		ErrorMessage:   "Unable to create assistant api deployment, please try again later.",
	}
	CreateAssistantApiDeploymentInvalidAudioProvider = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateAssistantApiDeploymentInvalidAudioProviderCode,
		Error:          "invalid audio provider parameter",
		ErrorMessage:   "Please provide a valid audioProvider parameter.",
	}
	CreateAssistantApiDeploymentInvalidIdealTimeout = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateAssistantApiDeploymentInvalidIdealTimeoutCode,
		Error:          "invalid ideal_timeout parameter",
		ErrorMessage:   "Please provide idealTimeout between 5 and 120 seconds.",
	}
	CreateAssistantApiDeploymentInvalidUnclearTimeout = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateAssistantApiDeploymentInvalidUnclearTimeoutCode,
		Error:          "invalid unclear_input_timeout parameter",
		ErrorMessage:   "Please provide unclearInputTimeout between 2 and 10 seconds.",
	}
	CreateAssistantApiDeploymentInvalidTimeoutBackoff = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateAssistantApiDeploymentInvalidTimeoutBackoffCode,
		Error:          "invalid ideal_timeout_backoff parameter",
		ErrorMessage:   "Please provide idealTimeoutBackoff between 0 and 5 times.",
	}
	CreateAssistantApiDeploymentInvalidSessionDuration = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateAssistantApiDeploymentInvalidSessionDurationCode,
		Error:          "invalid max_session_duration parameter",
		ErrorMessage:   "Please provide maxSessionDuration between 180 and 600 seconds.",
	}
	CreateAssistantWebpluginDeploymentInvalidRequest = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateAssistantWebpluginDeploymentInvalidRequestCode,
		Error:          "invalid request",
		ErrorMessage:   "Invalid request.",
	}
	CreateAssistantWebpluginDeploymentUnauthenticated = PlatformError{
		HTTPStatusCode: http.StatusUnauthorized,
		Code:           CreateAssistantWebpluginDeploymentUnauthenticatedCode,
		Error:          "unauthenticated request",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	CreateAssistantWebpluginDeploymentMissingAuthScope = PlatformError{
		HTTPStatusCode: http.StatusForbidden,
		Code:           CreateAssistantWebpluginDeploymentMissingAuthScopeCode,
		Error:          "missing authentication scope",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	CreateAssistantWebpluginDeploymentInvalidAssistantID = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateAssistantWebpluginDeploymentInvalidAssistantIDCode,
		Error:          "invalid assistant_id parameter",
		ErrorMessage:   "Please provide a valid assistantId parameter.",
	}
	CreateAssistantWebpluginDeploymentCreateDeployment = PlatformError{
		HTTPStatusCode: http.StatusInternalServerError,
		Code:           CreateAssistantWebpluginDeploymentCreateDeploymentCode,
		Error:          "unable to create assistant webplugin deployment",
		ErrorMessage:   "Unable to create assistant webplugin deployment, please try again later.",
	}
	CreateAssistantWebpluginDeploymentInvalidAudioProvider = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateAssistantWebpluginDeploymentInvalidAudioProviderCode,
		Error:          "invalid audio provider parameter",
		ErrorMessage:   "Please provide a valid audioProvider parameter.",
	}
	CreateAssistantWebpluginDeploymentInvalidIdealTimeout = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateAssistantWebpluginDeploymentInvalidIdealTimeoutCode,
		Error:          "invalid ideal_timeout parameter",
		ErrorMessage:   "Please provide idealTimeout between 5 and 120 seconds.",
	}
	CreateAssistantWebpluginDeploymentInvalidUnclearTimeout = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateAssistantWebpluginDeploymentInvalidUnclearTimeoutCode,
		Error:          "invalid unclear_input_timeout parameter",
		ErrorMessage:   "Please provide unclearInputTimeout between 2 and 10 seconds.",
	}
	CreateAssistantWebpluginDeploymentInvalidTimeoutBackoff = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateAssistantWebpluginDeploymentInvalidTimeoutBackoffCode,
		Error:          "invalid ideal_timeout_backoff parameter",
		ErrorMessage:   "Please provide idealTimeoutBackoff between 0 and 5 times.",
	}
	CreateAssistantWebpluginDeploymentInvalidSessionDuration = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateAssistantWebpluginDeploymentInvalidSessionDurationCode,
		Error:          "invalid max_session_duration parameter",
		ErrorMessage:   "Please provide maxSessionDuration between 180 and 600 seconds.",
	}
	CreateAssistantWhatsappDeploymentInvalidRequest = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateAssistantWhatsappDeploymentInvalidRequestCode,
		Error:          "invalid request",
		ErrorMessage:   "Invalid request.",
	}
	CreateAssistantWhatsappDeploymentUnauthenticated = PlatformError{
		HTTPStatusCode: http.StatusUnauthorized,
		Code:           CreateAssistantWhatsappDeploymentUnauthenticatedCode,
		Error:          "unauthenticated request",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	CreateAssistantWhatsappDeploymentMissingAuthScope = PlatformError{
		HTTPStatusCode: http.StatusForbidden,
		Code:           CreateAssistantWhatsappDeploymentMissingAuthScopeCode,
		Error:          "missing authentication scope",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	CreateAssistantWhatsappDeploymentInvalidAssistantID = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateAssistantWhatsappDeploymentInvalidAssistantIDCode,
		Error:          "invalid assistant_id parameter",
		ErrorMessage:   "Please provide a valid assistantId parameter.",
	}
	CreateAssistantWhatsappDeploymentCreateDeployment = PlatformError{
		HTTPStatusCode: http.StatusInternalServerError,
		Code:           CreateAssistantWhatsappDeploymentCreateDeploymentCode,
		Error:          "unable to create assistant whatsapp deployment",
		ErrorMessage:   "Unable to create assistant whatsapp deployment, please try again later.",
	}
	CreateAssistantWhatsappDeploymentInvalidIdealTimeout = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateAssistantWhatsappDeploymentInvalidIdealTimeoutCode,
		Error:          "invalid ideal_timeout parameter",
		ErrorMessage:   "Please provide idealTimeout between 5 and 120 seconds.",
	}
	CreateAssistantWhatsappDeploymentInvalidUnclearTimeout = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateAssistantWhatsappDeploymentInvalidUnclearTimeoutCode,
		Error:          "invalid unclear_input_timeout parameter",
		ErrorMessage:   "Please provide unclearInputTimeout between 2 and 10 seconds.",
	}
	CreateAssistantWhatsappDeploymentInvalidTimeoutBackoff = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateAssistantWhatsappDeploymentInvalidTimeoutBackoffCode,
		Error:          "invalid ideal_timeout_backoff parameter",
		ErrorMessage:   "Please provide idealTimeoutBackoff between 0 and 5 times.",
	}
	CreateAssistantWhatsappDeploymentInvalidSessionDuration = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateAssistantWhatsappDeploymentInvalidSessionDurationCode,
		Error:          "invalid max_session_duration parameter",
		ErrorMessage:   "Please provide maxSessionDuration between 180 and 600 seconds.",
	}
	CreateAssistantWhatsappDeploymentMissingProvider = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateAssistantWhatsappDeploymentMissingProviderCode,
		Error:          "missing whatsapp_provider_name parameter",
		ErrorMessage:   "Please provide the required whatsappProviderName parameter.",
	}
	GetAssistantDebuggerDeploymentUnauthenticated = PlatformError{
		HTTPStatusCode: http.StatusUnauthorized,
		Code:           GetAssistantDebuggerDeploymentUnauthenticatedCode,
		Error:          "unauthenticated request",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	GetAssistantDebuggerDeploymentMissingAuthScope = PlatformError{
		HTTPStatusCode: http.StatusForbidden,
		Code:           GetAssistantDebuggerDeploymentMissingAuthScopeCode,
		Error:          "missing authentication scope",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	GetAssistantDebuggerDeploymentInvalidAssistantID = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           GetAssistantDebuggerDeploymentInvalidAssistantIDCode,
		Error:          "invalid assistant_id parameter",
		ErrorMessage:   "Please provide a valid assistantId parameter.",
	}
	GetAssistantDebuggerDeploymentGetDeployment = PlatformError{
		HTTPStatusCode: http.StatusInternalServerError,
		Code:           GetAssistantDebuggerDeploymentGetDeploymentCode,
		Error:          "unable to get assistant debugger deployment",
		ErrorMessage:   "Unable to get assistant debugger deployment, please try again later.",
	}
	GetAssistantPhoneDeploymentUnauthenticated = PlatformError{
		HTTPStatusCode: http.StatusUnauthorized,
		Code:           GetAssistantPhoneDeploymentUnauthenticatedCode,
		Error:          "unauthenticated request",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	GetAssistantPhoneDeploymentMissingAuthScope = PlatformError{
		HTTPStatusCode: http.StatusForbidden,
		Code:           GetAssistantPhoneDeploymentMissingAuthScopeCode,
		Error:          "missing authentication scope",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	GetAssistantPhoneDeploymentInvalidAssistantID = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           GetAssistantPhoneDeploymentInvalidAssistantIDCode,
		Error:          "invalid assistant_id parameter",
		ErrorMessage:   "Please provide a valid assistantId parameter.",
	}
	GetAssistantPhoneDeploymentGetDeployment = PlatformError{
		HTTPStatusCode: http.StatusInternalServerError,
		Code:           GetAssistantPhoneDeploymentGetDeploymentCode,
		Error:          "unable to get assistant phone deployment",
		ErrorMessage:   "Unable to get assistant phone deployment, please try again later.",
	}
	GetAssistantApiDeploymentUnauthenticated = PlatformError{
		HTTPStatusCode: http.StatusUnauthorized,
		Code:           GetAssistantApiDeploymentUnauthenticatedCode,
		Error:          "unauthenticated request",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	GetAssistantApiDeploymentMissingAuthScope = PlatformError{
		HTTPStatusCode: http.StatusForbidden,
		Code:           GetAssistantApiDeploymentMissingAuthScopeCode,
		Error:          "missing authentication scope",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	GetAssistantApiDeploymentInvalidAssistantID = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           GetAssistantApiDeploymentInvalidAssistantIDCode,
		Error:          "invalid assistant_id parameter",
		ErrorMessage:   "Please provide a valid assistantId parameter.",
	}
	GetAssistantApiDeploymentGetDeployment = PlatformError{
		HTTPStatusCode: http.StatusInternalServerError,
		Code:           GetAssistantApiDeploymentGetDeploymentCode,
		Error:          "unable to get assistant api deployment",
		ErrorMessage:   "Unable to get assistant api deployment, please try again later.",
	}
	GetAssistantWebpluginDeploymentUnauthenticated = PlatformError{
		HTTPStatusCode: http.StatusUnauthorized,
		Code:           GetAssistantWebpluginDeploymentUnauthenticatedCode,
		Error:          "unauthenticated request",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	GetAssistantWebpluginDeploymentMissingAuthScope = PlatformError{
		HTTPStatusCode: http.StatusForbidden,
		Code:           GetAssistantWebpluginDeploymentMissingAuthScopeCode,
		Error:          "missing authentication scope",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	GetAssistantWebpluginDeploymentInvalidAssistantID = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           GetAssistantWebpluginDeploymentInvalidAssistantIDCode,
		Error:          "invalid assistant_id parameter",
		ErrorMessage:   "Please provide a valid assistantId parameter.",
	}
	GetAssistantWebpluginDeploymentGetDeployment = PlatformError{
		HTTPStatusCode: http.StatusInternalServerError,
		Code:           GetAssistantWebpluginDeploymentGetDeploymentCode,
		Error:          "unable to get assistant webplugin deployment",
		ErrorMessage:   "Unable to get assistant webplugin deployment, please try again later.",
	}
	GetAssistantWhatsappDeploymentUnauthenticated = PlatformError{
		HTTPStatusCode: http.StatusUnauthorized,
		Code:           GetAssistantWhatsappDeploymentUnauthenticatedCode,
		Error:          "unauthenticated request",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	GetAssistantWhatsappDeploymentMissingAuthScope = PlatformError{
		HTTPStatusCode: http.StatusForbidden,
		Code:           GetAssistantWhatsappDeploymentMissingAuthScopeCode,
		Error:          "missing authentication scope",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	GetAssistantWhatsappDeploymentInvalidAssistantID = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           GetAssistantWhatsappDeploymentInvalidAssistantIDCode,
		Error:          "invalid assistant_id parameter",
		ErrorMessage:   "Please provide a valid assistantId parameter.",
	}
	GetAssistantWhatsappDeploymentGetDeployment = PlatformError{
		HTTPStatusCode: http.StatusInternalServerError,
		Code:           GetAssistantWhatsappDeploymentGetDeploymentCode,
		Error:          "unable to get assistant whatsapp deployment",
		ErrorMessage:   "Unable to get assistant whatsapp deployment, please try again later.",
	}
	GetAllAssistantDebuggerDeploymentUnauthenticated = PlatformError{
		HTTPStatusCode: http.StatusUnauthorized,
		Code:           GetAllAssistantDebuggerDeploymentUnauthenticatedCode,
		Error:          "unauthenticated request",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	GetAllAssistantDebuggerDeploymentMissingAuthScope = PlatformError{
		HTTPStatusCode: http.StatusForbidden,
		Code:           GetAllAssistantDebuggerDeploymentMissingAuthScopeCode,
		Error:          "missing authentication scope",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	GetAllAssistantDebuggerDeploymentInvalidAssistantID = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           GetAllAssistantDebuggerDeploymentInvalidAssistantIDCode,
		Error:          "invalid assistant_id parameter",
		ErrorMessage:   "Please provide a valid assistantId parameter.",
	}
	GetAllAssistantDebuggerDeploymentInvalidRequest = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           GetAllAssistantDebuggerDeploymentInvalidRequestCode,
		Error:          "invalid request",
		ErrorMessage:   "Invalid request.",
	}
	GetAllAssistantDebuggerDeploymentGetDeployment = PlatformError{
		HTTPStatusCode: http.StatusInternalServerError,
		Code:           GetAllAssistantDebuggerDeploymentGetDeploymentCode,
		Error:          "unable to get assistant debugger deployments",
		ErrorMessage:   "Unable to get assistant debugger deployments, please try again later.",
	}
	GetAllAssistantPhoneDeploymentUnauthenticated = PlatformError{
		HTTPStatusCode: http.StatusUnauthorized,
		Code:           GetAllAssistantPhoneDeploymentUnauthenticatedCode,
		Error:          "unauthenticated request",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	GetAllAssistantPhoneDeploymentMissingAuthScope = PlatformError{
		HTTPStatusCode: http.StatusForbidden,
		Code:           GetAllAssistantPhoneDeploymentMissingAuthScopeCode,
		Error:          "missing authentication scope",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	GetAllAssistantPhoneDeploymentInvalidAssistantID = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           GetAllAssistantPhoneDeploymentInvalidAssistantIDCode,
		Error:          "invalid assistant_id parameter",
		ErrorMessage:   "Please provide a valid assistantId parameter.",
	}
	GetAllAssistantPhoneDeploymentInvalidRequest = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           GetAllAssistantPhoneDeploymentInvalidRequestCode,
		Error:          "invalid request",
		ErrorMessage:   "Invalid request.",
	}
	GetAllAssistantPhoneDeploymentGetDeployment = PlatformError{
		HTTPStatusCode: http.StatusInternalServerError,
		Code:           GetAllAssistantPhoneDeploymentGetDeploymentCode,
		Error:          "unable to get assistant phone deployments",
		ErrorMessage:   "Unable to get assistant phone deployments, please try again later.",
	}
	GetAllAssistantApiDeploymentUnauthenticated = PlatformError{
		HTTPStatusCode: http.StatusUnauthorized,
		Code:           GetAllAssistantApiDeploymentUnauthenticatedCode,
		Error:          "unauthenticated request",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	GetAllAssistantApiDeploymentMissingAuthScope = PlatformError{
		HTTPStatusCode: http.StatusForbidden,
		Code:           GetAllAssistantApiDeploymentMissingAuthScopeCode,
		Error:          "missing authentication scope",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	GetAllAssistantApiDeploymentInvalidAssistantID = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           GetAllAssistantApiDeploymentInvalidAssistantIDCode,
		Error:          "invalid assistant_id parameter",
		ErrorMessage:   "Please provide a valid assistantId parameter.",
	}
	GetAllAssistantApiDeploymentInvalidRequest = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           GetAllAssistantApiDeploymentInvalidRequestCode,
		Error:          "invalid request",
		ErrorMessage:   "Invalid request.",
	}
	GetAllAssistantApiDeploymentGetDeployment = PlatformError{
		HTTPStatusCode: http.StatusInternalServerError,
		Code:           GetAllAssistantApiDeploymentGetDeploymentCode,
		Error:          "unable to get assistant api deployments",
		ErrorMessage:   "Unable to get assistant api deployments, please try again later.",
	}
	GetAllAssistantWebpluginDeploymentUnauthenticated = PlatformError{
		HTTPStatusCode: http.StatusUnauthorized,
		Code:           GetAllAssistantWebpluginDeploymentUnauthenticatedCode,
		Error:          "unauthenticated request",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	GetAllAssistantWebpluginDeploymentMissingAuthScope = PlatformError{
		HTTPStatusCode: http.StatusForbidden,
		Code:           GetAllAssistantWebpluginDeploymentMissingAuthScopeCode,
		Error:          "missing authentication scope",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	GetAllAssistantWebpluginDeploymentInvalidAssistantID = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           GetAllAssistantWebpluginDeploymentInvalidAssistantIDCode,
		Error:          "invalid assistant_id parameter",
		ErrorMessage:   "Please provide a valid assistantId parameter.",
	}
	GetAllAssistantWebpluginDeploymentInvalidRequest = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           GetAllAssistantWebpluginDeploymentInvalidRequestCode,
		Error:          "invalid request",
		ErrorMessage:   "Invalid request.",
	}
	GetAllAssistantWebpluginDeploymentGetDeployment = PlatformError{
		HTTPStatusCode: http.StatusInternalServerError,
		Code:           GetAllAssistantWebpluginDeploymentGetDeploymentCode,
		Error:          "unable to get assistant webplugin deployments",
		ErrorMessage:   "Unable to get assistant webplugin deployments, please try again later.",
	}
	GetAllAssistantWhatsappDeploymentUnauthenticated = PlatformError{
		HTTPStatusCode: http.StatusUnauthorized,
		Code:           GetAllAssistantWhatsappDeploymentUnauthenticatedCode,
		Error:          "unauthenticated request",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	GetAllAssistantWhatsappDeploymentMissingAuthScope = PlatformError{
		HTTPStatusCode: http.StatusForbidden,
		Code:           GetAllAssistantWhatsappDeploymentMissingAuthScopeCode,
		Error:          "missing authentication scope",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	GetAllAssistantWhatsappDeploymentInvalidAssistantID = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           GetAllAssistantWhatsappDeploymentInvalidAssistantIDCode,
		Error:          "invalid assistant_id parameter",
		ErrorMessage:   "Please provide a valid assistantId parameter.",
	}
	GetAllAssistantWhatsappDeploymentInvalidRequest = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           GetAllAssistantWhatsappDeploymentInvalidRequestCode,
		Error:          "invalid request",
		ErrorMessage:   "Invalid request.",
	}
	GetAllAssistantWhatsappDeploymentGetDeployment = PlatformError{
		HTTPStatusCode: http.StatusInternalServerError,
		Code:           GetAllAssistantWhatsappDeploymentGetDeploymentCode,
		Error:          "unable to get assistant whatsapp deployments",
		ErrorMessage:   "Unable to get assistant whatsapp deployments, please try again later.",
	}
)
