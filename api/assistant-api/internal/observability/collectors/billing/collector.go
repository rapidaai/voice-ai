// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package billing

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Publisher interface {
	CreateProductUsages(context.Context, *types.Authentication, []*protos.ProductUsage) (*protos.CreateProductUsagesResponse, error)
	Close() error
}

type Collector struct {
	publisher Publisher
}

func New(publisher Publisher) observability.Collector {
	if !validator.NonNil(publisher) {
		return observability.NoopCollector{}
	}
	return &Collector{publisher: publisher}
}

func (c *Collector) Key() string {
	return "billing"
}

func (c *Collector) Collect(ctx context.Context, _ observability.Scope, observationContext observability.Context, record observability.Record) error {
	usage, ok := record.(observability.RecordUsage)
	if !ok {
		return nil
	}

	usageType := usage.Component.String()
	unit, ok := types.ProductUsageUnitFor(usageType)
	if !ok {
		return types.ValidateProductUsage(usageType, "")
	}
	if err := types.ValidateProductUsage(usageType, unit); err != nil {
		return err
	}
	if usage.Duration <= 0 {
		return fmt.Errorf("product usage %q must be greater than zero", usageType)
	}

	usageID := strings.TrimSpace(usage.ID)
	if usageID == "" {
		usageID = uuid.NewString()
	}
	_, err := c.publisher.CreateProductUsages(ctx, observationContext.Auth, []*protos.ProductUsage{{
		UsageId:    usageID,
		UsageType:  usageType,
		Usages:     usage.Duration.Nanoseconds(),
		Unit:       unit,
		OccurredAt: timestamppb.New(usage.OccurredAt.Truncate(time.Microsecond)),
	}})
	return err
}

func (c *Collector) Close(context.Context) error {
	return c.publisher.Close()
}
