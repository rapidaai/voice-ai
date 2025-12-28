// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_capturers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rapidaai/pkg/commons"
	gorm_generator "github.com/rapidaai/pkg/models/gorm/generators"
	"github.com/rapidaai/pkg/storages"
	storage_files "github.com/rapidaai/pkg/storages/file-storage"
	type_enums "github.com/rapidaai/pkg/types/enums"
)

type textCapturer struct {
	logger   commons.Logger
	opts     *CapturerOptions
	storage  storages.Storage
	messages []Message
}
type Message struct {
	Timestamp time.Time               `json:"timestamp"`
	Role      type_enums.MessageActor `json:"role"`
	Content   string                  `json:"content"`
}

func NewS3TextCapturer(lg commons.Logger, opts *CapturerOptions) (TextCapturer, error) {
	cfg, err := opts.S3Config(nil)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize s3 audio capturer")
	}
	return &textCapturer{
		logger:  lg,
		opts:    opts,
		storage: storage_files.NewAwsFileStorage(cfg, lg),
	}, nil
}

// Capture implements TextCapturer.
func (t *textCapturer) Capture(ctx context.Context, role type_enums.MessageActor, s string) error {
	message := Message{
		Timestamp: time.Now(),
		Role:      role,
		Content:   s,
	}
	t.messages = append(t.messages, message)
	// t.logger.Infof("Captured message: Timestamp=%s, Role=%s, Content=%s", message.Timestamp, role, s)
	return nil
}
func (t *textCapturer) Name() string {
	return "aws-s3-text-capturer"
}

// Persist implements TextCapturer.
func (t *textCapturer) Persist(ctx context.Context, key string) (*CapturerOutput, error) {
	t.logger.Infof("Persisting %d messages to S3 with key: %s", len(t.messages), key)

	// Convert the messages directly to JSON
	jsonData, err := json.Marshal(t.messages)
	if err != nil {
		t.logger.Errorf("Failed to marshal messages to JSON: %v", err)
		return nil, err
	}
	// Use the storage interface to save the JSON data
	fileName := fmt.Sprintf("%s/%d__%s", key, gorm_generator.ID(), "messages.json")
	storagePath := t.storage.Store(ctx, fileName, jsonData)
	if storagePath.Error != nil {
		return nil, storagePath.Error
	}
	return &CapturerOutput{
		Paths: []string{storagePath.CompletePath},
		Name:  t.Name(),
	}, nil
}
