package middlewares

import "errors"

type AuthenticationError struct {
	Error string `json:"error"`
}

var (
	errUserAuthNotSupported             = errors.New("User credential is not supported")
	errUserAuthIncomplete               = errors.New("User credential is incomplete")
	errUserAuthInvalidID                = errors.New("User credential has invalid auth id")
	errUserAuthRejected                 = errors.New("User credential was rejected")
	errUserAuthInvalidProjectID         = errors.New("User credential has invalid project id")
	errUserAuthProjectSelectionRejected = errors.New("User credential project selection was rejected")
	errUserAuthInvalidAuditActor        = errors.New("User credential has invalid audit actor")
)

const (
	AuthenticationFailureMessage             = "Invalid authentication credentials"
	authenticationConflictMessage            = "Authentication credential conflicts with an existing identity"
	projectAuthNotSupportedMessage           = "Project credential is not supported"
	projectAuthEmptyMessage                  = "Project credential is empty"
	projectAuthRejectedMessage               = "Project credential was rejected"
	projectAuthInvalidAuditActorMessage      = "Project credential has invalid audit actor"
	organizationAuthNotSupportedMessage      = "Organization credential is not supported"
	organizationAuthRejectedMessage          = "Organization credential was rejected"
	organizationAuthInvalidAuditActorMessage = "Organization credential has invalid audit actor"
	serviceAuthNotSupportedMessage           = "Service credential is not supported"
	serviceAuthRejectedMessage               = "Service credential was rejected"
	serviceAuthInvalidAuditActorMessage      = "Service credential has invalid audit actor"
)
