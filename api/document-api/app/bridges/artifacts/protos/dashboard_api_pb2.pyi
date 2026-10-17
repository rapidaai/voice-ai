import datetime

import app.bridges.artifacts.protos.common_pb2 as _common_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DashboardSectionState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DASHBOARD_SECTION_STATE_UNSPECIFIED: _ClassVar[DashboardSectionState]
    DASHBOARD_SECTION_STATE_READY: _ClassVar[DashboardSectionState]
    DASHBOARD_SECTION_STATE_EMPTY: _ClassVar[DashboardSectionState]
    DASHBOARD_SECTION_STATE_UNAVAILABLE: _ClassVar[DashboardSectionState]

class DashboardAssistantHealth(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DASHBOARD_ASSISTANT_HEALTH_UNSPECIFIED: _ClassVar[DashboardAssistantHealth]
    DASHBOARD_ASSISTANT_HEALTH_HEALTHY: _ClassVar[DashboardAssistantHealth]
    DASHBOARD_ASSISTANT_HEALTH_WATCH: _ClassVar[DashboardAssistantHealth]
    DASHBOARD_ASSISTANT_HEALTH_NEEDS_ATTENTION: _ClassVar[DashboardAssistantHealth]
    DASHBOARD_ASSISTANT_HEALTH_NO_DATA: _ClassVar[DashboardAssistantHealth]

class DashboardUsageComponent(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DASHBOARD_USAGE_COMPONENT_UNSPECIFIED: _ClassVar[DashboardUsageComponent]
    DASHBOARD_USAGE_COMPONENT_STT: _ClassVar[DashboardUsageComponent]
    DASHBOARD_USAGE_COMPONENT_TTS: _ClassVar[DashboardUsageComponent]
    DASHBOARD_USAGE_COMPONENT_LLM: _ClassVar[DashboardUsageComponent]
    DASHBOARD_USAGE_COMPONENT_VAD: _ClassVar[DashboardUsageComponent]
    DASHBOARD_USAGE_COMPONENT_EOS: _ClassVar[DashboardUsageComponent]
    DASHBOARD_USAGE_COMPONENT_DENOISING: _ClassVar[DashboardUsageComponent]

class DashboardFailureCode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DASHBOARD_FAILURE_CODE_UNSPECIFIED: _ClassVar[DashboardFailureCode]
    DASHBOARD_FAILURE_CODE_AUTHENTICATION: _ClassVar[DashboardFailureCode]
    DASHBOARD_FAILURE_CODE_CONFIGURATION: _ClassVar[DashboardFailureCode]
    DASHBOARD_FAILURE_CODE_PROVIDER: _ClassVar[DashboardFailureCode]
    DASHBOARD_FAILURE_CODE_NETWORK: _ClassVar[DashboardFailureCode]
    DASHBOARD_FAILURE_CODE_CANCELLED: _ClassVar[DashboardFailureCode]
    DASHBOARD_FAILURE_CODE_NO_ANSWER: _ClassVar[DashboardFailureCode]
    DASHBOARD_FAILURE_CODE_REJECTED: _ClassVar[DashboardFailureCode]
    DASHBOARD_FAILURE_CODE_UNKNOWN: _ClassVar[DashboardFailureCode]
DASHBOARD_SECTION_STATE_UNSPECIFIED: DashboardSectionState
DASHBOARD_SECTION_STATE_READY: DashboardSectionState
DASHBOARD_SECTION_STATE_EMPTY: DashboardSectionState
DASHBOARD_SECTION_STATE_UNAVAILABLE: DashboardSectionState
DASHBOARD_ASSISTANT_HEALTH_UNSPECIFIED: DashboardAssistantHealth
DASHBOARD_ASSISTANT_HEALTH_HEALTHY: DashboardAssistantHealth
DASHBOARD_ASSISTANT_HEALTH_WATCH: DashboardAssistantHealth
DASHBOARD_ASSISTANT_HEALTH_NEEDS_ATTENTION: DashboardAssistantHealth
DASHBOARD_ASSISTANT_HEALTH_NO_DATA: DashboardAssistantHealth
DASHBOARD_USAGE_COMPONENT_UNSPECIFIED: DashboardUsageComponent
DASHBOARD_USAGE_COMPONENT_STT: DashboardUsageComponent
DASHBOARD_USAGE_COMPONENT_TTS: DashboardUsageComponent
DASHBOARD_USAGE_COMPONENT_LLM: DashboardUsageComponent
DASHBOARD_USAGE_COMPONENT_VAD: DashboardUsageComponent
DASHBOARD_USAGE_COMPONENT_EOS: DashboardUsageComponent
DASHBOARD_USAGE_COMPONENT_DENOISING: DashboardUsageComponent
DASHBOARD_FAILURE_CODE_UNSPECIFIED: DashboardFailureCode
DASHBOARD_FAILURE_CODE_AUTHENTICATION: DashboardFailureCode
DASHBOARD_FAILURE_CODE_CONFIGURATION: DashboardFailureCode
DASHBOARD_FAILURE_CODE_PROVIDER: DashboardFailureCode
DASHBOARD_FAILURE_CODE_NETWORK: DashboardFailureCode
DASHBOARD_FAILURE_CODE_CANCELLED: DashboardFailureCode
DASHBOARD_FAILURE_CODE_NO_ANSWER: DashboardFailureCode
DASHBOARD_FAILURE_CODE_REJECTED: DashboardFailureCode
DASHBOARD_FAILURE_CODE_UNKNOWN: DashboardFailureCode

class GetProjectDashboardRequest(_message.Message):
    __slots__ = ("fromDate", "toDate", "assistantLimit", "recentFailureLimit")
    FROMDATE_FIELD_NUMBER: _ClassVar[int]
    TODATE_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTLIMIT_FIELD_NUMBER: _ClassVar[int]
    RECENTFAILURELIMIT_FIELD_NUMBER: _ClassVar[int]
    fromDate: _timestamp_pb2.Timestamp
    toDate: _timestamp_pb2.Timestamp
    assistantLimit: int
    recentFailureLimit: int
    def __init__(self, fromDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., toDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., assistantLimit: _Optional[int] = ..., recentFailureLimit: _Optional[int] = ...) -> None: ...

class GetProjectDashboardResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: ProjectDashboard
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Union[ProjectDashboard, _Mapping]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class GetProjectDashboardAnalyticsRequest(_message.Message):
    __slots__ = ("fromDate", "toDate", "comparisonFromDate", "comparisonToDate", "assistantLimit", "recentFailureLimit")
    FROMDATE_FIELD_NUMBER: _ClassVar[int]
    TODATE_FIELD_NUMBER: _ClassVar[int]
    COMPARISONFROMDATE_FIELD_NUMBER: _ClassVar[int]
    COMPARISONTODATE_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTLIMIT_FIELD_NUMBER: _ClassVar[int]
    RECENTFAILURELIMIT_FIELD_NUMBER: _ClassVar[int]
    fromDate: _timestamp_pb2.Timestamp
    toDate: _timestamp_pb2.Timestamp
    comparisonFromDate: _timestamp_pb2.Timestamp
    comparisonToDate: _timestamp_pb2.Timestamp
    assistantLimit: int
    recentFailureLimit: int
    def __init__(self, fromDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., toDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., comparisonFromDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., comparisonToDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., assistantLimit: _Optional[int] = ..., recentFailureLimit: _Optional[int] = ...) -> None: ...

class GetProjectDashboardAnalyticsResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: ProjectDashboardAnalytics
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Union[ProjectDashboardAnalytics, _Mapping]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class ProjectDashboard(_message.Message):
    __slots__ = ("generatedAt", "period", "analytics", "usage", "partial")
    GENERATEDAT_FIELD_NUMBER: _ClassVar[int]
    PERIOD_FIELD_NUMBER: _ClassVar[int]
    ANALYTICS_FIELD_NUMBER: _ClassVar[int]
    USAGE_FIELD_NUMBER: _ClassVar[int]
    PARTIAL_FIELD_NUMBER: _ClassVar[int]
    generatedAt: _timestamp_pb2.Timestamp
    period: DashboardPeriod
    analytics: DashboardAnalyticsSection
    usage: DashboardUsageSection
    partial: bool
    def __init__(self, generatedAt: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., period: _Optional[_Union[DashboardPeriod, _Mapping]] = ..., analytics: _Optional[_Union[DashboardAnalyticsSection, _Mapping]] = ..., usage: _Optional[_Union[DashboardUsageSection, _Mapping]] = ..., partial: bool = ...) -> None: ...

class DashboardPeriod(_message.Message):
    __slots__ = ("fromDate", "toDate", "comparisonFromDate", "comparisonToDate")
    FROMDATE_FIELD_NUMBER: _ClassVar[int]
    TODATE_FIELD_NUMBER: _ClassVar[int]
    COMPARISONFROMDATE_FIELD_NUMBER: _ClassVar[int]
    COMPARISONTODATE_FIELD_NUMBER: _ClassVar[int]
    fromDate: _timestamp_pb2.Timestamp
    toDate: _timestamp_pb2.Timestamp
    comparisonFromDate: _timestamp_pb2.Timestamp
    comparisonToDate: _timestamp_pb2.Timestamp
    def __init__(self, fromDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., toDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., comparisonFromDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., comparisonToDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class DashboardAnalyticsSection(_message.Message):
    __slots__ = ("state", "data")
    STATE_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    state: DashboardSectionState
    data: ProjectDashboardAnalytics
    def __init__(self, state: _Optional[_Union[DashboardSectionState, str]] = ..., data: _Optional[_Union[ProjectDashboardAnalytics, _Mapping]] = ...) -> None: ...

class DashboardUsageSection(_message.Message):
    __slots__ = ("state", "components")
    STATE_FIELD_NUMBER: _ClassVar[int]
    COMPONENTS_FIELD_NUMBER: _ClassVar[int]
    state: DashboardSectionState
    components: _containers.RepeatedCompositeFieldContainer[DashboardComponentUsage]
    def __init__(self, state: _Optional[_Union[DashboardSectionState, str]] = ..., components: _Optional[_Iterable[_Union[DashboardComponentUsage, _Mapping]]] = ...) -> None: ...

class ProjectDashboardAnalytics(_message.Message):
    __slots__ = ("projectState", "summary", "activity", "health", "assistants", "recentFailures")
    PROJECTSTATE_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    ACTIVITY_FIELD_NUMBER: _ClassVar[int]
    HEALTH_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTS_FIELD_NUMBER: _ClassVar[int]
    RECENTFAILURES_FIELD_NUMBER: _ClassVar[int]
    projectState: DashboardProjectState
    summary: DashboardSummary
    activity: _containers.RepeatedCompositeFieldContainer[DashboardActivityBucket]
    health: DashboardHealth
    assistants: _containers.RepeatedCompositeFieldContainer[DashboardAssistant]
    recentFailures: _containers.RepeatedCompositeFieldContainer[DashboardRecentFailure]
    def __init__(self, projectState: _Optional[_Union[DashboardProjectState, _Mapping]] = ..., summary: _Optional[_Union[DashboardSummary, _Mapping]] = ..., activity: _Optional[_Iterable[_Union[DashboardActivityBucket, _Mapping]]] = ..., health: _Optional[_Union[DashboardHealth, _Mapping]] = ..., assistants: _Optional[_Iterable[_Union[DashboardAssistant, _Mapping]]] = ..., recentFailures: _Optional[_Iterable[_Union[DashboardRecentFailure, _Mapping]]] = ...) -> None: ...

class DashboardProjectState(_message.Message):
    __slots__ = ("assistantCount", "hasAssistants", "hasActivity")
    ASSISTANTCOUNT_FIELD_NUMBER: _ClassVar[int]
    HASASSISTANTS_FIELD_NUMBER: _ClassVar[int]
    HASACTIVITY_FIELD_NUMBER: _ClassVar[int]
    assistantCount: int
    hasAssistants: bool
    hasActivity: bool
    def __init__(self, assistantCount: _Optional[int] = ..., hasAssistants: bool = ..., hasActivity: bool = ...) -> None: ...

class DashboardSummary(_message.Message):
    __slots__ = ("sessions", "successRate", "responseLatency", "failedSessions")
    SESSIONS_FIELD_NUMBER: _ClassVar[int]
    SUCCESSRATE_FIELD_NUMBER: _ClassVar[int]
    RESPONSELATENCY_FIELD_NUMBER: _ClassVar[int]
    FAILEDSESSIONS_FIELD_NUMBER: _ClassVar[int]
    sessions: DashboardCountComparison
    successRate: DashboardRateComparison
    responseLatency: DashboardLatencyComparison
    failedSessions: DashboardFailedComparison
    def __init__(self, sessions: _Optional[_Union[DashboardCountComparison, _Mapping]] = ..., successRate: _Optional[_Union[DashboardRateComparison, _Mapping]] = ..., responseLatency: _Optional[_Union[DashboardLatencyComparison, _Mapping]] = ..., failedSessions: _Optional[_Union[DashboardFailedComparison, _Mapping]] = ...) -> None: ...

class DashboardCountComparison(_message.Message):
    __slots__ = ("value", "previousValue", "changePercent")
    VALUE_FIELD_NUMBER: _ClassVar[int]
    PREVIOUSVALUE_FIELD_NUMBER: _ClassVar[int]
    CHANGEPERCENT_FIELD_NUMBER: _ClassVar[int]
    value: int
    previousValue: int
    changePercent: float
    def __init__(self, value: _Optional[int] = ..., previousValue: _Optional[int] = ..., changePercent: _Optional[float] = ...) -> None: ...

class DashboardRateComparison(_message.Message):
    __slots__ = ("valuePercent", "previousValuePercent", "changePoints")
    VALUEPERCENT_FIELD_NUMBER: _ClassVar[int]
    PREVIOUSVALUEPERCENT_FIELD_NUMBER: _ClassVar[int]
    CHANGEPOINTS_FIELD_NUMBER: _ClassVar[int]
    valuePercent: float
    previousValuePercent: float
    changePoints: float
    def __init__(self, valuePercent: _Optional[float] = ..., previousValuePercent: _Optional[float] = ..., changePoints: _Optional[float] = ...) -> None: ...

class DashboardLatencyComparison(_message.Message):
    __slots__ = ("valueMs", "previousValueMs", "changeMs")
    VALUEMS_FIELD_NUMBER: _ClassVar[int]
    PREVIOUSVALUEMS_FIELD_NUMBER: _ClassVar[int]
    CHANGEMS_FIELD_NUMBER: _ClassVar[int]
    valueMs: int
    previousValueMs: int
    changeMs: int
    def __init__(self, valueMs: _Optional[int] = ..., previousValueMs: _Optional[int] = ..., changeMs: _Optional[int] = ...) -> None: ...

class DashboardFailedComparison(_message.Message):
    __slots__ = ("value", "previousValue", "sharePercent")
    VALUE_FIELD_NUMBER: _ClassVar[int]
    PREVIOUSVALUE_FIELD_NUMBER: _ClassVar[int]
    SHAREPERCENT_FIELD_NUMBER: _ClassVar[int]
    value: int
    previousValue: int
    sharePercent: float
    def __init__(self, value: _Optional[int] = ..., previousValue: _Optional[int] = ..., sharePercent: _Optional[float] = ...) -> None: ...

class DashboardActivityBucket(_message.Message):
    __slots__ = ("fromDate", "toDate", "sessions", "failedSessions")
    FROMDATE_FIELD_NUMBER: _ClassVar[int]
    TODATE_FIELD_NUMBER: _ClassVar[int]
    SESSIONS_FIELD_NUMBER: _ClassVar[int]
    FAILEDSESSIONS_FIELD_NUMBER: _ClassVar[int]
    fromDate: _timestamp_pb2.Timestamp
    toDate: _timestamp_pb2.Timestamp
    sessions: int
    failedSessions: int
    def __init__(self, fromDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., toDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., sessions: _Optional[int] = ..., failedSessions: _Optional[int] = ...) -> None: ...

class DashboardHealth(_message.Message):
    __slots__ = ("failedSessions", "slowSessions", "slowSessionThresholdMs", "healthyAssistants", "watchAssistants", "needsAttentionAssistants", "noDataAssistants")
    FAILEDSESSIONS_FIELD_NUMBER: _ClassVar[int]
    SLOWSESSIONS_FIELD_NUMBER: _ClassVar[int]
    SLOWSESSIONTHRESHOLDMS_FIELD_NUMBER: _ClassVar[int]
    HEALTHYASSISTANTS_FIELD_NUMBER: _ClassVar[int]
    WATCHASSISTANTS_FIELD_NUMBER: _ClassVar[int]
    NEEDSATTENTIONASSISTANTS_FIELD_NUMBER: _ClassVar[int]
    NODATAASSISTANTS_FIELD_NUMBER: _ClassVar[int]
    failedSessions: int
    slowSessions: int
    slowSessionThresholdMs: int
    healthyAssistants: int
    watchAssistants: int
    needsAttentionAssistants: int
    noDataAssistants: int
    def __init__(self, failedSessions: _Optional[int] = ..., slowSessions: _Optional[int] = ..., slowSessionThresholdMs: _Optional[int] = ..., healthyAssistants: _Optional[int] = ..., watchAssistants: _Optional[int] = ..., needsAttentionAssistants: _Optional[int] = ..., noDataAssistants: _Optional[int] = ...) -> None: ...

class DashboardAssistant(_message.Message):
    __slots__ = ("assistantId", "name", "sessions", "successRatePercent", "responseLatencyMs", "conversationDurationMs", "lastActivityAt", "health")
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    SESSIONS_FIELD_NUMBER: _ClassVar[int]
    SUCCESSRATEPERCENT_FIELD_NUMBER: _ClassVar[int]
    RESPONSELATENCYMS_FIELD_NUMBER: _ClassVar[int]
    CONVERSATIONDURATIONMS_FIELD_NUMBER: _ClassVar[int]
    LASTACTIVITYAT_FIELD_NUMBER: _ClassVar[int]
    HEALTH_FIELD_NUMBER: _ClassVar[int]
    assistantId: str
    name: str
    sessions: int
    successRatePercent: float
    responseLatencyMs: int
    conversationDurationMs: int
    lastActivityAt: _timestamp_pb2.Timestamp
    health: DashboardAssistantHealth
    def __init__(self, assistantId: _Optional[str] = ..., name: _Optional[str] = ..., sessions: _Optional[int] = ..., successRatePercent: _Optional[float] = ..., responseLatencyMs: _Optional[int] = ..., conversationDurationMs: _Optional[int] = ..., lastActivityAt: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., health: _Optional[_Union[DashboardAssistantHealth, str]] = ...) -> None: ...

class DashboardComponentUsage(_message.Message):
    __slots__ = ("component", "durationMs")
    COMPONENT_FIELD_NUMBER: _ClassVar[int]
    DURATIONMS_FIELD_NUMBER: _ClassVar[int]
    component: DashboardUsageComponent
    durationMs: int
    def __init__(self, component: _Optional[_Union[DashboardUsageComponent, str]] = ..., durationMs: _Optional[int] = ...) -> None: ...

class DashboardRecentFailure(_message.Message):
    __slots__ = ("conversationId", "assistantId", "assistantName", "occurredAt", "channel", "durationMs", "failureCode")
    CONVERSATIONID_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTNAME_FIELD_NUMBER: _ClassVar[int]
    OCCURREDAT_FIELD_NUMBER: _ClassVar[int]
    CHANNEL_FIELD_NUMBER: _ClassVar[int]
    DURATIONMS_FIELD_NUMBER: _ClassVar[int]
    FAILURECODE_FIELD_NUMBER: _ClassVar[int]
    conversationId: str
    assistantId: str
    assistantName: str
    occurredAt: _timestamp_pb2.Timestamp
    channel: str
    durationMs: int
    failureCode: DashboardFailureCode
    def __init__(self, conversationId: _Optional[str] = ..., assistantId: _Optional[str] = ..., assistantName: _Optional[str] = ..., occurredAt: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., channel: _Optional[str] = ..., durationMs: _Optional[int] = ..., failureCode: _Optional[_Union[DashboardFailureCode, str]] = ...) -> None: ...
