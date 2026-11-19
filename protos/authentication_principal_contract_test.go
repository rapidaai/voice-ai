// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package protos

import (
	"testing"

	authentication_proto "github.com/rapidaai/protos/authentication"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestAuthenticationProtoContract(t *testing.T) {
	type fieldContract struct {
		number   protoreflect.FieldNumber
		name     protoreflect.Name
		kind     protoreflect.Kind
		message  protoreflect.FullName
		list     bool
		optional bool
	}
	contracts := map[protoreflect.Name][]fieldContract{
		"Identity": {
			{number: 1, name: "type", kind: protoreflect.StringKind},
			{number: 2, name: "id", kind: protoreflect.Uint64Kind},
			{number: 3, name: "issuer", kind: protoreflect.StringKind},
		},
		"ServiceActor": {
			{number: 1, name: "id", kind: protoreflect.Uint64Kind},
			{number: 2, name: "issuer", kind: protoreflect.StringKind},
		},
		"Tenant": {
			{number: 1, name: "organizationId", kind: protoreflect.Uint64Kind},
			{number: 2, name: "projectId", kind: protoreflect.Uint64Kind},
		},
		"ProofReference": {
			{number: 1, name: "type", kind: protoreflect.StringKind},
			{number: 2, name: "value", kind: protoreflect.StringKind},
		},
		"Proof": {
			{number: 1, name: "method", kind: protoreflect.StringKind},
			{number: 2, name: "audience", kind: protoreflect.StringKind},
			{number: 3, name: "reference", kind: protoreflect.MessageKind, message: "authentication.ProofReference"},
		},
		"Principal": {
			{number: 1, name: "subject", kind: protoreflect.MessageKind, message: "authentication.Identity"},
			{number: 2, name: "actor", kind: protoreflect.MessageKind, message: "authentication.ServiceActor", optional: true},
			{number: 3, name: "selectedTenant", kind: protoreflect.MessageKind, message: "authentication.Tenant"},
			{number: 4, name: "proof", kind: protoreflect.MessageKind, message: "authentication.Proof"},
		},
		"Permission": {
			{number: 1, name: "id", kind: protoreflect.Uint64Kind},
			{number: 2, name: "key", kind: protoreflect.StringKind},
			{number: 3, name: "scope", kind: protoreflect.StringKind},
			{number: 4, name: "name", kind: protoreflect.StringKind},
			{number: 5, name: "description", kind: protoreflect.StringKind},
		},
		"Role": {
			{number: 1, name: "id", kind: protoreflect.Uint64Kind},
			{number: 2, name: "key", kind: protoreflect.StringKind},
			{number: 3, name: "name", kind: protoreflect.StringKind},
			{number: 4, name: "description", kind: protoreflect.StringKind},
			{number: 5, name: "scope", kind: protoreflect.StringKind},
			{number: 6, name: "subject", kind: protoreflect.StringKind},
			{number: 7, name: "definitionType", kind: protoreflect.StringKind},
			{number: 8, name: "permissions", kind: protoreflect.MessageKind, message: "authentication.Permission", list: true},
		},
		"BillingPlanQuota": {
			{number: 1, name: "id", kind: protoreflect.Uint64Kind},
			{number: 2, name: "resourceType", kind: protoreflect.StringKind},
			{number: 3, name: "quotaLimit", kind: protoreflect.Int64Kind},
			{number: 4, name: "unit", kind: protoreflect.StringKind},
		},
		"BillingPlan": {
			{number: 1, name: "id", kind: protoreflect.Uint64Kind},
			{number: 2, name: "name", kind: protoreflect.StringKind},
			{number: 3, name: "slug", kind: protoreflect.StringKind},
			{number: 4, name: "description", kind: protoreflect.StringKind},
			{number: 5, name: "isDefault", kind: protoreflect.BoolKind},
			{number: 6, name: "isActive", kind: protoreflect.BoolKind},
			{number: 7, name: "sortOrder", kind: protoreflect.Int32Kind},
			{number: 8, name: "priceMonthly", kind: protoreflect.Int64Kind},
			{number: 9, name: "priceYearly", kind: protoreflect.Int64Kind},
			{number: 10, name: "currency", kind: protoreflect.StringKind},
			{number: 11, name: "stripeUrl", kind: protoreflect.StringKind},
			{number: 12, name: "quotas", kind: protoreflect.MessageKind, message: "authentication.BillingPlanQuota", list: true},
		},
		"Project": {
			{number: 1, name: "id", kind: protoreflect.Uint64Kind},
			{number: 2, name: "name", kind: protoreflect.StringKind},
			{number: 3, name: "roles", kind: protoreflect.MessageKind, message: "authentication.Role", list: true},
		},
		"Organization": {
			{number: 1, name: "id", kind: protoreflect.Uint64Kind},
			{number: 2, name: "name", kind: protoreflect.StringKind},
			{number: 3, name: "roles", kind: protoreflect.MessageKind, message: "authentication.Role", list: true},
			{number: 4, name: "billingPlan", kind: protoreflect.MessageKind, message: "authentication.BillingPlan", optional: true},
			{number: 5, name: "projects", kind: protoreflect.MessageKind, message: "authentication.Project", list: true},
		},
		"User": {
			{number: 1, name: "id", kind: protoreflect.Uint64Kind},
			{number: 2, name: "name", kind: protoreflect.StringKind},
			{number: 3, name: "email", kind: protoreflect.StringKind},
			{number: 4, name: "status", kind: protoreflect.StringKind},
		},
		"Authentication": {
			{number: 1, name: "principal", kind: protoreflect.MessageKind, message: "authentication.Principal"},
			{number: 2, name: "user", kind: protoreflect.MessageKind, message: "authentication.User", optional: true},
			{number: 3, name: "organizations", kind: protoreflect.MessageKind, message: "authentication.Organization", list: true},
		},
	}

	messages := authentication_proto.File_authentication_proto.Messages()
	if messages.Len() != len(contracts) {
		t.Fatalf("authentication message count = %d, want %d", messages.Len(), len(contracts))
	}
	for messageName, fields := range contracts {
		message := messages.ByName(messageName)
		if message == nil || message.FullName() != protoreflect.FullName("authentication."+messageName) || message.Fields().Len() != len(fields) {
			t.Fatalf("authentication.%s = %v", messageName, message)
		}
		for _, contract := range fields {
			field := message.Fields().ByNumber(contract.number)
			if field == nil || field.Name() != contract.name || field.Kind() != contract.kind || field.IsList() != contract.list || field.HasOptionalKeyword() != contract.optional {
				t.Fatalf("authentication.%s field %d = %v", messageName, contract.number, field)
			}
			if contract.message != "" && field.Message().FullName() != contract.message {
				t.Fatalf("authentication.%s.%s message = %s", messageName, contract.name, field.Message().FullName())
			}
		}
	}
	if messages.ByName("Session") != nil || messages.ByName("Authorization") != nil {
		t.Fatal("superseded authentication Session or Authorization still exists")
	}

	if services := authentication_proto.File_authentication_proto.Services(); services.Len() != 0 {
		t.Fatalf("authentication services = %v", services)
	}

	webMessages := File_web_api_proto.Messages()
	if session := webMessages.ByName("AuthenticationSession"); session != nil {
		t.Fatalf("removed web_api.AuthenticationSession = %v", session)
	}

	response := webMessages.ByName("AuthenticateResponse")
	if response == nil {
		t.Fatal("web_api.AuthenticateResponse is absent")
	}
	responseAuthenticationField := response.Fields().ByNumber(6)
	tokenField := response.Fields().ByNumber(7)
	if response.Fields().Len() != 5 ||
		response.Fields().ByNumber(3) != nil || response.Fields().ByName("data") != nil ||
		response.Fields().ByNumber(5) != nil || response.Fields().ByName("session") != nil ||
		!response.ReservedRanges().Has(3) || !response.ReservedRanges().Has(5) ||
		!response.ReservedNames().Has("data") || !response.ReservedNames().Has("session") {
		t.Fatalf("AuthenticateResponse removed fields are not reserved: %v", response)
	}
	if responseAuthenticationField == nil || responseAuthenticationField.Name() != "authentication" ||
		responseAuthenticationField.Message().FullName() != "authentication.Authentication" ||
		tokenField == nil || tokenField.Name() != "token" || tokenField.Message().FullName() != "web_api.Token" {
		t.Fatalf("AuthenticateResponse flat fields = authentication %v, token %v", responseAuthenticationField, tokenField)
	}
	for _, removed := range []protoreflect.Name{"Authentication", "ScopedAuthentication", "OrganizationRole", "ProjectRole", "FeaturePermission", "AuthBillingPlan", "ScopeAuthorizeRequest", "ScopedAuthenticationResponse"} {
		if webMessages.ByName(removed) != nil {
			t.Fatalf("legacy web authentication message %s still exists", removed)
		}
	}
	webAuthentication := File_web_api_proto.Services().ByName("AuthenticationService")
	if webAuthentication == nil || webAuthentication.Methods().ByName("ScopeAuthorize") != nil {
		t.Fatalf("legacy ScopeAuthorize method still exists: %v", webAuthentication)
	}
	verifyRequest := webMessages.ByName("VerifyTokenRequest")
	if verifyRequest == nil || verifyRequest.Fields().Len() != 3 || verifyRequest.Oneofs().Len() != 1 ||
		verifyRequest.Oneofs().Get(0).Name() != "credential" || !verifyRequest.ReservedRanges().Has(1) ||
		!verifyRequest.ReservedRanges().Has(2) || !verifyRequest.ReservedNames().Has("tokenType") ||
		!verifyRequest.ReservedNames().Has("token") {
		t.Fatalf("web_api.VerifyTokenRequest = %v", verifyRequest)
	}
	verifyFields := []struct {
		number  protoreflect.FieldNumber
		name    protoreflect.Name
		message protoreflect.FullName
	}{
		{number: 3, name: "user", message: "web_api.VerifyTokenRequest.UserCredential"},
		{number: 4, name: "project", message: "web_api.VerifyTokenRequest.ProjectCredential"},
		{number: 5, name: "organization", message: "web_api.VerifyTokenRequest.OrganizationCredential"},
	}
	for _, contract := range verifyFields {
		field := verifyRequest.Fields().ByNumber(contract.number)
		if field == nil || field.Name() != contract.name || field.Message().FullName() != contract.message || field.ContainingOneof() != verifyRequest.Oneofs().Get(0) {
			t.Fatalf("VerifyTokenRequest field %d = %v", contract.number, field)
		}
	}
	userCredential := verifyRequest.Messages().ByName("UserCredential")
	if userCredential == nil || userCredential.Fields().Len() != 4 ||
		userCredential.Fields().ByNumber(1).Name() != "token" ||
		userCredential.Fields().ByNumber(2).Name() != "userId" ||
		userCredential.Fields().ByNumber(3).Name() != "selectedOrganizationId" ||
		userCredential.Fields().ByNumber(4).Name() != "selectedProjectId" {
		t.Fatalf("web_api.VerifyTokenRequest.UserCredential = %v", userCredential)
	}
	for _, number := range []protoreflect.FieldNumber{2, 3, 4} {
		options, ok := userCredential.Fields().ByNumber(number).Options().(*descriptorpb.FieldOptions)
		if !ok || options.GetJstype() != descriptorpb.FieldOptions_JS_STRING {
			t.Fatalf("VerifyTokenRequest.UserCredential field %d options = %v", number, options)
		}
	}
	for _, name := range []protoreflect.Name{"ProjectCredential", "OrganizationCredential"} {
		credential := verifyRequest.Messages().ByName(name)
		if credential == nil || credential.Fields().Len() != 1 || credential.Fields().ByNumber(1).Name() != "token" {
			t.Fatalf("web_api.VerifyTokenRequest.%s = %v", name, credential)
		}
	}
	verifyResponse := webMessages.ByName("VerifyTokenResponse")
	if verifyResponse == nil {
		t.Fatal("web_api.VerifyTokenResponse is missing")
	}
	authenticationField := verifyResponse.Fields().ByNumber(4)
	if verifyResponse.Fields().Len() != 1 || authenticationField == nil ||
		authenticationField.Name() != "authentication" || authenticationField.Message().FullName() != "authentication.Authentication" ||
		!verifyResponse.ReservedRanges().Has(1) || !verifyResponse.ReservedRanges().Has(2) || !verifyResponse.ReservedRanges().Has(3) ||
		!verifyResponse.ReservedNames().Has("code") || !verifyResponse.ReservedNames().Has("success") || !verifyResponse.ReservedNames().Has("data") {
		t.Fatalf("web_api.VerifyTokenResponse = %v", verifyResponse)
	}
	verifyMethod := webAuthentication.Methods().ByName("VerifyToken")
	if verifyMethod == nil || verifyMethod.Input().FullName() != "web_api.VerifyTokenRequest" ||
		verifyMethod.Output().FullName() != "web_api.VerifyTokenResponse" || verifyMethod.IsStreamingClient() || verifyMethod.IsStreamingServer() {
		t.Fatalf("web_api.AuthenticationService.VerifyToken = %v", verifyMethod)
	}
}
