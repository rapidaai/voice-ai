package systemcheck

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

var inheritedServices = []string{"postgres", "redis", "nginx", "ui", "web-api", "integration-api", "endpoint-api", "assistant-api"}
var addedCIServices = []string{"migrate-web", "migrate-integration", "migrate-endpoint", "migrate-assistant", "test-runner"}

func validateComposeContract(basePath, overridePath, mergedPath, composeVersion, forbiddenPath string) error {
	if composeVersion != "2.24.4" {
		return fmt.Errorf("unsupported Compose version %q", composeVersion)
	}
	if forbiddenPath != "" {
		resolvedForbidden := filepath.Join(filepath.Dir(overridePath), forbiddenPath)
		if _, err := os.Stat(resolvedForbidden); err == nil {
			return fmt.Errorf("forbidden path exists: %s", forbiddenPath)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	base, err := readJSONMap(basePath)
	if err != nil {
		return fmt.Errorf("base model: %w", err)
	}
	merged, err := readJSONMap(mergedPath)
	if err != nil {
		return fmt.Errorf("merged model: %w", err)
	}
	overrideBytes, err := os.ReadFile(overridePath)
	if err != nil {
		return err
	}
	if forbiddenPath != "" {
		mergedBytes, err := os.ReadFile(mergedPath)
		if err != nil {
			return err
		}
		if strings.Contains(string(overrideBytes), forbiddenPath) || strings.Contains(string(mergedBytes), forbiddenPath) {
			return fmt.Errorf("Compose model references forbidden path %q", forbiddenPath)
		}
	}
	if err := validateOverrideKeys(overrideBytes); err != nil {
		return err
	}
	baseServices := mapValue(base, "services")
	mergedServices := mapValue(merged, "services")
	for _, name := range inheritedServices {
		baseService, baseOK := baseServices[name].(map[string]any)
		mergedService, mergedOK := mergedServices[name].(map[string]any)
		if !baseOK || !mergedOK {
			return fmt.Errorf("inherited service %q missing", name)
		}
		if err := compareInheritedService(name, baseService, mergedService); err != nil {
			return err
		}
	}
	for _, name := range addedCIServices {
		if _, ok := mergedServices[name]; !ok {
			return fmt.Errorf("CI service %q missing", name)
		}
	}
	for name := range mergedServices {
		if _, inherited := baseServices[name]; inherited {
			continue
		}
		if !contains(addedCIServices, name) {
			return fmt.Errorf("unexpected CI service %q", name)
		}
	}
	if err := validateMergedIsolation(mergedServices); err != nil {
		return err
	}
	if !reflect.DeepEqual(base["networks"], merged["networks"]) {
		return errors.New("CI override changes network topology")
	}
	baseVolumes, mergedVolumes := mapValue(base, "volumes"), mapValue(merged, "volumes")
	for name := range baseVolumes {
		if _, ok := mergedVolumes[name]; !ok {
			return fmt.Errorf("base volume %q missing", name)
		}
	}
	for name := range mergedVolumes {
		if _, ok := baseVolumes[name]; !ok && !contains([]string{"postgres-data", "system-assets", "system-reports"}, name) {
			return fmt.Errorf("unexpected CI volume %q", name)
		}
	}
	for _, name := range []string{"postgres-data", "system-assets", "system-reports"} {
		if _, ok := mergedVolumes[name]; !ok {
			return fmt.Errorf("required CI volume %q missing", name)
		}
	}
	return nil
}

func readJSONMap(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var value map[string]any
	err = json.Unmarshal(data, &value)
	return value, err
}

func mapValue(value map[string]any, key string) map[string]any {
	result, _ := value[key].(map[string]any)
	return result
}

func validateOverrideKeys(data []byte) error {
	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		return err
	}
	root := document.Content[0]
	for index := 0; index < len(root.Content); index += 2 {
		if root.Content[index].Value != "services" && root.Content[index].Value != "volumes" {
			return fmt.Errorf("unexpected override top-level key %q", root.Content[index].Value)
		}
	}
	services := mappingNode(root, "services")
	allowed := map[string]map[string]bool{
		"postgres":        {"container_name": true, "ports": true, "volumes": true},
		"redis":           {"container_name": true, "ports": true, "volumes": true, "tmpfs": true},
		"nginx":           {"container_name": true, "ports": true, "volumes": true},
		"ui":              {"container_name": true, "ports": true, "build": true},
		"web-api":         {"container_name": true, "ports": true, "command": true, "build": true, "volumes": true},
		"integration-api": {"container_name": true, "ports": true, "command": true, "build": true, "volumes": true},
		"endpoint-api":    {"container_name": true, "ports": true, "command": true, "build": true, "volumes": true},
		"assistant-api":   {"container_name": true, "ports": true, "command": true, "build": true, "volumes": true},
	}
	for index := 0; index < len(services.Content); index += 2 {
		name, service := services.Content[index].Value, services.Content[index+1]
		if !contains(inheritedServices, name) {
			continue
		}
		for fieldIndex := 0; fieldIndex < len(service.Content); fieldIndex += 2 {
			field, value := service.Content[fieldIndex].Value, service.Content[fieldIndex+1]
			if !allowed[name][field] {
				return fmt.Errorf("override duplicates disallowed field %s.%s", name, field)
			}
			if (field == "container_name" || field == "ports") && value.Tag != "!reset" {
				return fmt.Errorf("%s.%s must use !reset", name, field)
			}
			if name == "redis" && field == "volumes" && value.Tag != "!override" {
				return errors.New("redis.volumes must use !override")
			}
		}
	}
	return nil
}

func mappingNode(node *yaml.Node, key string) *yaml.Node {
	for index := 0; index < len(node.Content); index += 2 {
		if node.Content[index].Value == key {
			return node.Content[index+1]
		}
	}
	return &yaml.Node{}
}

func compareInheritedService(name string, base, merged map[string]any) error {
	baseCopy, mergedCopy := cloneMap(base), cloneMap(merged)
	for _, key := range []string{"container_name", "ports"} {
		delete(baseCopy, key)
		delete(mergedCopy, key)
	}
	if contains([]string{"postgres", "redis", "nginx", "web-api", "integration-api", "endpoint-api", "assistant-api"}, name) {
		delete(baseCopy, "volumes")
		delete(mergedCopy, "volumes")
	}
	if contains([]string{"web-api", "integration-api", "endpoint-api", "assistant-api"}, name) {
		delete(baseCopy, "command")
		delete(mergedCopy, "command")
	}
	if name == "redis" {
		delete(baseCopy, "tmpfs")
		delete(mergedCopy, "tmpfs")
	}
	if contains([]string{"ui", "web-api", "integration-api", "endpoint-api", "assistant-api"}, name) {
		stripBuildCaches(baseCopy)
		stripBuildCaches(mergedCopy)
	}
	if !reflect.DeepEqual(baseCopy, mergedCopy) {
		return fmt.Errorf("unapproved inherited-field change for %s", name)
	}
	if _, ok := merged["container_name"]; ok {
		return fmt.Errorf("%s retains container_name", name)
	}
	if ports, ok := merged["ports"].([]any); ok && len(ports) != 0 {
		return fmt.Errorf("%s retains ports", name)
	}
	return validateServiceDelta(name, base, merged)
}

func cloneMap(value map[string]any) map[string]any {
	data, _ := json.Marshal(value)
	var clone map[string]any
	_ = json.Unmarshal(data, &clone)
	return clone
}

func stripBuildCaches(service map[string]any) {
	build, ok := service["build"].(map[string]any)
	if !ok {
		return
	}
	delete(build, "cache_from")
	delete(build, "cache_to")
}

func validateServiceDelta(name string, base, merged map[string]any) error {
	switch name {
	case "postgres":
		if !reflect.DeepEqual(volumesExceptTarget(base, "/var/lib/postgresql/data"), volumesExceptTarget(merged, "/var/lib/postgresql/data")) {
			return errors.New("postgres non-data mounts changed")
		}
		if !hasVolumeTarget(merged, "/var/lib/postgresql/data", "postgres-data") || !hasVolumeTarget(merged, "/docker-entrypoint-initdb.d/init.sql", "") {
			return errors.New("postgres volume substitution is invalid")
		}
	case "redis":
		if !reflect.DeepEqual(volumesExceptTarget(base, "/data"), volumesExceptTarget(merged, "/data")) {
			return errors.New("redis non-data mounts changed")
		}
		if hasVolumeTarget(merged, "/data", "") || !stringListContains(merged["tmpfs"], "/data") {
			return errors.New("redis must use only tmpfs at /data")
		}
	case "nginx", "web-api", "integration-api", "endpoint-api", "assistant-api":
		if !reflect.DeepEqual(volumesExceptTarget(base, "/app/rapida-data/assets"), volumesExceptTarget(merged, "/app/rapida-data/assets")) {
			return fmt.Errorf("%s non-asset mounts changed", name)
		}
		if !hasVolumeTarget(merged, "/app/rapida-data/assets", "system-assets") {
			return fmt.Errorf("%s asset volume substitution is invalid", name)
		}
	}
	if contains([]string{"web-api", "integration-api", "endpoint-api", "assistant-api"}, name) {
		command := stringList(merged["command"])
		if len(command) != 2 || command[1] != "-skip-migration" {
			return fmt.Errorf("%s command does not select explicit migration owner", name)
		}
	}
	if contains([]string{"ui", "web-api", "integration-api", "endpoint-api", "assistant-api"}, name) {
		build, _ := merged["build"].(map[string]any)
		if len(stringList(build["cache_from"])) == 0 || len(stringList(build["cache_to"])) == 0 {
			return fmt.Errorf("%s build cache is missing", name)
		}
	}
	_ = base
	return nil
}

func volumesExceptTarget(service map[string]any, excluded string) []any {
	volumes, _ := service["volumes"].([]any)
	result := make([]any, 0, len(volumes))
	for _, item := range volumes {
		volume, _ := item.(map[string]any)
		if volume["target"] != excluded {
			result = append(result, item)
		}
	}
	return result
}

func hasVolumeTarget(service map[string]any, target, sourceSuffix string) bool {
	volumes, _ := service["volumes"].([]any)
	for _, item := range volumes {
		volume, _ := item.(map[string]any)
		if volume["target"] != target {
			continue
		}
		if sourceSuffix == "" {
			return true
		}
		source, _ := volume["source"].(string)
		return strings.HasSuffix(source, sourceSuffix)
	}
	return false
}

func stringList(value any) []string {
	items, _ := value.([]any)
	result := make([]string, 0, len(items))
	for _, item := range items {
		if text, ok := item.(string); ok {
			result = append(result, text)
		}
	}
	return result
}

func stringListContains(value any, expected string) bool {
	for _, item := range stringList(value) {
		if item == expected {
			return true
		}
	}
	return false
}

func validateMergedIsolation(services map[string]any) error {
	for name, raw := range services {
		service, _ := raw.(map[string]any)
		if nameValue, ok := service["container_name"].(string); ok && nameValue != "" {
			return fmt.Errorf("%s retains container_name", name)
		}
		if service["privileged"] == true || service["network_mode"] == "host" {
			return fmt.Errorf("%s uses forbidden isolation option", name)
		}
		if ports, ok := service["ports"].([]any); ok && len(ports) > 0 {
			return fmt.Errorf("%s publishes ports", name)
		}
		volumes, _ := service["volumes"].([]any)
		for _, volume := range volumes {
			text := fmt.Sprint(volume)
			entry, _ := volume.(map[string]any)
			source, _ := entry["source"].(string)
			typeName, _ := entry["type"].(string)
			if (typeName == "bind" && strings.Contains(source, "/rapida-data/assets")) || strings.Contains(text, "/var/run/docker.sock") {
				return fmt.Errorf("%s has forbidden mount", name)
			}
		}
	}
	return nil
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
