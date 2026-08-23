package systemcheck

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var buildDiagnosticSources = []string{
	"buildx-builder.txt",
	"buildx-du.txt",
	"docker-system-df.txt",
	"cache-scope.txt",
	"buildkit.log",
	"compose-images.json",
}

func collectBuildDiagnostics(ctx context.Context, run commandRunner, directory string) error {
	record := newBuildDiagnosticRecord()
	var composeImages map[string]string
	if data, ok, err := readOptionalFile(filepath.Join(directory, "buildx-builder.txt")); err != nil {
		return err
	} else if ok {
		record.Builder, record.Driver, record.Status, record.Buildkit = parseBuildxInspect(string(data))
	}
	if data, ok, err := readOptionalFile(filepath.Join(directory, "cache-scope.txt")); err != nil {
		return err
	} else if ok {
		record.CacheScope = strings.TrimSpace(string(data))
		if !regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`).MatchString(record.CacheScope) {
			return errors.New("invalid cache scope in diagnostics")
		}
	}
	for _, name := range []string{"buildx-du.txt", "docker-system-df.txt"} {
		data, ok, err := readOptionalFile(filepath.Join(directory, name))
		if err != nil {
			return err
		}
		if ok {
			record.DiskUsage = append(record.DiskUsage, allowlistedDiskUsage(string(data))...)
		}
	}
	if data, ok, err := readOptionalFile(filepath.Join(directory, "buildkit.log")); err != nil {
		return err
	} else if ok {
		record.Logs = selectedLogs(string(data))
	}
	if data, ok, err := readOptionalFile(filepath.Join(directory, "buildkit-metadata.json")); err != nil {
		return err
	} else if ok {
		var metadata buildMetadata
		if err := json.Unmarshal(data, &metadata); err != nil {
			return fmt.Errorf("parse build metadata: %w", err)
		}
		if err := validateBuildMetadata(metadata); err != nil {
			return err
		}
		record.Metadata = &metadata
		if record.Builder == "" {
			record.Builder = metadata.Builder
		}
		if record.CacheScope == "" {
			record.CacheScope = metadata.CacheScope
		}
	}
	if data, ok, err := readOptionalFile(filepath.Join(directory, "compose-images.json")); err != nil {
		return err
	} else if ok && json.Valid(data) && strings.HasPrefix(strings.TrimSpace(string(data)), "{") {
		composeImages, err = decodeComposeImageMapping(strings.NewReader(string(data)))
		if err != nil {
			return err
		}
	}
	if record.Metadata != nil && len(composeImages) > 0 {
		for _, image := range record.Metadata.Images {
			if composeImages[image.Service] != image.ImageRef {
				return fmt.Errorf("build metadata image reference for %s does not match Compose mapping", image.Service)
			}
		}
	}
	if record.Metadata == nil && len(composeImages) > 0 {
		metadata := buildMetadata{Builder: record.Builder, CacheScope: record.CacheScope}
		for _, service := range builtServices {
			reference := composeImages[service]
			image := buildImageMetadata{Service: service, ImageRef: reference, Status: "unavailable"}
			var stdout bytes.Buffer
			if err := run(ctx, "docker", []string{"image", "inspect", "--format", "{{.Id}}", reference}, nil, &stdout, io.Discard, nil); err == nil {
				imageID := strings.TrimSpace(stdout.String())
				if regexpImageID.MatchString(imageID) {
					image.ImageID = imageID
					image.Status = "available"
				}
			}
			metadata.Images = append(metadata.Images, image)
		}
		record.Metadata = &metadata
	}
	for _, name := range buildDiagnosticSources {
		if err := os.Remove(filepath.Join(directory, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(directory, "build-diagnostics.json"), data, 0600)
}

func validateBuildMetadata(metadata buildMetadata) error {
	if metadata.Builder == "" || metadata.CacheScope == "" {
		return errors.New("build metadata is missing builder or cache scope")
	}
	if !safeIdentifier(metadata.Builder) || !safeIdentifier(metadata.CacheScope) {
		return errors.New("build metadata contains an invalid builder or cache scope")
	}
	seen := make(map[string]bool, len(metadata.Images))
	for _, image := range metadata.Images {
		if image.Service == "" || !validImageReference(image.ImageRef) || !regexpImageID.MatchString(image.ImageID) {
			return errors.New("build metadata contains an incomplete image record")
		}
		if !contains(builtServices, image.Service) {
			return fmt.Errorf("build metadata contains unexpected service %s", image.Service)
		}
		if seen[image.Service] {
			return fmt.Errorf("build metadata contains duplicate service %s", image.Service)
		}
		seen[image.Service] = true
	}
	for _, service := range builtServices {
		if !seen[service] {
			return fmt.Errorf("build metadata missing %s", service)
		}
	}
	if len(metadata.Images) != len(builtServices) {
		return fmt.Errorf("build metadata contains %d image records; want %d", len(metadata.Images), len(builtServices))
	}
	return nil
}

func readOptionalFile(path string) ([]byte, bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	return data, err == nil, err
}

func parseBuildxInspect(raw string) (name, driver, status, buildkit string) {
	for _, line := range strings.Split(raw, "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok {
			continue
		}
		value = strings.TrimSpace(sanitize(value))
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "name":
			name = value
		case "driver":
			driver = value
		case "status":
			status = value
		case "buildkit":
			buildkit = value
		}
	}
	return name, driver, status, buildkit
}

func allowlistedDiskUsage(raw string) []string {
	allowed := regexp.MustCompile(`(?i)^(total:|images\s|containers\s|local volumes\s|build cache\s)`)
	var result []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && allowed.MatchString(line) {
			result = append(result, sanitize(line))
		}
	}
	return result
}

func newBuildDiagnosticRecord() buildDiagnosticRecord {
	return buildDiagnosticRecord{Timestamp: time.Now().UTC().Format(time.RFC3339)}
}
