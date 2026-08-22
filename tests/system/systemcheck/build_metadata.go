package systemcheck

import (
	"bufio"
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
)

var builtServices = []string{
	"web-api",
	"integration-api",
	"endpoint-api",
	"assistant-api",
	"ui",
	"test-runner",
}

type buildImageMetadata struct {
	Service  string `json:"service"`
	ImageRef string `json:"imageRef"`
	ImageID  string `json:"imageId"`
	Status   string `json:"status,omitempty"`
}

type buildMetadata struct {
	Builder    string               `json:"builder"`
	CacheScope string               `json:"cacheScope"`
	Images     []buildImageMetadata `json:"images"`
}

func writeBuildMetadata(ctx context.Context, run commandRunner, composeImagesPath, outputPath, builder, cacheScope string) error {
	if composeImagesPath == "" || outputPath == "" || builder == "" || cacheScope == "" {
		return errors.New("compose images flag, output, BUILDX_BUILDER, and SYSTEM_CACHE_SCOPE are required")
	}
	if !safeIdentifier(builder) || !safeIdentifier(cacheScope) {
		return errors.New("builder and cache scope must be opaque-safe identifiers")
	}
	input, err := os.Open(composeImagesPath)
	if err != nil {
		return fmt.Errorf("open Compose image mapping: %w", err)
	}
	images, err := decodeComposeImageMapping(input)
	closeErr := input.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	metadata := buildMetadata{Builder: builder, CacheScope: cacheScope}
	for _, service := range builtServices {
		reference := images[service]
		var stdout bytes.Buffer
		if err := run(ctx, "docker", []string{"image", "inspect", "--format", "{{.Id}}", reference}, nil, &stdout, io.Discard, nil); err != nil {
			return fmt.Errorf("inspect built image %s (%s): %w", service, reference, err)
		}
		imageID := strings.TrimSpace(stdout.String())
		if !regexpImageID.MatchString(imageID) {
			return fmt.Errorf("inspect built image %s (%s) returned invalid image ID", service, reference)
		}
		metadata.Images = append(metadata.Images, buildImageMetadata{Service: service, ImageRef: reference, ImageID: imageID, Status: "available"})
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, data, 0600)
}

func decodeComposeImageMapping(reader io.Reader) (map[string]string, error) {
	decoder := json.NewDecoder(reader)
	token, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("decode Compose image mapping: %w", err)
	}
	if delimiter, ok := token.(json.Delim); !ok || delimiter != '{' {
		return nil, errors.New("Compose image mapping must be a JSON object")
	}
	images := make(map[string]string, len(builtServices))
	normalizedServices := make(map[string]string, len(builtServices))
	for decoder.More() {
		serviceToken, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("decode Compose image service: %w", err)
		}
		service, ok := serviceToken.(string)
		if !ok {
			return nil, errors.New("Compose image mapping contains a non-string service")
		}
		normalized := strings.ToLower(strings.TrimSpace(service))
		if previous, duplicate := normalizedServices[normalized]; duplicate {
			return nil, fmt.Errorf("Compose image mapping contains duplicate services %q and %q after normalization", previous, service)
		}
		normalizedServices[normalized] = service
		if _, duplicate := images[service]; duplicate {
			return nil, fmt.Errorf("Compose image mapping contains duplicate service %q", service)
		}
		var reference string
		if err := decoder.Decode(&reference); err != nil {
			return nil, fmt.Errorf("decode Compose image reference for %q: %w", service, err)
		}
		images[service] = reference
	}
	if _, err := decoder.Token(); err != nil {
		return nil, fmt.Errorf("decode Compose image mapping: %w", err)
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return nil, errors.New("Compose image mapping contains trailing JSON data")
	}
	if err := validateComposeImageMapping(images); err != nil {
		return nil, err
	}
	return images, nil
}

func validateComposeImageMapping(images map[string]string) error {
	expected := make(map[string]bool, len(builtServices))
	for _, service := range builtServices {
		expected[service] = true
	}
	for service, reference := range images {
		if service != strings.ToLower(strings.TrimSpace(service)) || !expected[service] {
			return fmt.Errorf("Compose image mapping contains unexpected service %q", service)
		}
		if !validImageReference(reference) {
			return fmt.Errorf("Compose image mapping contains invalid image reference for %s", service)
		}
	}
	for _, service := range builtServices {
		if _, ok := images[service]; !ok {
			return fmt.Errorf("Compose image mapping missing service %q", service)
		}
	}
	return nil
}

func validImageReference(reference string) bool {
	return reference != "" && reference == strings.TrimSpace(reference) && !strings.ContainsAny(reference, " \t\r\n")
}

func composeImageIDs(path string) (map[string]string, bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, false, nil
	}
	if json.Valid(data) && strings.HasPrefix(strings.TrimSpace(string(data)), "{") {
		if _, err := decodeComposeImageMapping(bytes.NewReader(data)); err != nil {
			return nil, false, err
		}
		return nil, false, nil
	}
	type composeImageRecord struct {
		Service       string `json:"Service"`
		ContainerName string `json:"ContainerName"`
		ID            string `json:"ID"`
	}
	var records []composeImageRecord
	if err := json.Unmarshal(data, &records); err != nil {
		scanner := bufio.NewScanner(bytes.NewReader(data))
		for scanner.Scan() {
			line := bytes.TrimSpace(scanner.Bytes())
			if len(line) == 0 {
				continue
			}
			var record composeImageRecord
			if err := json.Unmarshal(line, &record); err != nil {
				return nil, false, fmt.Errorf("decode Compose image IDs: %w", err)
			}
			records = append(records, record)
		}
		if err := scanner.Err(); err != nil {
			return nil, false, err
		}
	}
	if len(records) == 0 {
		return nil, false, errors.New("Compose image IDs contain no records")
	}
	result := make(map[string]string, len(records))
	for _, record := range records {
		service := record.Service
		if service == "" || record.ID == "" {
			continue
		}
		if !regexpImageID.MatchString(record.ID) {
			return nil, false, fmt.Errorf("Compose image ID for %q is malformed", service)
		}
		result[service] = record.ID
	}
	return result, true, nil
}

var regexpImageID = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)

func safeIdentifier(value string) bool {
	return regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`).MatchString(value)
}
