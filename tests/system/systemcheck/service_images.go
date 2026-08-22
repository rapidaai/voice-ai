package systemcheck

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type imageService struct {
	Image    string `yaml:"image"`
	Platform string `yaml:"platform"`
}
type imageCompose struct {
	Services map[string]imageService `yaml:"services"`
}

func verifyServiceImages(composePath, lockPath, baselinePath string, forbidMajorChange bool) error {
	lock, err := readLock(lockPath)
	if err != nil {
		return err
	}
	current, err := readImageCompose(composePath)
	if err != nil {
		return err
	}
	for _, service := range []string{"postgres", "redis", "nginx"} {
		if err := matchLockedImage(service, current.Services[service], lock[service+".image"], lock[service+".platform"]); err != nil {
			return err
		}
	}
	root := filepath.Dir(composePath)
	ci, err := readImageCompose(filepath.Join(root, "docker-compose.ci.yml"))
	if err != nil {
		return err
	}
	for _, service := range []string{"migrate-web", "migrate-integration", "migrate-endpoint", "migrate-assistant"} {
		if err := matchLockedImage(service, ci.Services[service], lock["migrate.image"], lock["migrate.platform"]); err != nil {
			return err
		}
	}
	if ci.Services["test-runner"].Platform != lock["test-runner.platform"] {
		return errors.New("test-runner platform does not match lock")
	}
	if err := verifyDockerfileImages(filepath.Join(root, "tests/system/test-runner/Dockerfile"), lock); err != nil {
		return err
	}
	if forbidMajorChange && baselinePath != "" {
		baseline, err := readImageCompose(baselinePath)
		if err != nil {
			return err
		}
		for _, service := range []string{"postgres", "redis", "nginx"} {
			if stripDigest(current.Services[service].Image) != stripDigest(baseline.Services[service].Image) {
				return fmt.Errorf("%s image track changed from %s to %s", service, stripDigest(baseline.Services[service].Image), stripDigest(current.Services[service].Image))
			}
		}
	}
	return nil
}

func readLock(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	result := map[string]string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || key == "" || value == "" {
			return nil, fmt.Errorf("invalid lock line %q", line)
		}
		result[key] = value
	}
	return result, scanner.Err()
}

func readImageCompose(path string) (imageCompose, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return imageCompose{}, err
	}
	var compose imageCompose
	err = yaml.Unmarshal(data, &compose)
	return compose, err
}

func matchLockedImage(service string, actual imageService, image, platform string) error {
	if actual.Image != image {
		return fmt.Errorf("%s image %q does not match lock %q", service, actual.Image, image)
	}
	if actual.Platform != platform {
		return fmt.Errorf("%s platform %q does not match lock %q", service, actual.Platform, platform)
	}
	if !regexp.MustCompile(`@sha256:[0-9a-f]{64}$`).MatchString(actual.Image) {
		return fmt.Errorf("%s image is not digest qualified", service)
	}
	return nil
}

func verifyDockerfileImages(path string, lock map[string]string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	pattern := regexp.MustCompile(`(?m)^FROM\s+(\S+)`)
	matches := pattern.FindAllStringSubmatch(string(data), -1)
	if len(matches) != 2 {
		return fmt.Errorf("expected two test-runner FROM images, got %d", len(matches))
	}
	if matches[0][1] != lock["test-runner-builder.image"] || matches[1][1] != lock["test-runner-runtime.image"] {
		return errors.New("test-runner Dockerfile images do not match lock")
	}
	return nil
}

func stripDigest(image string) string { before, _, _ := strings.Cut(image, "@"); return before }
