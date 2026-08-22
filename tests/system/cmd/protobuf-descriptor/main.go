package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	_ "github.com/rapidaai/protos"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

const repositoryProtoPackage = "github.com/rapidaai/protos"

func main() {
	output := flag.String("output", "", "output FileDescriptorSet path")
	flag.Parse()
	if *output == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: protobuf-descriptor --output <path>")
		os.Exit(2)
	}
	if err := writeDescriptorSet(*output); err != nil {
		fmt.Fprintf(os.Stderr, "generate protobuf descriptor: %v\n", err)
		os.Exit(1)
	}
}

func writeDescriptorSet(output string) error {
	descriptorSet, err := repositoryDescriptorSet()
	if err != nil {
		return err
	}
	data, err := proto.MarshalOptions{Deterministic: true}.Marshal(descriptorSet)
	if err != nil {
		return fmt.Errorf("marshal descriptor set: %w", err)
	}

	directory := filepath.Dir(output)
	temporary, err := os.CreateTemp(directory, ".protobuf-descriptor-*")
	if err != nil {
		return fmt.Errorf("create temporary output: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return fmt.Errorf("protect temporary output: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return fmt.Errorf("write temporary output: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary output: %w", err)
	}
	if err := os.Rename(temporaryPath, output); err != nil {
		return fmt.Errorf("publish output: %w", err)
	}
	return nil
}

func repositoryDescriptorSet() (*descriptorpb.FileDescriptorSet, error) {
	files := make(map[string]protoreflect.FileDescriptor)
	protoregistry.GlobalFiles.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		options, ok := file.Options().(*descriptorpb.FileOptions)
		if ok && options.GetGoPackage() == repositoryProtoPackage {
			addFileAndDependencies(files, file)
		}
		return true
	})
	if len(files) == 0 {
		return nil, fmt.Errorf("no descriptors registered for %s", repositoryProtoPackage)
	}

	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	descriptorSet := &descriptorpb.FileDescriptorSet{File: make([]*descriptorpb.FileDescriptorProto, 0, len(paths))}
	for _, path := range paths {
		descriptorSet.File = append(descriptorSet.File, protodesc.ToFileDescriptorProto(files[path]))
	}
	return descriptorSet, nil
}

func addFileAndDependencies(files map[string]protoreflect.FileDescriptor, file protoreflect.FileDescriptor) {
	if _, exists := files[file.Path()]; exists {
		return
	}
	files[file.Path()] = file
	imports := file.Imports()
	for index := 0; index < imports.Len(); index++ {
		dependency, err := protoregistry.GlobalFiles.FindFileByPath(imports.Get(index).Path())
		if err == nil {
			addFileAndDependencies(files, dependency)
		}
	}
}
