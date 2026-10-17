package internal_dashboard_service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var ErrInvalidProjectDashboardAnalyticsRequest = errors.New("invalid project dashboard analytics request")

const (
	projectDashboardSlowSessionThresholdMilliseconds = uint64(1000)
	projectDashboardMaximumLatencyMilliseconds       = uint64(1<<53 - 1)
	projectDashboardMaximumRange                     = 90 * 24 * time.Hour
)

type projectDashboardAnalyticsService struct {
	postgres connectors.PostgresConnector
}

func NewProjectDashboardAnalyticsService(postgres connectors.PostgresConnector) internal_services.ProjectDashboardAnalyticsService {
	return &projectDashboardAnalyticsService{postgres: postgres}
}

func (dashboardService *projectDashboardAnalyticsService) GetProjectDashboardAnalytics(ctx context.Context, projectContext types.ProjectContext, request *protos.GetProjectDashboardAnalyticsRequest) (*protos.ProjectDashboardAnalytics, error) {
	if request == nil {
		return nil, fmt.Errorf("%w: request is required", ErrInvalidProjectDashboardAnalyticsRequest)
	}
	fromDate := request.GetFromDate()
	toDate := request.GetToDate()
	comparisonFromDate := request.GetComparisonFromDate()
	comparisonToDate := request.GetComparisonToDate()
	if !fromDate.IsValid() || !toDate.IsValid() || !comparisonFromDate.IsValid() || !comparisonToDate.IsValid() {
		return nil, fmt.Errorf("%w: valid range timestamps are required", ErrInvalidProjectDashboardAnalyticsRequest)
	}
	currentFromDate := fromDate.AsTime().UTC()
	currentToDate := toDate.AsTime().UTC()
	previousFromDate := comparisonFromDate.AsTime().UTC()
	previousToDate := comparisonToDate.AsTime().UTC()
	if !currentFromDate.Before(currentToDate) || !previousFromDate.Before(previousToDate) {
		return nil, fmt.Errorf("%w: ranges must have positive duration", ErrInvalidProjectDashboardAnalyticsRequest)
	}
	if currentToDate.After(currentFromDate.Add(projectDashboardMaximumRange)) || previousToDate.After(previousFromDate.Add(projectDashboardMaximumRange)) {
		return nil, fmt.Errorf("%w: date range cannot exceed 90 days", ErrInvalidProjectDashboardAnalyticsRequest)
	}
	if currentToDate.Sub(currentFromDate) != previousToDate.Sub(previousFromDate) {
		return nil, fmt.Errorf("%w: comparison range must have equal duration", ErrInvalidProjectDashboardAnalyticsRequest)
	}
	if !previousToDate.Equal(currentFromDate) {
		return nil, fmt.Errorf("%w: comparison range must end at fromDate", ErrInvalidProjectDashboardAnalyticsRequest)
	}
	if request.GetAssistantLimit() < 1 || request.GetAssistantLimit() > 10 {
		return nil, fmt.Errorf("%w: assistantLimit must be from 1 through 10", ErrInvalidProjectDashboardAnalyticsRequest)
	}
	if request.GetRecentFailureLimit() < 1 || request.GetRecentFailureLimit() > 20 {
		return nil, fmt.Errorf("%w: recentFailureLimit must be from 1 through 20", ErrInvalidProjectDashboardAnalyticsRequest)
	}

	currentSessions, currentFailed, currentLatency, err := dashboardService.getProjectDashboardSummary(ctx, projectContext, currentFromDate, currentToDate)
	if err != nil {
		return nil, err
	}
	previousSessions, previousFailed, previousLatency, err := dashboardService.getProjectDashboardSummary(ctx, projectContext, previousFromDate, previousToDate)
	if err != nil {
		return nil, err
	}
	activity, err := dashboardService.getProjectDashboardActivity(ctx, projectContext, currentFromDate, currentToDate)
	if err != nil {
		return nil, err
	}
	assistants, health, assistantCount, err := dashboardService.getProjectDashboardAssistants(ctx, projectContext, currentFromDate, currentToDate, currentFailed, request.GetAssistantLimit())
	if err != nil {
		return nil, err
	}
	recentFailures, err := dashboardService.getProjectDashboardRecentFailures(ctx, projectContext, currentFromDate, currentToDate, request.GetRecentFailureLimit())
	if err != nil {
		return nil, err
	}

	sessionComparison := &protos.DashboardCountComparison{Value: currentSessions, PreviousValue: previousSessions}
	if previousSessions > 0 {
		change := (float64(currentSessions) - float64(previousSessions)) / float64(previousSessions) * 100
		sessionComparison.ChangePercent = &change
	}
	successRateComparison := &protos.DashboardRateComparison{}
	if currentSessions > 0 {
		value := float64(currentSessions-currentFailed) / float64(currentSessions) * 100
		successRateComparison.ValuePercent = &value
	}
	if previousSessions > 0 {
		previous := float64(previousSessions-previousFailed) / float64(previousSessions) * 100
		successRateComparison.PreviousValuePercent = &previous
		if successRateComparison.ValuePercent != nil {
			change := *successRateComparison.ValuePercent - previous
			successRateComparison.ChangePoints = &change
		}
	}
	latencyComparison := &protos.DashboardLatencyComparison{ValueMs: currentLatency, PreviousValueMs: previousLatency}
	if currentLatency != nil && previousLatency != nil {
		currentLatencyMilliseconds, conversionErr := utils.Uint64ToInt64(*currentLatency)
		if conversionErr != nil {
			return nil, fmt.Errorf("current latency: %w", conversionErr)
		}
		previousLatencyMilliseconds, conversionErr := utils.Uint64ToInt64(*previousLatency)
		if conversionErr != nil {
			return nil, fmt.Errorf("previous latency: %w", conversionErr)
		}
		change := currentLatencyMilliseconds - previousLatencyMilliseconds
		latencyComparison.ChangeMs = &change
	}
	failedComparison := &protos.DashboardFailedComparison{Value: currentFailed, PreviousValue: previousFailed}
	if currentSessions > 0 {
		share := float64(currentFailed) / float64(currentSessions) * 100
		failedComparison.SharePercent = &share
	}

	return &protos.ProjectDashboardAnalytics{
		ProjectState: &protos.DashboardProjectState{
			AssistantCount: assistantCount,
			HasAssistants:  assistantCount > 0,
			HasActivity:    currentSessions > 0,
		},
		Summary: &protos.DashboardSummary{
			Sessions:        sessionComparison,
			SuccessRate:     successRateComparison,
			ResponseLatency: latencyComparison,
			FailedSessions:  failedComparison,
		},
		Activity:       activity,
		Health:         health,
		Assistants:     assistants,
		RecentFailures: recentFailures,
	}, nil
}

func (dashboardService *projectDashboardAnalyticsService) getProjectDashboardSummary(ctx context.Context, project types.ProjectContext, fromDate, toDate time.Time) (uint64, uint64, *uint64, error) {
	database := dashboardService.postgres.DB(ctx)
	latencyValueExpression := fmt.Sprintf(`CASE
		WHEN LENGTH(BTRIM(assistant_conversation_message_metrics.value)) <= 64
			AND BTRIM(assistant_conversation_message_metrics.value) ~ '^([0-9]+([.][0-9]*)|[.][0-9]+)$'
		THEN CASE
			WHEN CAST(BTRIM(assistant_conversation_message_metrics.value) AS DOUBLE PRECISION) > 0
				AND CAST(BTRIM(assistant_conversation_message_metrics.value) AS DOUBLE PRECISION) <= %d
			THEN CAST(BTRIM(assistant_conversation_message_metrics.value) AS DOUBLE PRECISION)
		END
	END`, projectDashboardMaximumLatencyMilliseconds)
	if database.Dialector.Name() == "sqlite" {
		latencyValueExpression = fmt.Sprintf(`CASE
			WHEN LENGTH(TRIM(assistant_conversation_message_metrics.value)) <= 64
				AND json_valid(TRIM(assistant_conversation_message_metrics.value))
			THEN CASE
				WHEN json_type(TRIM(assistant_conversation_message_metrics.value)) IN ('integer', 'real')
					AND INSTR(LOWER(TRIM(assistant_conversation_message_metrics.value)), 'e') = 0
				THEN CASE
					WHEN CAST(TRIM(assistant_conversation_message_metrics.value) AS REAL) > 0
						AND CAST(TRIM(assistant_conversation_message_metrics.value) AS REAL) <= %d
					THEN CAST(TRIM(assistant_conversation_message_metrics.value) AS REAL)
				END
			END
		END`, projectDashboardMaximumLatencyMilliseconds)
	}
	counts := struct {
		Sessions uint64
		Failed   uint64
	}{}
	if err := database.Raw(`
SELECT
	COUNT(*) AS sessions,
	COALESCE(SUM(CASE WHEN conversation_status.status_value IN ('FAILED', 'ERROR') THEN 1 ELSE 0 END), 0) AS failed
FROM assistant_conversations
LEFT JOIN (
	SELECT
		assistant_conversation_metrics.assistant_conversation_id,
		MAX(UPPER(TRIM(assistant_conversation_metrics.value))) AS status_value
	FROM assistant_conversation_metrics
	JOIN assistant_conversations scoped_conversations
		ON scoped_conversations.id = assistant_conversation_metrics.assistant_conversation_id
	WHERE assistant_conversation_metrics.name = ?
		AND scoped_conversations.organization_id = ?
		AND scoped_conversations.project_id = ?
		AND scoped_conversations.created_date >= ?
		AND scoped_conversations.created_date < ?
	GROUP BY assistant_conversation_metrics.assistant_conversation_id
) conversation_status ON conversation_status.assistant_conversation_id = assistant_conversations.id
WHERE assistant_conversations.organization_id = ?
	AND assistant_conversations.project_id = ?
	AND assistant_conversations.created_date >= ?
	AND assistant_conversations.created_date < ?
`, observability.MetricConversationStatus,
		project.OrganizationID, project.ProjectID, fromDate, toDate,
		project.OrganizationID, project.ProjectID, fromDate, toDate,
	).Scan(&counts).Error; err != nil {
		return 0, 0, nil, err
	}

	latency := struct{ Value sql.NullFloat64 }{}
	latencyQuery := fmt.Sprintf(`
SELECT AVG(component_latency.average_value) AS value
FROM (
	SELECT
		assistant_conversation_message_metrics.name,
		AVG(%s) AS average_value
	FROM assistant_conversation_message_metrics
	JOIN assistant_conversation_messages
		ON assistant_conversation_messages.message_id = assistant_conversation_message_metrics.assistant_conversation_message_id
		AND assistant_conversation_messages.assistant_conversation_id = assistant_conversation_message_metrics.assistant_conversation_id
	JOIN assistant_conversations
		ON assistant_conversations.id = assistant_conversation_messages.assistant_conversation_id
	WHERE assistant_conversations.organization_id = ?
		AND assistant_conversations.project_id = ?
		AND assistant_conversations.created_date >= ?
		AND assistant_conversations.created_date < ?
		AND assistant_conversation_message_metrics.name IN (?, ?, ?, ?)
	GROUP BY assistant_conversation_message_metrics.name
) component_latency
`, latencyValueExpression)
	if err := database.Raw(latencyQuery, project.OrganizationID, project.ProjectID, fromDate, toDate,
		observability.MetricSTTLatencyMs,
		observability.MetricEOSLatencyMs,
		observability.MetricTTSLatencyMs,
		observability.MetricAgentTTFTMs,
	).Scan(&latency).Error; err != nil {
		return 0, 0, nil, err
	}
	if !latency.Value.Valid {
		return counts.Sessions, counts.Failed, nil, nil
	}
	latencyMilliseconds := uint64(math.Round(latency.Value.Float64))
	return counts.Sessions, counts.Failed, &latencyMilliseconds, nil
}

func (dashboardService *projectDashboardAnalyticsService) getProjectDashboardActivity(ctx context.Context, project types.ProjectContext, fromDate, toDate time.Time) ([]*protos.DashboardActivityBucket, error) {
	duration := toDate.Sub(fromDate)
	bucketDuration := 24 * time.Hour
	switch {
	case duration <= 24*time.Hour:
		bucketDuration = 30 * time.Minute
	case duration <= 3*24*time.Hour:
		bucketDuration = 2 * time.Hour
	case duration <= 7*24*time.Hour:
		bucketDuration = 4 * time.Hour
	}
	buckets := make([]*protos.DashboardActivityBucket, 0, int(math.Ceil(float64(duration)/float64(bucketDuration))))
	for bucketStart := fromDate; bucketStart.Before(toDate); bucketStart = bucketStart.Add(bucketDuration) {
		bucketEnd := bucketStart.Add(bucketDuration)
		if bucketEnd.After(toDate) {
			bucketEnd = toDate
		}
		buckets = append(buckets, &protos.DashboardActivityBucket{FromDate: timestamppb.New(bucketStart), ToDate: timestamppb.New(bucketEnd)})
	}
	rows := []struct {
		BucketIndex   int
		Sessions      uint64
		FailedSession uint64
	}{}
	bucketExpression := "FLOOR(EXTRACT(EPOCH FROM (assistant_conversations.created_date - CAST(? AS timestamp))) / ?)::bigint"
	if dashboardService.postgres.DB(ctx).Dialector.Name() == "sqlite" {
		bucketExpression = "CAST(((julianday(assistant_conversations.created_date) - julianday(?)) * 86400) / ? AS INTEGER)"
	}
	query := fmt.Sprintf(`
SELECT
	%s AS bucket_index,
	COUNT(*) AS sessions,
	COALESCE(SUM(CASE WHEN conversation_status.status_value IN ('FAILED', 'ERROR') THEN 1 ELSE 0 END), 0) AS failed_session
FROM assistant_conversations
LEFT JOIN (
	SELECT
		assistant_conversation_metrics.assistant_conversation_id,
		MAX(UPPER(TRIM(assistant_conversation_metrics.value))) AS status_value
	FROM assistant_conversation_metrics
	JOIN assistant_conversations scoped_conversations
		ON scoped_conversations.id = assistant_conversation_metrics.assistant_conversation_id
	WHERE assistant_conversation_metrics.name = ?
		AND scoped_conversations.organization_id = ?
		AND scoped_conversations.project_id = ?
		AND scoped_conversations.created_date >= ?
		AND scoped_conversations.created_date < ?
	GROUP BY assistant_conversation_metrics.assistant_conversation_id
) conversation_status ON conversation_status.assistant_conversation_id = assistant_conversations.id
WHERE assistant_conversations.organization_id = ?
	AND assistant_conversations.project_id = ?
	AND assistant_conversations.created_date >= ?
	AND assistant_conversations.created_date < ?
GROUP BY bucket_index
ORDER BY bucket_index
`, bucketExpression)
	if err := dashboardService.postgres.DB(ctx).Raw(query,
		fromDate,
		int64(bucketDuration/time.Second),
		observability.MetricConversationStatus,
		project.OrganizationID,
		project.ProjectID,
		fromDate,
		toDate,
		project.OrganizationID,
		project.ProjectID,
		fromDate,
		toDate,
	).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row.BucketIndex < 0 || row.BucketIndex >= len(buckets) {
			continue
		}
		buckets[row.BucketIndex].Sessions = row.Sessions
		buckets[row.BucketIndex].FailedSessions = row.FailedSession
	}
	return buckets, nil
}

func (dashboardService *projectDashboardAnalyticsService) getProjectDashboardAssistants(ctx context.Context, project types.ProjectContext, fromDate, toDate time.Time, failedSessions uint64, limit uint32) ([]*protos.DashboardAssistant, *protos.DashboardHealth, uint32, error) {
	database := dashboardService.postgres.DB(ctx)
	latencyValueExpression := fmt.Sprintf(`CASE
		WHEN LENGTH(BTRIM(assistant_conversation_message_metrics.value)) <= 64
			AND BTRIM(assistant_conversation_message_metrics.value) ~ '^([0-9]+([.][0-9]*)|[.][0-9]+)$'
		THEN CASE
			WHEN CAST(BTRIM(assistant_conversation_message_metrics.value) AS DOUBLE PRECISION) > 0
				AND CAST(BTRIM(assistant_conversation_message_metrics.value) AS DOUBLE PRECISION) <= %d
			THEN CAST(BTRIM(assistant_conversation_message_metrics.value) AS DOUBLE PRECISION)
		END
	END`, projectDashboardMaximumLatencyMilliseconds)
	durationValueExpression := `CASE
		WHEN LENGTH(BTRIM(assistant_conversation_metrics.value)) <= 18
			AND BTRIM(assistant_conversation_metrics.value) ~ '^[0-9]+$'
		THEN CAST(BTRIM(assistant_conversation_metrics.value) AS BIGINT)
	END`
	if database.Dialector.Name() == "sqlite" {
		latencyValueExpression = fmt.Sprintf(`CASE
			WHEN LENGTH(TRIM(assistant_conversation_message_metrics.value)) <= 64
				AND json_valid(TRIM(assistant_conversation_message_metrics.value))
			THEN CASE
				WHEN json_type(TRIM(assistant_conversation_message_metrics.value)) IN ('integer', 'real')
					AND INSTR(LOWER(TRIM(assistant_conversation_message_metrics.value)), 'e') = 0
				THEN CASE
					WHEN CAST(TRIM(assistant_conversation_message_metrics.value) AS REAL) > 0
						AND CAST(TRIM(assistant_conversation_message_metrics.value) AS REAL) <= %d
					THEN CAST(TRIM(assistant_conversation_message_metrics.value) AS REAL)
				END
			END
		END`, projectDashboardMaximumLatencyMilliseconds)
		durationValueExpression = `CASE
			WHEN LENGTH(TRIM(assistant_conversation_metrics.value)) <= 18
				AND json_valid(TRIM(assistant_conversation_metrics.value))
			THEN CASE
				WHEN json_type(TRIM(assistant_conversation_metrics.value)) = 'integer'
					AND CAST(TRIM(assistant_conversation_metrics.value) AS INTEGER) >= 0
				THEN CAST(TRIM(assistant_conversation_metrics.value) AS INTEGER)
			END
		END`
	}
	rows := []struct {
		AssistantID          uint64
		Name                 string
		Sessions             uint64
		Failed               uint64
		ConversationDuration uint64
		LastActivity         sql.NullString
	}{}
	assistantQuery := fmt.Sprintf(`
SELECT
	assistants.id AS assistant_id,
	assistants.name,
	COUNT(assistant_conversations.id) AS sessions,
	COALESCE(SUM(CASE WHEN conversation_metrics.status_value IN ('FAILED', 'ERROR') THEN 1 ELSE 0 END), 0) AS failed,
	COALESCE(SUM(conversation_metrics.duration_value), 0) AS conversation_duration,
	CAST(MAX(assistant_conversations.created_date) AS TEXT) AS last_activity
FROM assistants
LEFT JOIN assistant_conversations
	ON assistant_conversations.assistant_id = assistants.id
	AND assistant_conversations.organization_id = assistants.organization_id
	AND assistant_conversations.project_id = assistants.project_id
	AND assistant_conversations.created_date >= ?
	AND assistant_conversations.created_date < ?
LEFT JOIN (
	SELECT
		assistant_conversation_metrics.assistant_conversation_id,
		MAX(CASE WHEN assistant_conversation_metrics.name = ? THEN UPPER(TRIM(assistant_conversation_metrics.value)) END) AS status_value,
		MAX(CASE WHEN assistant_conversation_metrics.name = ? THEN %s END) AS duration_value
	FROM assistant_conversation_metrics
	JOIN assistant_conversations scoped_conversations
		ON scoped_conversations.id = assistant_conversation_metrics.assistant_conversation_id
	WHERE assistant_conversation_metrics.name IN (?, ?)
		AND scoped_conversations.organization_id = ?
		AND scoped_conversations.project_id = ?
		AND scoped_conversations.created_date >= ?
		AND scoped_conversations.created_date < ?
	GROUP BY assistant_conversation_metrics.assistant_conversation_id
) conversation_metrics ON conversation_metrics.assistant_conversation_id = assistant_conversations.id
WHERE assistants.organization_id = ?
	AND assistants.project_id = ?
	AND assistants.status = ?
GROUP BY assistants.id, assistants.name
`, durationValueExpression)
	if err := database.Raw(assistantQuery, fromDate, toDate,
		observability.MetricConversationStatus,
		observability.MetricConversationDuration,
		observability.MetricConversationStatus,
		observability.MetricConversationDuration,
		project.OrganizationID,
		project.ProjectID,
		fromDate,
		toDate,
		project.OrganizationID,
		project.ProjectID,
		type_enums.RECORD_ACTIVE,
	).Scan(&rows).Error; err != nil {
		return nil, nil, 0, err
	}

	latencies := []struct {
		AssistantID uint64
		Value       sql.NullFloat64
	}{}
	latencyQuery := fmt.Sprintf(`
SELECT assistant_latency.assistant_id, AVG(assistant_latency.component_value) AS value
FROM (
	SELECT
		assistant_conversations.assistant_id,
		assistant_conversation_message_metrics.name,
		AVG(%s) AS component_value
	FROM assistant_conversation_message_metrics
	JOIN assistant_conversation_messages
		ON assistant_conversation_messages.message_id = assistant_conversation_message_metrics.assistant_conversation_message_id
		AND assistant_conversation_messages.assistant_conversation_id = assistant_conversation_message_metrics.assistant_conversation_id
	JOIN assistant_conversations
		ON assistant_conversations.id = assistant_conversation_messages.assistant_conversation_id
	WHERE assistant_conversations.organization_id = ?
		AND assistant_conversations.project_id = ?
		AND assistant_conversations.created_date >= ?
		AND assistant_conversations.created_date < ?
		AND assistant_conversation_message_metrics.name IN (?, ?, ?, ?)
	GROUP BY assistant_conversations.assistant_id, assistant_conversation_message_metrics.name
) assistant_latency
GROUP BY assistant_latency.assistant_id
`, latencyValueExpression)
	if err := database.Raw(latencyQuery, project.OrganizationID, project.ProjectID, fromDate, toDate,
		observability.MetricSTTLatencyMs,
		observability.MetricEOSLatencyMs,
		observability.MetricTTSLatencyMs,
		observability.MetricAgentTTFTMs,
	).Scan(&latencies).Error; err != nil {
		return nil, nil, 0, err
	}
	latencyByAssistant := make(map[uint64]uint64, len(latencies))
	for _, latency := range latencies {
		if latency.Value.Valid {
			latencyByAssistant[latency.AssistantID] = uint64(math.Round(latency.Value.Float64))
		}
	}

	slowSessions := uint64(0)
	slowSessionQuery := fmt.Sprintf(`
SELECT COUNT(*)
FROM (
	SELECT conversation_latency.assistant_conversation_id
	FROM (
		SELECT
			assistant_conversations.id AS assistant_conversation_id,
			assistant_conversation_message_metrics.name,
			AVG(%s) AS component_value
		FROM assistant_conversation_message_metrics
		JOIN assistant_conversation_messages
			ON assistant_conversation_messages.message_id = assistant_conversation_message_metrics.assistant_conversation_message_id
			AND assistant_conversation_messages.assistant_conversation_id = assistant_conversation_message_metrics.assistant_conversation_id
		JOIN assistant_conversations
			ON assistant_conversations.id = assistant_conversation_messages.assistant_conversation_id
		WHERE assistant_conversations.organization_id = ?
			AND assistant_conversations.project_id = ?
			AND assistant_conversations.created_date >= ?
			AND assistant_conversations.created_date < ?
			AND assistant_conversation_message_metrics.name IN (?, ?, ?, ?)
		GROUP BY assistant_conversations.id, assistant_conversation_message_metrics.name
	) conversation_latency
	GROUP BY conversation_latency.assistant_conversation_id
	HAVING AVG(conversation_latency.component_value) >= ?
) slow_conversations
`, latencyValueExpression)
	if err := database.Raw(slowSessionQuery, project.OrganizationID, project.ProjectID, fromDate, toDate,
		observability.MetricSTTLatencyMs,
		observability.MetricEOSLatencyMs,
		observability.MetricTTSLatencyMs,
		observability.MetricAgentTTFTMs,
		projectDashboardSlowSessionThresholdMilliseconds,
	).Scan(&slowSessions).Error; err != nil {
		return nil, nil, 0, err
	}

	health := &protos.DashboardHealth{
		FailedSessions:         failedSessions,
		SlowSessions:           slowSessions,
		SlowSessionThresholdMs: projectDashboardSlowSessionThresholdMilliseconds,
	}
	assistants := make([]*protos.DashboardAssistant, 0, len(rows))
	for _, row := range rows {
		assistant := &protos.DashboardAssistant{
			AssistantId:            strconv.FormatUint(row.AssistantID, 10),
			Name:                   row.Name,
			Sessions:               row.Sessions,
			ConversationDurationMs: row.ConversationDuration,
			Health:                 protos.DashboardAssistantHealth_DASHBOARD_ASSISTANT_HEALTH_NO_DATA,
		}
		if row.LastActivity.Valid {
			for _, layout := range []string{
				time.RFC3339Nano,
				"2006-01-02 15:04:05.999999999-07:00",
				"2006-01-02 15:04:05.999999999Z07:00",
				"2006-01-02 15:04:05.999999999",
				"2006-01-02 15:04:05",
			} {
				lastActivity, parseErr := time.Parse(layout, row.LastActivity.String)
				if parseErr == nil {
					assistant.LastActivityAt = timestamppb.New(lastActivity.UTC())
					break
				}
			}
		}
		if row.Sessions == 0 {
			health.NoDataAssistants++
			assistants = append(assistants, assistant)
			continue
		}
		successRate := float64(row.Sessions-row.Failed) / float64(row.Sessions) * 100
		assistant.SuccessRatePercent = &successRate
		if latency, ok := latencyByAssistant[row.AssistantID]; ok {
			assistant.ResponseLatencyMs = &latency
		}
		assistant.Health = projectDashboardAssistantHealth(assistant.SuccessRatePercent, assistant.ResponseLatencyMs)
		switch assistant.Health {
		case protos.DashboardAssistantHealth_DASHBOARD_ASSISTANT_HEALTH_HEALTHY:
			health.HealthyAssistants++
		case protos.DashboardAssistantHealth_DASHBOARD_ASSISTANT_HEALTH_WATCH:
			health.WatchAssistants++
		case protos.DashboardAssistantHealth_DASHBOARD_ASSISTANT_HEALTH_NEEDS_ATTENTION:
			health.NeedsAttentionAssistants++
		default:
			health.NoDataAssistants++
		}
		assistants = append(assistants, assistant)
	}
	sort.Slice(assistants, func(left, right int) bool {
		if assistants[left].GetSessions() == assistants[right].GetSessions() {
			leftID, _ := strconv.ParseUint(assistants[left].GetAssistantId(), 10, 64)
			rightID, _ := strconv.ParseUint(assistants[right].GetAssistantId(), 10, 64)
			return leftID < rightID
		}
		return assistants[left].GetSessions() > assistants[right].GetSessions()
	})
	if len(assistants) > int(limit) {
		assistants = assistants[:limit]
	}
	assistantCount, err := utils.IntToUint32(len(rows))
	if err != nil {
		return nil, nil, 0, fmt.Errorf("assistant count: %w", err)
	}
	return assistants, health, assistantCount, nil
}

func (dashboardService *projectDashboardAnalyticsService) getProjectDashboardRecentFailures(ctx context.Context, project types.ProjectContext, fromDate, toDate time.Time, limit uint32) ([]*protos.DashboardRecentFailure, error) {
	database := dashboardService.postgres.DB(ctx)
	durationValueExpression := `CASE
		WHEN LENGTH(BTRIM(assistant_conversation_metrics.value)) <= 18
			AND BTRIM(assistant_conversation_metrics.value) ~ '^[0-9]+$'
		THEN CAST(BTRIM(assistant_conversation_metrics.value) AS BIGINT)
	END`
	if database.Dialector.Name() == "sqlite" {
		durationValueExpression = `CASE
			WHEN LENGTH(TRIM(assistant_conversation_metrics.value)) <= 18
				AND json_valid(TRIM(assistant_conversation_metrics.value))
			THEN CASE
				WHEN json_type(TRIM(assistant_conversation_metrics.value)) = 'integer'
					AND CAST(TRIM(assistant_conversation_metrics.value) AS INTEGER) >= 0
				THEN CAST(TRIM(assistant_conversation_metrics.value) AS INTEGER)
			END
		END`
	}
	rows := []struct {
		ConversationID uint64
		AssistantID    uint64
		AssistantName  string
		OccurredAt     time.Time
		Channel        string
		Duration       uint64
	}{}
	recentFailureQuery := fmt.Sprintf(`
SELECT
	assistant_conversations.id AS conversation_id,
	assistant_conversations.assistant_id,
	COALESCE(assistants.name, '') AS assistant_name,
	assistant_conversations.created_date AS occurred_at,
	assistant_conversations.source AS channel,
	COALESCE(conversation_metrics.duration_value, 0) AS duration
FROM assistant_conversations
JOIN (
	SELECT
		assistant_conversation_metrics.assistant_conversation_id,
		MAX(CASE WHEN assistant_conversation_metrics.name = ? THEN UPPER(TRIM(assistant_conversation_metrics.value)) END) AS status_value,
		MAX(CASE WHEN assistant_conversation_metrics.name = ? THEN %s END) AS duration_value
	FROM assistant_conversation_metrics
	JOIN assistant_conversations scoped_conversations
		ON scoped_conversations.id = assistant_conversation_metrics.assistant_conversation_id
	WHERE assistant_conversation_metrics.name IN (?, ?)
		AND scoped_conversations.organization_id = ?
		AND scoped_conversations.project_id = ?
		AND scoped_conversations.created_date >= ?
		AND scoped_conversations.created_date < ?
	GROUP BY assistant_conversation_metrics.assistant_conversation_id
) conversation_metrics ON conversation_metrics.assistant_conversation_id = assistant_conversations.id
LEFT JOIN assistants
	ON assistants.id = assistant_conversations.assistant_id
	AND assistants.organization_id = assistant_conversations.organization_id
	AND assistants.project_id = assistant_conversations.project_id
	AND assistants.status = ?
WHERE assistant_conversations.organization_id = ?
	AND assistant_conversations.project_id = ?
	AND assistant_conversations.created_date >= ?
	AND assistant_conversations.created_date < ?
	AND conversation_metrics.status_value IN ('FAILED', 'ERROR')
ORDER BY assistant_conversations.created_date DESC, assistant_conversations.id ASC
LIMIT ?
`, durationValueExpression)
	if err := database.Raw(recentFailureQuery, observability.MetricConversationStatus,
		observability.MetricConversationDuration,
		observability.MetricConversationStatus,
		observability.MetricConversationDuration,
		project.OrganizationID,
		project.ProjectID,
		fromDate,
		toDate,
		type_enums.RECORD_ACTIVE,
		project.OrganizationID,
		project.ProjectID,
		fromDate,
		toDate,
		limit,
	).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []*protos.DashboardRecentFailure{}, nil
	}
	conversationIDs := make([]uint64, 0, len(rows))
	for _, row := range rows {
		conversationIDs = append(conversationIDs, row.ConversationID)
	}
	causeValues := make(map[uint64][]string, len(rows))
	metricRows := []struct {
		ConversationID uint64
		Description    string
	}{}
	if err := database.Table("assistant_conversation_metrics").
		Select("assistant_conversation_id AS conversation_id, description").
		Where("assistant_conversation_id IN ? AND name IN ?", conversationIDs, []string{observability.MetricConversationStatus, observability.MetricCallStatus}).
		Scan(&metricRows).Error; err != nil {
		return nil, err
	}
	for _, metric := range metricRows {
		causeValues[metric.ConversationID] = append(causeValues[metric.ConversationID], metric.Description)
	}
	metadataRows := []struct {
		ConversationID uint64
		Key            string
		Value          string
	}{}
	if err := database.Table("assistant_conversation_metadata").
		Select("assistant_conversation_id AS conversation_id, key, value").
		Where("assistant_conversation_id IN ?", conversationIDs).
		Scan(&metadataRows).Error; err != nil {
		return nil, err
	}
	for _, metadata := range metadataRows {
		key := strings.ToLower(metadata.Key)
		if strings.Contains(key, "disconnect") || strings.Contains(key, "failure") || strings.Contains(key, "error") {
			causeValues[metadata.ConversationID] = append(causeValues[metadata.ConversationID], metadata.Value)
		}
	}
	failures := make([]*protos.DashboardRecentFailure, 0, len(rows))
	for _, row := range rows {
		failures = append(failures, &protos.DashboardRecentFailure{
			ConversationId: strconv.FormatUint(row.ConversationID, 10),
			AssistantId:    strconv.FormatUint(row.AssistantID, 10),
			AssistantName:  row.AssistantName,
			OccurredAt:     timestamppb.New(row.OccurredAt.UTC()),
			Channel:        row.Channel,
			DurationMs:     row.Duration,
			FailureCode:    projectDashboardFailureCode(causeValues[row.ConversationID]),
		})
	}
	return failures, nil
}

func projectDashboardAssistantHealth(successRate *float64, latencyMs *uint64) protos.DashboardAssistantHealth {
	if successRate == nil {
		return protos.DashboardAssistantHealth_DASHBOARD_ASSISTANT_HEALTH_NO_DATA
	}
	if *successRate < 90 || (latencyMs != nil && *latencyMs >= 1500) {
		return protos.DashboardAssistantHealth_DASHBOARD_ASSISTANT_HEALTH_NEEDS_ATTENTION
	}
	if *successRate >= 95 && latencyMs != nil && *latencyMs < 1000 {
		return protos.DashboardAssistantHealth_DASHBOARD_ASSISTANT_HEALTH_HEALTHY
	}
	return protos.DashboardAssistantHealth_DASHBOARD_ASSISTANT_HEALTH_WATCH
}

func projectDashboardFailureCode(values []string) protos.DashboardFailureCode {
	cause := strings.ToLower(strings.Join(values, " "))
	switch {
	case strings.Contains(cause, "auth"), strings.Contains(cause, "forbidden"):
		return protos.DashboardFailureCode_DASHBOARD_FAILURE_CODE_AUTHENTICATION
	case strings.Contains(cause, "config"), strings.Contains(cause, "invalid"):
		return protos.DashboardFailureCode_DASHBOARD_FAILURE_CODE_CONFIGURATION
	case strings.Contains(cause, "provider"), strings.Contains(cause, "upstream"):
		return protos.DashboardFailureCode_DASHBOARD_FAILURE_CODE_PROVIDER
	case strings.Contains(cause, "network"), strings.Contains(cause, "transport"), strings.Contains(cause, "dns"):
		return protos.DashboardFailureCode_DASHBOARD_FAILURE_CODE_NETWORK
	case strings.Contains(cause, "cancel"):
		return protos.DashboardFailureCode_DASHBOARD_FAILURE_CODE_CANCELLED
	case strings.Contains(cause, "no_answer"), strings.Contains(cause, "no answer"), strings.Contains(cause, "timeout"):
		return protos.DashboardFailureCode_DASHBOARD_FAILURE_CODE_NO_ANSWER
	case strings.Contains(cause, "reject"), strings.Contains(cause, "decline"), strings.Contains(cause, "busy"):
		return protos.DashboardFailureCode_DASHBOARD_FAILURE_CODE_REJECTED
	default:
		return protos.DashboardFailureCode_DASHBOARD_FAILURE_CODE_UNKNOWN
	}
}
