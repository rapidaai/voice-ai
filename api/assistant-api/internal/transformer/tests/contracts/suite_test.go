// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package contracts_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/allure-framework/allure-go/commons/model"
	"github.com/allure-framework/allure-go/commons/writer"
	"github.com/google/uuid"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	"github.com/onsi/gomega"
)

func TestTransformerContracts(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	suiteConfig, reporterConfig := ginkgo.GinkgoConfiguration()
	suiteConfig.FailOnEmpty = true
	suiteConfig.FailOnPending = true
	if directory := os.Getenv("TRANSFORMER_REPORT_DIR"); directory != "" {
		if err := os.MkdirAll(directory, 0700); err != nil {
			t.Fatalf("create transformer report directory: %v", err)
		}
		entries, err := os.ReadDir(directory)
		if err != nil || len(entries) != 0 {
			t.Fatalf("transformer report directory must be empty: %s (read error: %v)", directory, err)
		}
		reporterConfig.JSONReport = filepath.Join(directory, "ginkgo.json")
		reporterConfig.JUnitReport = filepath.Join(directory, "junit.xml")
	}
	ginkgo.RunSpecs(t, "Transformer Contracts", suiteConfig, reporterConfig)
}

var _ = ginkgo.ReportAfterSuite("write Allure results", func(report ginkgo.Report) {
	directory := os.Getenv("TRANSFORMER_REPORT_DIR")
	if directory == "" {
		return
	}
	// Reporting must survive a canceled spec, but still has its own bounded lifetime.
	reportContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resultWriter := writer.NewFileSystemWriter(filepath.Join(directory, "allure-results"))
	gomega.Expect(resultWriter.WriteEnvironmentInfo(reportContext, map[string]string{
		"go.version": runtime.Version(), "platform": runtime.GOOS + "/" + runtime.GOARCH,
		"ginkgo.seed": strconv.FormatInt(report.SuiteConfig.RandomSeed, 10),
		"commit":      os.Getenv("GITHUB_SHA"), "mode": "offline",
	})).To(gomega.Succeed(), "write Allure environment")

	for _, spec := range report.SpecReports {
		if spec.LeafNodeType != types.NodeTypeIt {
			continue
		}
		identity := fmt.Sprintf("%x", sha256.Sum256([]byte(report.SuiteDescription+"\n"+spec.FullText())))
		result := model.TestResult{
			UUID: uuid.NewString(), Name: spec.LeafNodeText, FullName: spec.FullText(),
			TestCaseID: identity, HistoryID: identity, Stage: model.StageFinished,
			TitlePath: spec.ContainerHierarchyTexts,
			Labels: []model.Label{
				{Name: "parentSuite", Value: "transformer"},
				{Name: "suite", Value: report.SuiteDescription},
				{Name: "framework", Value: "ginkgo"},
			},
		}
		if !spec.StartTime.IsZero() {
			result.Start = spec.StartTime.UnixMilli()
			result.Stop = spec.EndTime.UnixMilli()
		}
		switch spec.State {
		case types.SpecStatePassed:
			result.Status = model.StatusPassed
		case types.SpecStateFailed, types.SpecStateTimedout:
			result.Status = model.StatusFailed
		case types.SpecStateSkipped, types.SpecStatePending:
			result.Status = model.StatusSkipped
		default:
			result.Status = model.StatusBroken
		}
		if spec.Failure.Message != "" {
			result.StatusDetails = &model.StatusDetails{
				Message: spec.Failure.Message,
				Trace:   spec.Failure.Location.String() + "\n" + spec.Failure.Location.FullStackTrace,
			}
		}
		for _, label := range spec.Labels() {
			result.Labels = append(result.Labels, model.Label{Name: "tag", Value: label})
		}
		for _, event := range spec.Timeline() {
			switch event := event.(type) {
			case types.SpecEvent:
				if event.SpecEventType != types.SpecEventByStart {
					continue
				}
				if len(result.Steps) > 0 {
					result.Steps[len(result.Steps)-1].Stop = event.TimelineLocation.Time.UnixMilli()
				}
				result.Steps = append(result.Steps, model.StepResult{
					Name: event.Message, Status: model.StatusPassed, Stage: model.StageFinished,
					Start: event.TimelineLocation.Time.UnixMilli(), Stop: result.Stop,
				})
			case types.Failure:
				if len(result.Steps) > 0 {
					result.Steps[len(result.Steps)-1].Status = result.Status
					result.Steps[len(result.Steps)-1].StatusDetails = result.StatusDetails
				}
			case types.AdditionalFailure:
				if len(result.Steps) > 0 {
					result.Steps[len(result.Steps)-1].Status = model.StatusFailed
					result.Steps[len(result.Steps)-1].StatusDetails = &model.StatusDetails{
						Message: event.Failure.Message,
						Trace:   event.Failure.Location.String() + "\n" + event.Failure.Location.FullStackTrace,
					}
				}
			}
		}
		for _, entry := range spec.ReportEntries {
			if entry.Name != "packet-timeline" {
				continue
			}
			content, err := json.MarshalIndent(entry.Value.GetRawValue(), "", "  ")
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "encode packet timeline")
			attachment := model.Attachment{
				Name: entry.Name, Type: "application/json", Source: result.UUID + "-packets.json",
			}
			gomega.Expect(resultWriter.WriteAttachment(reportContext, attachment.Source, content)).To(gomega.Succeed(), "write packet timeline")
			result.Attachments = append(result.Attachments, attachment)
		}
		gomega.Expect(resultWriter.WriteResult(reportContext, result)).To(gomega.Succeed(), "write Allure result")
	}
})
