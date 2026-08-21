// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_assistant_service

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	observability "github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type assistantDashboardSQLFilters struct {
	conversationConditions []string
	conversationArguments  []interface{}
	messageConditions      []string
	messageArguments       []interface{}
	rangeStart             time.Time
	rangeEnd               time.Time
}

func (assistantService *assistantService) GetAssistantDashboard(
	ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64,
	fromDate *timestamppb.Timestamp,
	toDate *timestamppb.Timestamp,
) (*protos.AssistantDashboard, error) {
	queryFilters, err := assistantService.buildAssistantDashboardSQLFilters(auth, assistantId, fromDate, toDate)
	if err != nil {
		return nil, err
	}

	sessionSummary, sttDurationMs, ttsDurationMs, err := assistantService.getAssistantDashboardSessionSummary(ctx, queryFilters)
	if err != nil {
		return nil, err
	}

	latencySummary, totalMessages, userMessages, totalTokens, err := assistantService.getAssistantDashboardMessageSummary(ctx, queryFilters)
	if err != nil {
		return nil, err
	}

	totalDurationSeconds, err := assistantService.getAssistantDashboardDurationSummary(ctx, queryFilters)
	if err != nil {
		return nil, err
	}

	sourceDistribution, err := assistantService.getAssistantDashboardSourceDistribution(ctx, queryFilters, totalMessages)
	if err != nil {
		return nil, err
	}

	languageDistribution, err := assistantService.getAssistantDashboardLanguageDistribution(ctx, queryFilters, userMessages)
	if err != nil {
		return nil, err
	}

	timeBuckets, err := assistantService.getAssistantDashboardBuckets(ctx, queryFilters)
	if err != nil {
		return nil, err
	}

	sessionSummary.TotalMessages = totalMessages
	sessionSummary.UserMessages = userMessages
	if sessionSummary.TotalSessions > sessionSummary.CompletedSessions+sessionSummary.FailedSessions {
		sessionSummary.ActiveSessions = sessionSummary.TotalSessions - sessionSummary.CompletedSessions - sessionSummary.FailedSessions
	}

	if sessionSummary.TotalSessions > 0 {
		sessionSummary.FailureRate = (float64(sessionSummary.FailedSessions) / float64(sessionSummary.TotalSessions)) * 100
		sessionSummary.AverageSessionDurationSeconds = totalDurationSeconds / float64(sessionSummary.TotalSessions)
	}

	averageLatencyValues := make([]float64, 0, 4)
	for _, averageLatencyValue := range []float64{
		latencySummary.SttMs,
		latencySummary.EosMs,
		latencySummary.TtsMs,
		latencySummary.LlmMs,
	} {
		if averageLatencyValue > 0 {
			averageLatencyValues = append(averageLatencyValues, averageLatencyValue)
		}
	}

	averageLatencyMs := float64(0)
	for _, averageLatencyValue := range averageLatencyValues {
		averageLatencyMs += averageLatencyValue
	}
	if len(averageLatencyValues) > 0 {
		averageLatencyMs = averageLatencyMs / float64(len(averageLatencyValues))
	}
	latencySummary.AverageMs = averageLatencyMs

	return &protos.AssistantDashboard{
		Summary: sessionSummary,
		Latency: latencySummary,
		Usage: &protos.AssistantDashboardUsage{
			TotalTokens:          totalTokens,
			SttDurationSeconds:   sttDurationMs / 1000,
			TtsDurationSeconds:   ttsDurationMs / 1000,
			TotalDurationSeconds: totalDurationSeconds,
		},
		Sources:   sourceDistribution,
		Languages: languageDistribution,
		Buckets:   timeBuckets,
	}, nil
}

func (assistantService *assistantService) getAssistantDashboardSessionSummary(
	ctx context.Context,
	queryFilters *assistantDashboardSQLFilters,
) (*protos.AssistantDashboardSummary, float64, float64, error) {
	database := assistantService.postgres.DB(ctx)
	conversationWhereClause := strings.Join(queryFilters.conversationConditions, " AND ")

	sessionSummaryQuery := fmt.Sprintf(`
SELECT
	COUNT(*) AS total_sessions,
	COALESCE(SUM(CASE WHEN UPPER(COALESCE(conversation_status_metric.value, 'ACTIVE')) IN ('COMPLETE', 'COMPLETED') THEN 1 ELSE 0 END), 0) AS completed_sessions,
	COALESCE(SUM(CASE WHEN UPPER(COALESCE(conversation_status_metric.value, '')) IN ('FAILED', 'ERROR') THEN 1 ELSE 0 END), 0) AS failed_sessions,
	COALESCE(SUM(conversation_stt_duration_metric.value::double precision), 0) AS stt_duration_ms,
	COALESCE(SUM(conversation_tts_duration_metric.value::double precision), 0) AS tts_duration_ms
FROM assistant_conversations
LEFT JOIN assistant_conversation_metrics conversation_status_metric
	ON conversation_status_metric.assistant_conversation_id = assistant_conversations.id
	AND conversation_status_metric.name = ?
LEFT JOIN assistant_conversation_metrics conversation_stt_duration_metric
	ON conversation_stt_duration_metric.assistant_conversation_id = assistant_conversations.id
	AND conversation_stt_duration_metric.name = ?
LEFT JOIN assistant_conversation_metrics conversation_tts_duration_metric
	ON conversation_tts_duration_metric.assistant_conversation_id = assistant_conversations.id
	AND conversation_tts_duration_metric.name = ?
WHERE %s
`, conversationWhereClause)

	sessionSummaryArguments := make([]interface{}, 0, len(queryFilters.conversationArguments)+3)
	sessionSummaryArguments = append(sessionSummaryArguments,
		observability.MetricConversationStatus,
		observability.MetricConversationSTTDuration,
		observability.MetricConversationTTSDuration,
	)
	sessionSummaryArguments = append(sessionSummaryArguments, queryFilters.conversationArguments...)

	sessionSummaryRow := struct {
		TotalSessions     uint32
		CompletedSessions uint32
		FailedSessions    uint32
		SttDurationMs     float64
		TtsDurationMs     float64
	}{}
	if err := database.Raw(sessionSummaryQuery, sessionSummaryArguments...).Scan(&sessionSummaryRow).Error; err != nil {
		assistantService.logger.Errorf("unable to get assistant dashboard session summary %v", err)
		return nil, 0, 0, err
	}
	return &protos.AssistantDashboardSummary{
		TotalSessions:     sessionSummaryRow.TotalSessions,
		CompletedSessions: sessionSummaryRow.CompletedSessions,
		FailedSessions:    sessionSummaryRow.FailedSessions,
	}, sessionSummaryRow.SttDurationMs, sessionSummaryRow.TtsDurationMs, nil
}

func (assistantService *assistantService) getAssistantDashboardMessageSummary(
	ctx context.Context,
	queryFilters *assistantDashboardSQLFilters,
) (*protos.AssistantDashboardLatency, uint32, uint32, float64, error) {
	database := assistantService.postgres.DB(ctx)
	messageWhereClause := strings.Join(queryFilters.messageConditions, " AND ")

	messageSummaryQuery := fmt.Sprintf(`
SELECT
	COUNT(*) AS total_messages,
	COALESCE(SUM(CASE WHEN LOWER(COALESCE(assistant_conversation_messages.role, '')) = 'user' OR assistant_conversation_messages.message_id LIKE 'user-%%' THEN 1 ELSE 0 END), 0) AS user_messages,
	COALESCE(AVG(message_metric_values.stt_latency_value), 0) AS average_stt_ms,
	COALESCE(AVG(message_metric_values.eos_latency_value), 0) AS average_eos_ms,
	COALESCE(AVG(message_metric_values.tts_latency_value), 0) AS average_tts_ms,
	COALESCE(AVG(message_metric_values.llm_latency_value), 0) AS average_llm_ms,
	COALESCE(SUM(message_metric_values.total_token_value), 0) AS total_tokens
FROM assistant_conversation_messages
JOIN assistant_conversations
	ON assistant_conversations.id = assistant_conversation_messages.assistant_conversation_id
LEFT JOIN (
	SELECT
		assistant_conversation_message_id,
		MAX(CASE WHEN name = ? THEN value::double precision END) AS stt_latency_value,
		MAX(CASE WHEN name = ? THEN value::double precision END) AS eos_latency_value,
		MAX(CASE WHEN name = ? THEN value::double precision END) AS tts_latency_value,
		MAX(CASE WHEN name = ? THEN value::double precision END) AS llm_latency_value,
		MAX(CASE WHEN name = ? THEN value::double precision END) AS total_token_value
	FROM assistant_conversation_message_metrics
	WHERE name IN (?, ?, ?, ?, ?)
	GROUP BY assistant_conversation_message_id
) message_metric_values
	ON message_metric_values.assistant_conversation_message_id = assistant_conversation_messages.message_id
WHERE %s
`, messageWhereClause)

	messageSummaryArguments := make([]interface{}, 0, len(queryFilters.messageArguments)+10)
	messageSummaryArguments = append(messageSummaryArguments,
		observability.MetricSTTLatencyMs,
		observability.MetricEOSLatencyMs,
		observability.MetricTTSLatencyMs,
		observability.MetricAgentLatencyMs,
		observability.MetricAgentTotalToken,
		observability.MetricSTTLatencyMs,
		observability.MetricEOSLatencyMs,
		observability.MetricTTSLatencyMs,
		observability.MetricAgentLatencyMs,
		observability.MetricAgentTotalToken,
	)
	messageSummaryArguments = append(messageSummaryArguments, queryFilters.messageArguments...)

	messageSummaryRow := struct {
		TotalMessages uint32
		UserMessages  uint32
		AverageSttMs  float64
		AverageEosMs  float64
		AverageTtsMs  float64
		AverageLlmMs  float64
		TotalTokens   float64
	}{}
	if err := database.Raw(messageSummaryQuery, messageSummaryArguments...).Scan(&messageSummaryRow).Error; err != nil {
		assistantService.logger.Errorf("unable to get assistant dashboard message summary %v", err)
		return nil, 0, 0, 0, err
	}
	return &protos.AssistantDashboardLatency{
		SttMs: messageSummaryRow.AverageSttMs,
		EosMs: messageSummaryRow.AverageEosMs,
		TtsMs: messageSummaryRow.AverageTtsMs,
		LlmMs: messageSummaryRow.AverageLlmMs,
	}, messageSummaryRow.TotalMessages, messageSummaryRow.UserMessages, messageSummaryRow.TotalTokens, nil
}

func (assistantService *assistantService) getAssistantDashboardDurationSummary(
	ctx context.Context,
	queryFilters *assistantDashboardSQLFilters,
) (float64, error) {
	database := assistantService.postgres.DB(ctx)
	messageWhereClause := strings.Join(queryFilters.messageConditions, " AND ")

	durationSummaryQuery := fmt.Sprintf(`
SELECT COALESCE(SUM(conversation_message_duration.duration_seconds), 0) AS total_duration_seconds
FROM (
	SELECT
		assistant_conversation_messages.assistant_conversation_id,
		EXTRACT(EPOCH FROM (MAX(assistant_conversation_messages.created_date) - MIN(assistant_conversation_messages.created_date))) AS duration_seconds
	FROM assistant_conversation_messages
	JOIN assistant_conversations
		ON assistant_conversations.id = assistant_conversation_messages.assistant_conversation_id
	WHERE %s
	GROUP BY assistant_conversation_messages.assistant_conversation_id
) conversation_message_duration
`, messageWhereClause)

	durationSummaryRow := struct {
		TotalDurationSeconds float64
	}{}
	if err := database.Raw(durationSummaryQuery, queryFilters.messageArguments...).Scan(&durationSummaryRow).Error; err != nil {
		assistantService.logger.Errorf("unable to get assistant dashboard duration summary %v", err)
		return 0, err
	}
	return durationSummaryRow.TotalDurationSeconds, nil
}

func (assistantService *assistantService) getAssistantDashboardSourceDistribution(
	ctx context.Context,
	queryFilters *assistantDashboardSQLFilters,
	totalMessages uint32,
) ([]*protos.AssistantDashboardDistribution, error) {
	database := assistantService.postgres.DB(ctx)
	messageWhereClause := strings.Join(queryFilters.messageConditions, " AND ")

	sourceDistributionQuery := fmt.Sprintf(`
SELECT COALESCE(assistant_conversation_messages.source, 'unknown') AS name, COUNT(*) AS count
FROM assistant_conversation_messages
JOIN assistant_conversations
	ON assistant_conversations.id = assistant_conversation_messages.assistant_conversation_id
WHERE %s
GROUP BY COALESCE(assistant_conversation_messages.source, 'unknown')
ORDER BY count DESC
`, messageWhereClause)

	sourceDistributionRows := []struct {
		Name  string
		Count uint32
	}{}
	if err := database.Raw(sourceDistributionQuery, queryFilters.messageArguments...).Scan(&sourceDistributionRows).Error; err != nil {
		assistantService.logger.Errorf("unable to get assistant dashboard source distribution %v", err)
		return nil, err
	}

	sourceDistribution := make([]*protos.AssistantDashboardDistribution, 0, len(sourceDistributionRows))
	for _, sourceDistributionRow := range sourceDistributionRows {
		percentage := float64(0)
		if totalMessages > 0 {
			percentage = (float64(sourceDistributionRow.Count) / float64(totalMessages)) * 100
		}
		sourceDistribution = append(sourceDistribution, &protos.AssistantDashboardDistribution{
			Name:       sourceDistributionRow.Name,
			Count:      sourceDistributionRow.Count,
			Percentage: percentage,
		})
	}

	return sourceDistribution, nil
}

func (assistantService *assistantService) getAssistantDashboardLanguageDistribution(
	ctx context.Context,
	queryFilters *assistantDashboardSQLFilters,
	userMessages uint32,
) ([]*protos.AssistantDashboardDistribution, error) {
	database := assistantService.postgres.DB(ctx)
	messageWhereClause := strings.Join(queryFilters.messageConditions, " AND ")

	languageDistributionQuery := fmt.Sprintf(`
SELECT assistant_conversation_message_metadata.value AS name, COUNT(*) AS count
FROM assistant_conversation_messages
JOIN assistant_conversations
	ON assistant_conversations.id = assistant_conversation_messages.assistant_conversation_id
JOIN assistant_conversation_message_metadata
	ON assistant_conversation_message_metadata.assistant_conversation_message_id = assistant_conversation_messages.message_id
	AND assistant_conversation_message_metadata.key = ?
WHERE %s
	AND (LOWER(COALESCE(assistant_conversation_messages.role, '')) = 'user' OR assistant_conversation_messages.message_id LIKE 'user-%%')
GROUP BY assistant_conversation_message_metadata.value
ORDER BY count DESC
`, messageWhereClause)

	languageDistributionArguments := make([]interface{}, 0, len(queryFilters.messageArguments)+1)
	languageDistributionArguments = append(languageDistributionArguments, observability.MetadataLanguage)
	languageDistributionArguments = append(languageDistributionArguments, queryFilters.messageArguments...)

	languageDistributionRows := []struct {
		Name  string
		Count uint32
	}{}
	if err := database.Raw(languageDistributionQuery, languageDistributionArguments...).Scan(&languageDistributionRows).Error; err != nil {
		assistantService.logger.Errorf("unable to get assistant dashboard language distribution %v", err)
		return nil, err
	}

	languageDistribution := make([]*protos.AssistantDashboardDistribution, 0, len(languageDistributionRows))
	for _, languageDistributionRow := range languageDistributionRows {
		percentage := float64(0)
		if userMessages > 0 {
			percentage = (float64(languageDistributionRow.Count) / float64(userMessages)) * 100
		}
		languageDistribution = append(languageDistribution, &protos.AssistantDashboardDistribution{
			Name:       languageDistributionRow.Name,
			Count:      languageDistributionRow.Count,
			Percentage: percentage,
		})
	}

	return languageDistribution, nil
}

func (assistantService *assistantService) getAssistantDashboardBuckets(
	ctx context.Context,
	queryFilters *assistantDashboardSQLFilters,
) ([]*protos.AssistantDashboardBucket, error) {
	database := assistantService.postgres.DB(ctx)
	messageWhereClause := strings.Join(queryFilters.messageConditions, " AND ")
	bucketInterval := getAssistantDashboardBucketInterval(queryFilters.rangeStart, queryFilters.rangeEnd)
	bucketIntervalSeconds := int64(bucketInterval.Seconds())

	bucketQueryArguments := make([]interface{}, 0, len(queryFilters.messageArguments)+10)
	bucketQueryArguments = append(bucketQueryArguments, queryFilters.rangeStart)
	bucketQueryArguments = append(bucketQueryArguments, bucketIntervalSeconds)
	bucketQueryArguments = append(bucketQueryArguments,
		observability.MetricSTTLatencyMs,
		observability.MetricEOSLatencyMs,
		observability.MetricTTSLatencyMs,
		observability.MetricAgentLatencyMs,
		observability.MetricSTTLatencyMs,
		observability.MetricEOSLatencyMs,
		observability.MetricTTSLatencyMs,
		observability.MetricAgentLatencyMs,
	)
	bucketQueryArguments = append(bucketQueryArguments, queryFilters.messageArguments...)

	bucketQuery := fmt.Sprintf(`
SELECT
	FLOOR(EXTRACT(EPOCH FROM (assistant_conversation_messages.created_date - CAST(? AS timestamp))) / ?)::bigint AS bucket_index,
	COUNT(*) AS message_count,
	COALESCE(AVG(message_metric_values.stt_latency_value), 0) AS average_stt_ms,
	COALESCE(AVG(message_metric_values.eos_latency_value), 0) AS average_eos_ms,
	COALESCE(AVG(message_metric_values.tts_latency_value), 0) AS average_tts_ms,
	COALESCE(AVG(message_metric_values.llm_latency_value), 0) AS average_llm_ms
FROM assistant_conversation_messages
JOIN assistant_conversations
	ON assistant_conversations.id = assistant_conversation_messages.assistant_conversation_id
LEFT JOIN (
	SELECT
		assistant_conversation_message_id,
		MAX(CASE WHEN name = ? THEN value::double precision END) AS stt_latency_value,
		MAX(CASE WHEN name = ? THEN value::double precision END) AS eos_latency_value,
		MAX(CASE WHEN name = ? THEN value::double precision END) AS tts_latency_value,
		MAX(CASE WHEN name = ? THEN value::double precision END) AS llm_latency_value
	FROM assistant_conversation_message_metrics
	WHERE name IN (?, ?, ?, ?)
	GROUP BY assistant_conversation_message_id
) message_metric_values
	ON message_metric_values.assistant_conversation_message_id = assistant_conversation_messages.message_id
WHERE %s
GROUP BY bucket_index
ORDER BY bucket_index ASC
`, messageWhereClause)

	bucketRows := []struct {
		BucketIndex  int64
		MessageCount uint32
		AverageSttMs float64
		AverageEosMs float64
		AverageTtsMs float64
		AverageLlmMs float64
	}{}
	if err := database.Raw(bucketQuery, bucketQueryArguments...).Scan(&bucketRows).Error; err != nil {
		assistantService.logger.Errorf("unable to get assistant dashboard buckets %v", err)
		return nil, err
	}

	bucketRowsByIndex := make(map[int64]*protos.AssistantDashboardBucket, len(bucketRows))
	for _, bucketRow := range bucketRows {
		bucketRowsByIndex[bucketRow.BucketIndex] = &protos.AssistantDashboardBucket{
			MessageCount: bucketRow.MessageCount,
			SttLatencyMs: bucketRow.AverageSttMs,
			EosLatencyMs: bucketRow.AverageEosMs,
			TtsLatencyMs: bucketRow.AverageTtsMs,
			LlmLatencyMs: bucketRow.AverageLlmMs,
		}
	}

	bucketCount := int(math.Ceil(queryFilters.rangeEnd.Sub(queryFilters.rangeStart).Seconds() / bucketInterval.Seconds()))
	if bucketCount < 1 {
		bucketCount = 1
	}
	if bucketCount > 5000 {
		return nil, fmt.Errorf("dashboard date range creates too many buckets")
	}

	dashboardBuckets := make([]*protos.AssistantDashboardBucket, 0, bucketCount)
	for bucketIndex := 0; bucketIndex < bucketCount; bucketIndex++ {
		bucketStartDate := queryFilters.rangeStart.Add(time.Duration(bucketIndex) * bucketInterval)
		bucketEndDate := bucketStartDate.Add(bucketInterval)
		if bucketEndDate.After(queryFilters.rangeEnd) {
			bucketEndDate = queryFilters.rangeEnd
		}

		dashboardBucket := &protos.AssistantDashboardBucket{}
		if bucketRow := bucketRowsByIndex[int64(bucketIndex)]; bucketRow != nil {
			dashboardBucket = bucketRow
		}
		dashboardBucket.StartDate = timestamppb.New(bucketStartDate)
		dashboardBucket.EndDate = timestamppb.New(bucketEndDate)
		dashboardBuckets = append(dashboardBuckets, &protos.AssistantDashboardBucket{
			StartDate:    dashboardBucket.StartDate,
			EndDate:      dashboardBucket.EndDate,
			MessageCount: dashboardBucket.MessageCount,
			SttLatencyMs: dashboardBucket.SttLatencyMs,
			EosLatencyMs: dashboardBucket.EosLatencyMs,
			TtsLatencyMs: dashboardBucket.TtsLatencyMs,
			LlmLatencyMs: dashboardBucket.LlmLatencyMs,
		})
	}

	return dashboardBuckets, nil
}

func (assistantService *assistantService) buildAssistantDashboardSQLFilters(
	auth types.SimplePrinciple,
	assistantId uint64,
	fromDate *timestamppb.Timestamp,
	toDate *timestamppb.Timestamp,
) (*assistantDashboardSQLFilters, error) {
	projectContext, err := types.RequireProject(auth)
	if err != nil {
		return nil, err
	}

	rangeEnd := time.Now()
	if toDate != nil {
		if err := toDate.CheckValid(); err != nil {
			return nil, fmt.Errorf("invalid dashboard toDate: %w", err)
		}
		rangeEnd = toDate.AsTime()
	}

	rangeStart := rangeEnd.AddDate(0, 0, -30)
	if fromDate != nil {
		if err := fromDate.CheckValid(); err != nil {
			return nil, fmt.Errorf("invalid dashboard fromDate: %w", err)
		}
		rangeStart = fromDate.AsTime()
	}

	if !rangeEnd.After(rangeStart) {
		return nil, fmt.Errorf("dashboard fromDate must be before toDate")
	}

	queryFilters := &assistantDashboardSQLFilters{
		conversationConditions: []string{
			"assistant_conversations.assistant_id = ?",
			"assistant_conversations.organization_id = ?",
			"assistant_conversations.project_id = ?",
			"assistant_conversations.created_date >= ?",
			"assistant_conversations.created_date <= ?",
		},
		conversationArguments: []interface{}{assistantId, projectContext.OrganizationID, projectContext.ProjectID, rangeStart, rangeEnd},
		messageConditions: []string{
			"assistant_conversations.assistant_id = ?",
			"assistant_conversations.organization_id = ?",
			"assistant_conversations.project_id = ?",
			"assistant_conversation_messages.created_date >= ?",
			"assistant_conversation_messages.created_date <= ?",
		},
		messageArguments: []interface{}{assistantId, projectContext.OrganizationID, projectContext.ProjectID, rangeStart, rangeEnd},
		rangeStart:       rangeStart,
		rangeEnd:         rangeEnd,
	}

	return queryFilters, nil
}

func getAssistantDashboardBucketInterval(rangeStart time.Time, rangeEnd time.Time) time.Duration {
	rangeDuration := rangeEnd.Sub(rangeStart)
	switch {
	case rangeDuration <= 24*time.Hour:
		return 30 * time.Minute
	case rangeDuration <= 3*24*time.Hour:
		return 2 * time.Hour
	case rangeDuration <= 7*24*time.Hour:
		return 4 * time.Hour
	default:
		return 24 * time.Hour
	}
}
