// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

//go:build !cgo

package internal_livekit

import (
	"os"
	"path/filepath"
	"runtime"
)

type TurnDetectorConfig struct {
	ModelPath     string
	TokenizerPath string
	ModelType     string
}

type TurnDetector struct{}

func NewTurnDetector(TurnDetectorConfig) (*TurnDetector, error) {
	return nil, errTurnDetectorRuntimeAPIUnavailable
}

func (td *TurnDetector) Predict(string) (float64, error) {
	return 0, errTurnDetectorRuntimeAPIUnavailable
}

func (td *TurnDetector) Destroy() {}

func resolveModelPath(configured string, multilingual bool) string {
	if configured != "" {
		return configured
	}
	if multilingual {
		if envPath := os.Getenv(envModelMultiPathKey); envPath != "" {
			return envPath
		}
		_, currentFile, _, _ := runtime.Caller(0)
		return filepath.Join(filepath.Dir(currentFile), defaultModelFileMulti)
	}
	if envPath := os.Getenv(envModelPathKey); envPath != "" {
		return envPath
	}
	_, currentFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(currentFile), defaultModelFileEn)
}

func resolveTokenizerPath(configured string) string {
	if configured != "" {
		return configured
	}
	if envPath := os.Getenv(envTokenizerPathKey); envPath != "" {
		return envPath
	}
	_, currentFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(currentFile), defaultTokenizerFile)
}
