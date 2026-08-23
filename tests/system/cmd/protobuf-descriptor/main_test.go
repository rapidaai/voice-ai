package main

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
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
