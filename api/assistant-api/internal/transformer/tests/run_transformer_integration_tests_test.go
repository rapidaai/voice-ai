package tests_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestIntegrationRunner(t *testing.T) {
	for _, test := range []struct {
		name       string
		args       []string
		matches    []string
		rejects    []string
		packages   []string
		exitCode   int
		goExitCode string
		noGo       bool
	}{
		{
			name: "tts_provider_filter", args: []string{"--tts-only", "deepgram"},
			matches:  []string{"TestDeepgramTTSLifecycle", "TestTTSIntegration/deepgram/basic"},
			rejects:  []string{"TestDeepgramSTTSilentAudio", "TestTTSIntegration/cartesia/basic", "TestTTSHTTPContract"},
			packages: []string{"deepgram"},
		},
		{
			name: "stt_aliases", args: []string{"--stt-only", "azure", "google-speech-service", "assembly-ai"},
			matches: []string{"TestAzureSTTLifecycle", "TestGoogleSTTSilentAudio", "TestAssemblyaiSTTLifecycle",
				"TestSTTBasic/azure-speech-service", "TestSTTBasic/google-speech-service", "TestSTTBasic/assemblyai"},
			rejects:  []string{"TestAzureTTSLifecycle", "TestTTSIntegration/azure-speech-service", "TestSTTBasic/deepgram"},
			packages: []string{"azure", "google", "assembly-ai"},
		},
		{
			name: "both_directions", args: []string{"cartesia"},
			matches:  []string{"TestCartesiaTTSLifecycle", "TestCartesiaSTTLifecycle", "TestTTSIntegration/cartesia", "TestSTTBasic/cartesia"},
			rejects:  []string{"TestTTSIntegration/cartesia-other", "TestSTTBasic/deepgram"},
			packages: []string{"cartesia"},
		},
		{
			name: "shared_only_provider", args: []string{"--tts-only", "resemble"},
			matches: []string{"TestTTSIntegration/resembleai"},
			rejects: []string{"TestTTSIntegration/deepgram", "TestSTTBasic/resembleai"},
		},
		{
			name: "all_providers",
			matches: []string{"TestSmallestTTSLifecycle", "TestSmallestSTTLifecycle", "TestSpeechmaticsSTTLifecycle",
				"TestAssemblyaiSTTLifecycle", "TestTTSIntegration/aws", "TestSTTBasic/revai", "TestTTSIntegration/custom-tts"},
			rejects: []string{"TestTTSHTTPContract", "TestTTSIntegration/unknown"},
			packages: []string{"deepgram", "google", "sarvam", "elevenlabs", "cartesia", "assembly-ai", "azure", "rime",
				"speechmatics", "smallest"},
		},
		{name: "go_failure", args: []string{"deepgram"}, packages: []string{"deepgram"}, goExitCode: "7", exitCode: 7},
		{name: "unsupported_direction", args: []string{"--tts-only", "assemblyai"}, noGo: true},
		{name: "unknown_provider", args: []string{"unknown"}, exitCode: 2, noGo: true},
		{name: "unknown_flag", args: []string{"--unknown"}, exitCode: 2, noGo: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			fakeBin := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(fakeBin, "go"), []byte(
				"#!/bin/sh\nprintf '%s\\n' \"$@\"\nprintf '%s\\n' '--- SKIP: TestDisabled (0.00s)'\nexit \"${TEST_GO_EXIT_CODE:-0}\"\n"), 0o755))
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			runner, err := filepath.Abs("../../../../../bin/run-transformer-integration-tests.sh")
			require.NoError(t, err)
			command := exec.CommandContext(ctx, "bash", append([]string{runner}, test.args...)...)
			command.Env = append(os.Environ(), "PATH="+fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"),
				"TEST_GO_EXIT_CODE="+test.goExitCode)
			output, err := command.CombinedOutput()
			if test.exitCode == 0 {
				require.NoError(t, err, "%s", output)
			} else {
				require.Error(t, err)
			}
			require.Equal(t, test.exitCode, command.ProcessState.ExitCode(), "%s", output)
			arguments := strings.Split(strings.TrimSpace(string(output)), "\n")
			filterIndex := slices.Index(arguments, "-run")
			if test.noGo {
				require.Equal(t, -1, filterIndex, "%s", output)
				return
			}
			require.GreaterOrEqual(t, filterIndex, 0, "%s", output)
			require.Contains(t, arguments, "-v")
			require.Contains(t, string(output), "--- SKIP: TestDisabled")
			require.NotContains(t, string(output), "PASS:", "the runner must not invent a pass summary")
			filters := strings.Split(arguments[filterIndex+1], "/")
			require.Len(t, filters, 2)
			for expected, names := range map[bool][]string{true: test.matches, false: test.rejects} {
				for _, name := range names {
					matches := true
					for index, part := range strings.Split(name, "/") {
						if index == len(filters) {
							break
						}
						matches = matches && regexp.MustCompile(filters[index]).MatchString(part)
					}
					require.Equal(t, expected, matches, "test selection for %s", name)
				}
			}
			packages := []string{"./api/assistant-api/internal/transformer/tests/integration"}
			for _, provider := range test.packages {
				packages = append(packages, packages[0]+"/"+provider)
			}
			require.Equal(t, packages, arguments[filterIndex+2:len(arguments)-1])
		})
	}
}
