// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package contracts_test

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/allure-framework/allure-go/commons/model"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/reporters"
	"github.com/onsi/ginkgo/v2/types"
	"github.com/onsi/gomega"
)

var _ = func() bool {
	if os.Getenv("TRANSFORMER_REPORT_FIXTURE") != "1" {
		return true
	}
	return ginkgo.Describe("reporting fixture", ginkgo.Label("report-fixture"), ginkgo.Ordered, func() {
		ginkgo.It("records the outcome", func(ctx ginkgo.SpecContext) {
			ginkgo.By("recording safe diagnostic evidence")
			ginkgo.AddReportEntry("packet-timeline", []map[string]any{
				{"kind": "audio", "context_id": "fixture-message", "bytes": 2},
			}, ginkgo.ReportEntryVisibilityNever)
			ginkgo.By("exercising the selected outcome")
			switch os.Getenv("TRANSFORMER_REPORT_OUTCOME") {
			case "cleanup":
				ginkgo.DeferCleanup(func() { ginkgo.By("successful cleanup") })
				gomega.Expect("actual").To(gomega.Equal("expected"), "intentional assertion failure")
			case "failure", "ordered-failure":
				gomega.Expect("actual").To(gomega.Equal("expected"), "intentional assertion failure")
			case "timeout":
				<-ctx.Done()
			case "skip":
				ginkgo.Skip("intentional skip")
			case "panic":
				panic("intentional panic")
			case "write-error":
				gomega.Expect(os.WriteFile(filepath.Join(os.Getenv("TRANSFORMER_REPORT_DIR"), "allure-results"), []byte("not a directory"), 0600)).To(gomega.Succeed())
			}
		}, ginkgo.SpecTimeout(100*time.Millisecond))
		if os.Getenv("TRANSFORMER_REPORT_OUTCOME") == "ordered-failure" {
			ginkgo.It("is skipped after an earlier failure", func() {
				ginkgo.Fail("this spec must not execute")
			})
		}
	})
}()

func TestContractReports(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	var passIdentity string
	for _, scenario := range []struct {
		name      string
		status    model.Status
		state     types.SpecState
		shouldErr bool
	}{
		{name: "pass", status: model.StatusPassed, state: types.SpecStatePassed},
		{name: "repeat", status: model.StatusPassed, state: types.SpecStatePassed},
		{name: "failure", status: model.StatusFailed, state: types.SpecStateFailed, shouldErr: true},
		{name: "cleanup", status: model.StatusFailed, state: types.SpecStateFailed, shouldErr: true},
		{name: "ordered-failure", status: model.StatusFailed, state: types.SpecStateFailed, shouldErr: true},
		{name: "timeout", status: model.StatusFailed, state: types.SpecStateTimedout, shouldErr: true},
		{name: "skip", status: model.StatusSkipped, state: types.SpecStateSkipped},
		{name: "panic", status: model.StatusBroken, state: types.SpecStatePanicked, shouldErr: true},
		{name: "empty", shouldErr: true},
		{name: "write-error", shouldErr: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			directory := t.TempDir()
			ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, executable, "-test.run=^TestTransformerContracts$", "-test.timeout=15s", "-ginkgo.label-filter=report-fixture")
			for _, entry := range os.Environ() {
				if !strings.HasPrefix(entry, "TRANSFORMER_REPORT_") {
					command.Env = append(command.Env, entry)
				}
			}
			command.Env = append(command.Env, "TRANSFORMER_REPORT_FIXTURE=1", "TRANSFORMER_REPORT_OUTCOME="+scenario.name, "TRANSFORMER_REPORT_DIR="+directory)
			if scenario.name == "empty" {
				command.Args[len(command.Args)-1] = "-ginkgo.label-filter=missing-label"
			}
			output, err := command.CombinedOutput()
			if (err != nil) != scenario.shouldErr || ctx.Err() != nil {
				t.Fatalf("unexpected execution result: %v\n%s", err, output)
			}
			content, err := os.ReadFile(filepath.Join(directory, "ginkgo.json"))
			if err != nil {
				t.Fatalf("missing native report: %v\n%s", err, output)
			}
			var nativeReports []types.Report
			if err := json.Unmarshal(content, &nativeReports); err != nil || len(nativeReports) != 1 {
				t.Fatalf("invalid native report: %v", err)
			}
			if nativeReports[0].SuiteSucceeded == scenario.shouldErr {
				t.Fatalf("native suite status does not match process result: %s", content)
			}
			content, err = os.ReadFile(filepath.Join(directory, "junit.xml"))
			if err != nil {
				t.Fatal(err)
			}
			var junit reporters.JUnitTestSuites
			if err := xml.Unmarshal(content, &junit); err != nil || len(junit.TestSuites) == 0 {
				t.Fatalf("invalid JUnit report: %v", err)
			}
			results, err := filepath.Glob(filepath.Join(directory, "allure-results", "*-result.json"))
			if err != nil {
				t.Fatal(err)
			}
			if scenario.name == "write-error" {
				if len(results) != 0 {
					t.Fatalf("unexpected Allure test results: %v", results)
				}
				return
			}
			var result model.TestResult
			for _, path := range results {
				content, err = os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				var candidate model.TestResult
				if err := json.Unmarshal(content, &candidate); err != nil {
					t.Fatal(err)
				}
				if candidate.Name == "records the outcome" {
					result = candidate
				}
				if (candidate.Name != "records the outcome" || scenario.name == "empty") && candidate.Status != model.StatusSkipped {
					t.Fatalf("unexecuted spec must remain skipped: %s", content)
				}
			}
			selectedCount := 0
			for _, spec := range nativeReports[0].SpecReports {
				if spec.LeafNodeType == types.NodeTypeIt {
					selectedCount++
				}
			}
			if len(results) != selectedCount {
				t.Fatalf("Allure lost native outcomes: %d != %d", len(results), selectedCount)
			}
			if scenario.name == "empty" {
				return
			}
			if result.UUID == "" {
				t.Fatalf("missing fixture result\n%s", output)
			}
			if result.Status != scenario.status || result.HistoryID == "" || result.Stop < result.Start {
				t.Fatalf("incorrect Allure outcome: %s", content)
			}
			expectedSteps := 2
			if scenario.name == "cleanup" {
				expectedSteps = 3
			}
			if len(result.Steps) != expectedSteps || result.Steps[0].Status != model.StatusPassed || result.Steps[1].Status != scenario.status {
				t.Fatalf("incorrect step outcomes: %s", content)
			}
			if scenario.name == "cleanup" && result.Steps[2].Status != model.StatusPassed {
				t.Fatal("successful cleanup was attributed the earlier assertion failure")
			}
			if scenario.name == "pass" {
				passIdentity = result.HistoryID
			}
			if scenario.name == "repeat" && result.HistoryID != passIdentity {
				t.Fatalf("history identity changed: %q != %q", result.HistoryID, passIdentity)
			}
			if scenario.status != model.StatusPassed && (result.StatusDetails == nil || result.StatusDetails.Message == "" || result.StatusDetails.Trace == "") {
				t.Fatalf("missing failure or skip details: %s", content)
			}
			if len(result.Attachments) != 1 || result.Attachments[0].Type != "application/json" {
				t.Fatalf("missing packet evidence: %s", content)
			}
			content, err = os.ReadFile(filepath.Join(directory, "allure-results", result.Attachments[0].Source))
			if err != nil || !json.Valid(content) || !strings.Contains(string(content), "fixture-message") {
				t.Fatalf("invalid packet attachment: %v, %s", err, content)
			}
			found := false
			for _, spec := range nativeReports[0].SpecReports {
				if spec.LeafNodeText == "records the outcome" {
					found = true
					if spec.State != scenario.state {
						t.Fatalf("native outcome mismatch: %s", spec.State)
					}
				}
			}
			if !found {
				t.Fatal("native report omitted the fixture")
			}
		})
	}
}
