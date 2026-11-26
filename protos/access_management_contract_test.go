// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package protos

import (
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestAccessManagementReductionContract(t *testing.T) {
	file := File_access_management_api_proto
	request := file.Messages().ByName("ReduceCustomRolePermissionsRequest")
	response := file.Messages().ByName("ReduceCustomRolePermissionsResponse")
	impact := file.Messages().ByName("RoleReductionImpact")
	preview := file.Messages().ByName("RoleReductionPreview")

	assertMessageFields(t, request, []fieldContract{
		{name: "roleDefinitionId", number: 1, kind: protoreflect.Uint64Kind, jsString: true},
		{name: "permissionDefinitionIds", number: 2, kind: protoreflect.Uint64Kind, cardinality: protoreflect.Repeated, jsString: true},
		{name: "reason", number: 3, kind: protoreflect.StringKind},
		{name: "confirmationDigest", number: 4, kind: protoreflect.StringKind},
		{name: "preview", number: 5, kind: protoreflect.BoolKind},
	})
	assertMessageFields(t, response, []fieldContract{
		{name: "code", number: 1, kind: protoreflect.Int32Kind},
		{name: "success", number: 2, kind: protoreflect.BoolKind},
		{name: "role", number: 3, kind: protoreflect.MessageKind, typeName: "web_api.RoleDefinition"},
		{name: "error", number: 4, kind: protoreflect.MessageKind, typeName: "Error"},
		{name: "correlationId", number: 5, kind: protoreflect.StringKind},
		{name: "preview", number: 6, kind: protoreflect.MessageKind, typeName: "web_api.RoleReductionPreview"},
	})
	assertMessageFields(t, impact, []fieldContract{
		{name: "organizationId", number: 1, kind: protoreflect.Uint64Kind, jsString: true},
		{name: "projectId", number: 2, kind: protoreflect.Uint64Kind, jsString: true},
		{name: "subject", number: 3, kind: protoreflect.EnumKind, typeName: "web_api.RoleSubject"},
		{name: "subjectId", number: 4, kind: protoreflect.Uint64Kind, jsString: true},
		{name: "assignmentStatus", number: 5, kind: protoreflect.StringKind},
	})
	assertMessageFields(t, preview, []fieldContract{
		{name: "resultingRole", number: 1, kind: protoreflect.MessageKind, typeName: "web_api.RoleDefinition"},
		{name: "impacts", number: 2, kind: protoreflect.MessageKind, cardinality: protoreflect.Repeated, typeName: "web_api.RoleReductionImpact"},
		{name: "confirmationDigest", number: 3, kind: protoreflect.StringKind},
	})

	service := file.Services().ByName("AccessManagementService")
	if service == nil {
		t.Fatal("AccessManagementService descriptor is missing")
	}
	method := service.Methods().ByName("ReduceCustomRolePermissions")
	if method == nil {
		t.Fatal("ReduceCustomRolePermissions method descriptor is missing")
	}
	if got, want := method.Input().FullName(), protoreflect.FullName("web_api.ReduceCustomRolePermissionsRequest"); got != want {
		t.Errorf("ReduceCustomRolePermissions input = %q, want %q", got, want)
	}
	if got, want := method.Output().FullName(), protoreflect.FullName("web_api.ReduceCustomRolePermissionsResponse"); got != want {
		t.Errorf("ReduceCustomRolePermissions output = %q, want %q", got, want)
	}
}

type fieldContract struct {
	name        protoreflect.Name
	number      protoreflect.FieldNumber
	kind        protoreflect.Kind
	cardinality protoreflect.Cardinality
	typeName    protoreflect.FullName
	jsString    bool
}

func assertMessageFields(t *testing.T, message protoreflect.MessageDescriptor, expected []fieldContract) {
	t.Helper()
	if message == nil {
		t.Fatal("message descriptor is missing")
	}
	if got := message.Fields().Len(); got != len(expected) {
		t.Fatalf("%s field count = %d, want %d", message.FullName(), got, len(expected))
	}
	for _, contract := range expected {
		field := message.Fields().ByName(contract.name)
		if field == nil {
			t.Errorf("%s.%s is missing", message.FullName(), contract.name)
			continue
		}
		if got := field.Number(); got != contract.number {
			t.Errorf("%s.%s number = %d, want %d", message.FullName(), contract.name, got, contract.number)
		}
		if got := field.Kind(); got != contract.kind {
			t.Errorf("%s.%s kind = %s, want %s", message.FullName(), contract.name, got, contract.kind)
		}
		cardinality := contract.cardinality
		if cardinality == 0 {
			cardinality = protoreflect.Optional
		}
		if got := field.Cardinality(); got != cardinality {
			t.Errorf("%s.%s cardinality = %s, want %s", message.FullName(), contract.name, got, cardinality)
		}
		if contract.jsString {
			options, ok := field.Options().(*descriptorpb.FieldOptions)
			if !ok {
				t.Errorf("%s.%s options have type %T", message.FullName(), contract.name, field.Options())
			} else if got := options.GetJstype(); got != descriptorpb.FieldOptions_JS_STRING {
				t.Errorf("%s.%s jstype = %s, want %s", message.FullName(), contract.name, got, descriptorpb.FieldOptions_JS_STRING)
			}
		}
		if contract.typeName == "" {
			continue
		}
		var got protoreflect.FullName
		switch field.Kind() {
		case protoreflect.MessageKind:
			got = field.Message().FullName()
		case protoreflect.EnumKind:
			got = field.Enum().FullName()
		default:
			t.Errorf("%s.%s has no referenced type", message.FullName(), contract.name)
			continue
		}
		if got != contract.typeName {
			t.Errorf("%s.%s type = %q, want %q", message.FullName(), contract.name, got, contract.typeName)
		}
	}
}
