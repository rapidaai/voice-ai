// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

//go:build !cgo

package internal_pipecat

import (
	"os"
	"path/filepath"
	"runtime"
)

type PipecatDetectorConfig struct {
	ModelPath string
}

type PipecatDetector struct{}

func NewPipecatDetector(PipecatDetectorConfig) (*PipecatDetector, error) {
	return nil, errPipecatDetectorRuntimeAPIUnavailable
}

func (pd *PipecatDetector) Predict([]float32) (float64, error) {
	return 0, errPipecatDetectorRuntimeAPIUnavailable
}

func (pd *PipecatDetector) Destroy() {}

func resolvePctModelPath(configured string) string {
	if configured != "" {
		return configured
	}
	if envPath := os.Getenv(envPctModelPathKey); envPath != "" {
		return envPath
	}
	_, currentFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(currentFile), defaultPctModel)
}
