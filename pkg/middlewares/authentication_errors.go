package middlewares

const (
	authenticationFailureMessage             = "Invalid authentication credentials"
	authenticationConflictMessage            = "Authentication credential conflicts with an existing identity"
	userAuthNotSupportedMessage              = "User credential is not supported"
	userAuthIncompleteMessage                = "User credential is incomplete"
	userAuthInvalidIDMessage                 = "User credential has invalid auth id"
	userAuthRejectedMessage                  = "User credential was rejected"
	userAuthInvalidProjectIDMessage          = "User credential has invalid project id"
	userAuthProjectSelectionRejectedMessage  = "User credential project selection was rejected"
	userAuthInvalidAuditActorMessage         = "User credential has invalid audit actor"
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
