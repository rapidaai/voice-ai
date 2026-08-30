package main

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/rapidaai/protos"
)

func TestRepositoryDescriptorSetIsSortedAndComplete(t *testing.T) {
	descriptorSet, err := repositoryDescriptorSet()
	if err != nil {
		t.Fatal(err)
	}

	paths := make([]string, 0, len(descriptorSet.File))
	seen := make(map[string]bool, len(descriptorSet.File))
	for _, file := range descriptorSet.File {
		path := file.GetName()
		if seen[path] {
			t.Fatalf("duplicate descriptor %q", path)
		}
		seen[path] = true
		paths = append(paths, path)
	}
	if !sort.StringsAreSorted(paths) {
		t.Fatalf("descriptor paths are not sorted: %v", paths)
	}
	for _, required := range []string{"assistant-api.proto", "talk-api.proto", "common.proto", "google/protobuf/timestamp.proto"} {
		if !seen[required] {
			t.Errorf("missing descriptor %q", required)
		}
	}
}

func TestProductUsageContract(t *testing.T) {
	file := protos.File_billing_api_proto
	service := file.Services().ByName("ProductUsageService")
	if service == nil {
		t.Fatal("ProductUsageService descriptor is missing")
	}
	for _, methodName := range []protoreflect.Name{"CreateProductUsage", "GetProductUsages", "GetOrganizationUsages"} {
		if service.Methods().ByName(methodName) == nil {
			t.Fatalf("%s descriptor is missing", methodName)
		}
	}
	usage := file.Messages().ByName("ProductUsage")
	if usage == nil {
		t.Fatal("ProductUsage descriptor is missing")
	}
	for name, number := range map[protoreflect.Name]protoreflect.FieldNumber{
		"id": 1, "projectId": 2, "usageType": 3, "usages": 4, "unit": 5, "occurredAt": 6,
	} {
		field := usage.Fields().ByName(name)
		if field == nil || field.Number() != number {
			t.Errorf("ProductUsage.%s number = %v, want %d", name, field, number)
		}
	}
	createRequest := file.Messages().ByName("CreateProductUsageRequest")
	if createRequest == nil || createRequest.Fields().ByName("id") != nil {
		t.Fatalf("CreateProductUsageRequest must exist without an id field")
	}
	quota := file.Messages().ByName("BillingPlanQuota")
	if field := quota.Fields().ByName("unit"); field == nil || field.Number() != 4 {
		t.Fatalf("BillingPlanQuota.unit descriptor = %v", field)
	}
}

func TestWriteDescriptorSetIsDeterministic(t *testing.T) {
	directory := t.TempDir()
	first := filepath.Join(directory, "first.binpb")
	second := filepath.Join(directory, "second.binpb")
	if err := writeDescriptorSet(first); err != nil {
		t.Fatal(err)
	}
	if err := writeDescriptorSet(second); err != nil {
		t.Fatal(err)
	}

	firstData, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	secondData, err := os.ReadFile(second)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstData, secondData) {
		t.Fatal("descriptor output is not deterministic")
	}

	var descriptorSet descriptorpb.FileDescriptorSet
	if err := proto.Unmarshal(firstData, &descriptorSet); err != nil {
		t.Fatalf("unmarshal descriptor set: %v", err)
	}
	if len(descriptorSet.File) == 0 {
		t.Fatal("descriptor set is empty")
	}
}
