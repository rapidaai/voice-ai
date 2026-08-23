import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
import app.bridges.artifacts.protos.common_pb2 as _common_pb2
import app.bridges.artifacts.protos.assistant_deployment_pb2 as _assistant_deployment_pb2
import app.bridges.artifacts.protos.assistant_tool_pb2 as _assistant_tool_pb2
import app.bridges.artifacts.protos.assistant_http_log_pb2 as _assistant_http_log_pb2
import app.bridges.artifacts.protos.assistant_knowledge_pb2 as _assistant_knowledge_pb2
import app.bridges.artifacts.protos.assistant_provider_pb2 as _assistant_provider_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Assistant(_message.Message):
    __slots__ = ("id", "status", "visibility", "source", "sourceIdentifier", "projectId", "organizationId", "assistantProvider", "assistantProviderId", "name", "description", "assistantProviderModel", "assistantProviderAgentkit", "assistantProviderWebsocket", "assistantProviderAgentflow", "assistantTag", "createdActor", "updatedActor", "createdDate", "updatedDate", "debuggerDeployment", "phoneDeployment", "whatsappDeployment", "webPluginDeployment", "apiDeployment", "assistantConversations", "assistantTools", "assistantConfigurations")
    ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    VISIBILITY_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    SOURCEIDENTIFIER_FIELD_NUMBER: _ClassVar[int]
    PROJECTID_FIELD_NUMBER: _ClassVar[int]
    ORGANIZATIONID_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTPROVIDER_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTPROVIDERID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTPROVIDERMODEL_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTPROVIDERAGENTKIT_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTPROVIDERWEBSOCKET_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTPROVIDERAGENTFLOW_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTTAG_FIELD_NUMBER: _ClassVar[int]
    CREATEDACTOR_FIELD_NUMBER: _ClassVar[int]
    UPDATEDACTOR_FIELD_NUMBER: _ClassVar[int]
    CREATEDDATE_FIELD_NUMBER: _ClassVar[int]
    UPDATEDDATE_FIELD_NUMBER: _ClassVar[int]
    DEBUGGERDEPLOYMENT_FIELD_NUMBER: _ClassVar[int]
    PHONEDEPLOYMENT_FIELD_NUMBER: _ClassVar[int]
    WHATSAPPDEPLOYMENT_FIELD_NUMBER: _ClassVar[int]
    WEBPLUGINDEPLOYMENT_FIELD_NUMBER: _ClassVar[int]
    APIDEPLOYMENT_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTCONVERSATIONS_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTTOOLS_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTCONFIGURATIONS_FIELD_NUMBER: _ClassVar[int]
    id: int
    status: str
    visibility: str
    source: str
    sourceIdentifier: int
    projectId: int
    organizationId: int
    assistantProvider: str
    assistantProviderId: int
    name: str
    description: str
    assistantProviderModel: _assistant_provider_pb2.AssistantProviderModel
    assistantProviderAgentkit: _assistant_provider_pb2.AssistantProviderAgentkit
    assistantProviderWebsocket: _assistant_provider_pb2.AssistantProviderWebsocket
    assistantProviderAgentflow: _assistant_provider_pb2.AssistantProviderAgentflow
    assistantTag: _common_pb2.Tag
    createdActor: _common_pb2.AuditActor
    updatedActor: _common_pb2.AuditActor
    createdDate: _timestamp_pb2.Timestamp
    updatedDate: _timestamp_pb2.Timestamp
    debuggerDeployment: _assistant_deployment_pb2.AssistantDebuggerDeployment
    phoneDeployment: _assistant_deployment_pb2.AssistantPhoneDeployment
    whatsappDeployment: _assistant_deployment_pb2.AssistantWhatsappDeployment
    webPluginDeployment: _assistant_deployment_pb2.AssistantWebpluginDeployment
    apiDeployment: _assistant_deployment_pb2.AssistantApiDeployment
    assistantConversations: _containers.RepeatedCompositeFieldContainer[_common_pb2.AssistantConversation]
    assistantTools: _containers.RepeatedCompositeFieldContainer[_assistant_tool_pb2.AssistantTool]
    assistantConfigurations: _containers.RepeatedCompositeFieldContainer[AssistantConfiguration]
    def __init__(self, id: _Optional[int] = ..., status: _Optional[str] = ..., visibility: _Optional[str] = ..., source: _Optional[str] = ..., sourceIdentifier: _Optional[int] = ..., projectId: _Optional[int] = ..., organizationId: _Optional[int] = ..., assistantProvider: _Optional[str] = ..., assistantProviderId: _Optional[int] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., assistantProviderModel: _Optional[_Union[_assistant_provider_pb2.AssistantProviderModel, _Mapping]] = ..., assistantProviderAgentkit: _Optional[_Union[_assistant_provider_pb2.AssistantProviderAgentkit, _Mapping]] = ..., assistantProviderWebsocket: _Optional[_Union[_assistant_provider_pb2.AssistantProviderWebsocket, _Mapping]] = ..., assistantProviderAgentflow: _Optional[_Union[_assistant_provider_pb2.AssistantProviderAgentflow, _Mapping]] = ..., assistantTag: _Optional[_Union[_common_pb2.Tag, _Mapping]] = ..., createdActor: _Optional[_Union[_common_pb2.AuditActor, _Mapping]] = ..., updatedActor: _Optional[_Union[_common_pb2.AuditActor, _Mapping]] = ..., createdDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updatedDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., debuggerDeployment: _Optional[_Union[_assistant_deployment_pb2.AssistantDebuggerDeployment, _Mapping]] = ..., phoneDeployment: _Optional[_Union[_assistant_deployment_pb2.AssistantPhoneDeployment, _Mapping]] = ..., whatsappDeployment: _Optional[_Union[_assistant_deployment_pb2.AssistantWhatsappDeployment, _Mapping]] = ..., webPluginDeployment: _Optional[_Union[_assistant_deployment_pb2.AssistantWebpluginDeployment, _Mapping]] = ..., apiDeployment: _Optional[_Union[_assistant_deployment_pb2.AssistantApiDeployment, _Mapping]] = ..., assistantConversations: _Optional[_Iterable[_Union[_common_pb2.AssistantConversation, _Mapping]]] = ..., assistantTools: _Optional[_Iterable[_Union[_assistant_tool_pb2.AssistantTool, _Mapping]]] = ..., assistantConfigurations: _Optional[_Iterable[_Union[AssistantConfiguration, _Mapping]]] = ...) -> None: ...

class CreateAssistantRequest(_message.Message):
    __slots__ = ("assistantProvider", "assistantKnowledges", "assistantTools", "description", "visibility", "language", "source", "sourceIdentifier", "tags", "name")
    ASSISTANTPROVIDER_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTKNOWLEDGES_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTTOOLS_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    VISIBILITY_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    SOURCEIDENTIFIER_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    assistantProvider: _assistant_provider_pb2.CreateAssistantProviderRequest
    assistantKnowledges: _containers.RepeatedCompositeFieldContainer[_assistant_knowledge_pb2.CreateAssistantKnowledgeRequest]
    assistantTools: _containers.RepeatedCompositeFieldContainer[_assistant_tool_pb2.CreateAssistantToolRequest]
    description: str
    visibility: str
    language: str
    source: str
    sourceIdentifier: int
    tags: _containers.RepeatedScalarFieldContainer[str]
    name: str
    def __init__(self, assistantProvider: _Optional[_Union[_assistant_provider_pb2.CreateAssistantProviderRequest, _Mapping]] = ..., assistantKnowledges: _Optional[_Iterable[_Union[_assistant_knowledge_pb2.CreateAssistantKnowledgeRequest, _Mapping]]] = ..., assistantTools: _Optional[_Iterable[_Union[_assistant_tool_pb2.CreateAssistantToolRequest, _Mapping]]] = ..., description: _Optional[str] = ..., visibility: _Optional[str] = ..., language: _Optional[str] = ..., source: _Optional[str] = ..., sourceIdentifier: _Optional[int] = ..., tags: _Optional[_Iterable[str]] = ..., name: _Optional[str] = ...) -> None: ...

class CreateAssistantTagRequest(_message.Message):
    __slots__ = ("assistantId", "tags")
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    assistantId: int
    tags: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, assistantId: _Optional[int] = ..., tags: _Optional[_Iterable[str]] = ...) -> None: ...

class GetAssistantRequest(_message.Message):
    __slots__ = ("assistantDefinition",)
    ASSISTANTDEFINITION_FIELD_NUMBER: _ClassVar[int]
    assistantDefinition: _common_pb2.AssistantDefinition
    def __init__(self, assistantDefinition: _Optional[_Union[_common_pb2.AssistantDefinition, _Mapping]] = ...) -> None: ...

class DeleteAssistantRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: int
    def __init__(self, id: _Optional[int] = ...) -> None: ...

class GetAssistantResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: Assistant
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Union[Assistant, _Mapping]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class GetAllAssistantRequest(_message.Message):
    __slots__ = ("paginate", "criterias")
    PAGINATE_FIELD_NUMBER: _ClassVar[int]
    CRITERIAS_FIELD_NUMBER: _ClassVar[int]
    paginate: _common_pb2.Paginate
    criterias: _containers.RepeatedCompositeFieldContainer[_common_pb2.Criteria]
    def __init__(self, paginate: _Optional[_Union[_common_pb2.Paginate, _Mapping]] = ..., criterias: _Optional[_Iterable[_Union[_common_pb2.Criteria, _Mapping]]] = ...) -> None: ...

class AssistantConfiguration(_message.Message):
    __slots__ = ("id", "assistantId", "projectId", "organizationId", "configurationType", "provider", "enabled", "options", "status", "createdActor", "updatedActor", "createdDate", "updatedDate")
    ID_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    PROJECTID_FIELD_NUMBER: _ClassVar[int]
    ORGANIZATIONID_FIELD_NUMBER: _ClassVar[int]
    CONFIGURATIONTYPE_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    OPTIONS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CREATEDACTOR_FIELD_NUMBER: _ClassVar[int]
    UPDATEDACTOR_FIELD_NUMBER: _ClassVar[int]
    CREATEDDATE_FIELD_NUMBER: _ClassVar[int]
    UPDATEDDATE_FIELD_NUMBER: _ClassVar[int]
    id: int
    assistantId: int
    projectId: int
    organizationId: int
    configurationType: str
    provider: str
    enabled: bool
    options: _containers.RepeatedCompositeFieldContainer[_common_pb2.Metadata]
    status: str
    createdActor: _common_pb2.AuditActor
    updatedActor: _common_pb2.AuditActor
    createdDate: _timestamp_pb2.Timestamp
    updatedDate: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[int] = ..., assistantId: _Optional[int] = ..., projectId: _Optional[int] = ..., organizationId: _Optional[int] = ..., configurationType: _Optional[str] = ..., provider: _Optional[str] = ..., enabled: bool = ..., options: _Optional[_Iterable[_Union[_common_pb2.Metadata, _Mapping]]] = ..., status: _Optional[str] = ..., createdActor: _Optional[_Union[_common_pb2.AuditActor, _Mapping]] = ..., updatedActor: _Optional[_Union[_common_pb2.AuditActor, _Mapping]] = ..., createdDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updatedDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetAssistantConfigurationRequest(_message.Message):
    __slots__ = ("id", "assistantId")
    ID_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    id: int
    assistantId: int
    def __init__(self, id: _Optional[int] = ..., assistantId: _Optional[int] = ...) -> None: ...

class GetAssistantConfigurationResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: AssistantConfiguration
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Union[AssistantConfiguration, _Mapping]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class GetAllAssistantConfigurationRequest(_message.Message):
    __slots__ = ("assistantId", "configurationType", "provider", "paginate", "criterias")
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    CONFIGURATIONTYPE_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    PAGINATE_FIELD_NUMBER: _ClassVar[int]
    CRITERIAS_FIELD_NUMBER: _ClassVar[int]
    assistantId: int
    configurationType: str
    provider: str
    paginate: _common_pb2.Paginate
    criterias: _containers.RepeatedCompositeFieldContainer[_common_pb2.Criteria]
    def __init__(self, assistantId: _Optional[int] = ..., configurationType: _Optional[str] = ..., provider: _Optional[str] = ..., paginate: _Optional[_Union[_common_pb2.Paginate, _Mapping]] = ..., criterias: _Optional[_Iterable[_Union[_common_pb2.Criteria, _Mapping]]] = ...) -> None: ...

class GetAllAssistantConfigurationResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error", "paginated")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    PAGINATED_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: _containers.RepeatedCompositeFieldContainer[AssistantConfiguration]
    error: _common_pb2.Error
    paginated: _common_pb2.Paginated
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Iterable[_Union[AssistantConfiguration, _Mapping]]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ..., paginated: _Optional[_Union[_common_pb2.Paginated, _Mapping]] = ...) -> None: ...

class CreateAssistantConfigurationRequest(_message.Message):
    __slots__ = ("assistantId", "configurationType", "provider", "enabled", "options")
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    CONFIGURATIONTYPE_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    OPTIONS_FIELD_NUMBER: _ClassVar[int]
    assistantId: int
    configurationType: str
    provider: str
    enabled: bool
    options: _containers.RepeatedCompositeFieldContainer[_common_pb2.Metadata]
    def __init__(self, assistantId: _Optional[int] = ..., configurationType: _Optional[str] = ..., provider: _Optional[str] = ..., enabled: bool = ..., options: _Optional[_Iterable[_Union[_common_pb2.Metadata, _Mapping]]] = ...) -> None: ...

class UpdateAssistantConfigurationRequest(_message.Message):
    __slots__ = ("id", "assistantId", "configurationType", "provider", "enabled", "options")
    ID_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    CONFIGURATIONTYPE_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    OPTIONS_FIELD_NUMBER: _ClassVar[int]
    id: int
    assistantId: int
    configurationType: str
    provider: str
    enabled: bool
    options: _containers.RepeatedCompositeFieldContainer[_common_pb2.Metadata]
    def __init__(self, id: _Optional[int] = ..., assistantId: _Optional[int] = ..., configurationType: _Optional[str] = ..., provider: _Optional[str] = ..., enabled: bool = ..., options: _Optional[_Iterable[_Union[_common_pb2.Metadata, _Mapping]]] = ...) -> None: ...

class DeleteAssistantConfigurationRequest(_message.Message):
    __slots__ = ("id", "assistantId")
    ID_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    id: int
    assistantId: int
    def __init__(self, id: _Optional[int] = ..., assistantId: _Optional[int] = ...) -> None: ...

class GetAllAssistantResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error", "paginated")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    PAGINATED_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: _containers.RepeatedCompositeFieldContainer[Assistant]
    error: _common_pb2.Error
    paginated: _common_pb2.Paginated
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Iterable[_Union[Assistant, _Mapping]]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ..., paginated: _Optional[_Union[_common_pb2.Paginated, _Mapping]] = ...) -> None: ...

class GetAllAssistantMessageRequest(_message.Message):
    __slots__ = ("paginate", "criterias", "assistantId", "order", "selectors")
    PAGINATE_FIELD_NUMBER: _ClassVar[int]
    CRITERIAS_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    ORDER_FIELD_NUMBER: _ClassVar[int]
    SELECTORS_FIELD_NUMBER: _ClassVar[int]
    paginate: _common_pb2.Paginate
    criterias: _containers.RepeatedCompositeFieldContainer[_common_pb2.Criteria]
    assistantId: int
    order: _common_pb2.Ordering
    selectors: _containers.RepeatedCompositeFieldContainer[_common_pb2.FieldSelector]
    def __init__(self, paginate: _Optional[_Union[_common_pb2.Paginate, _Mapping]] = ..., criterias: _Optional[_Iterable[_Union[_common_pb2.Criteria, _Mapping]]] = ..., assistantId: _Optional[int] = ..., order: _Optional[_Union[_common_pb2.Ordering, _Mapping]] = ..., selectors: _Optional[_Iterable[_Union[_common_pb2.FieldSelector, _Mapping]]] = ...) -> None: ...

class GetAllAssistantMessageResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error", "paginated")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    PAGINATED_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: _containers.RepeatedCompositeFieldContainer[_common_pb2.AssistantConversationMessage]
    error: _common_pb2.Error
    paginated: _common_pb2.Paginated
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Iterable[_Union[_common_pb2.AssistantConversationMessage, _Mapping]]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ..., paginated: _Optional[_Union[_common_pb2.Paginated, _Mapping]] = ...) -> None: ...

class GetAllMessageRequest(_message.Message):
    __slots__ = ("paginate", "criterias", "order", "selectors")
    PAGINATE_FIELD_NUMBER: _ClassVar[int]
    CRITERIAS_FIELD_NUMBER: _ClassVar[int]
    ORDER_FIELD_NUMBER: _ClassVar[int]
    SELECTORS_FIELD_NUMBER: _ClassVar[int]
    paginate: _common_pb2.Paginate
    criterias: _containers.RepeatedCompositeFieldContainer[_common_pb2.Criteria]
    order: _common_pb2.Ordering
    selectors: _containers.RepeatedCompositeFieldContainer[_common_pb2.FieldSelector]
    def __init__(self, paginate: _Optional[_Union[_common_pb2.Paginate, _Mapping]] = ..., criterias: _Optional[_Iterable[_Union[_common_pb2.Criteria, _Mapping]]] = ..., order: _Optional[_Union[_common_pb2.Ordering, _Mapping]] = ..., selectors: _Optional[_Iterable[_Union[_common_pb2.FieldSelector, _Mapping]]] = ...) -> None: ...

class GetAllMessageResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error", "paginated")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    PAGINATED_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: _containers.RepeatedCompositeFieldContainer[_common_pb2.AssistantConversationMessage]
    error: _common_pb2.Error
    paginated: _common_pb2.Paginated
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Iterable[_Union[_common_pb2.AssistantConversationMessage, _Mapping]]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ..., paginated: _Optional[_Union[_common_pb2.Paginated, _Mapping]] = ...) -> None: ...

class GetAssistantDashboardRequest(_message.Message):
    __slots__ = ("assistantId", "fromDate", "toDate")
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    FROMDATE_FIELD_NUMBER: _ClassVar[int]
    TODATE_FIELD_NUMBER: _ClassVar[int]
    assistantId: int
    fromDate: _timestamp_pb2.Timestamp
    toDate: _timestamp_pb2.Timestamp
    def __init__(self, assistantId: _Optional[int] = ..., fromDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., toDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetAssistantDashboardResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: AssistantDashboard
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Union[AssistantDashboard, _Mapping]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class AssistantDashboard(_message.Message):
    __slots__ = ("summary", "latency", "usage", "sources", "languages", "buckets")
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    LATENCY_FIELD_NUMBER: _ClassVar[int]
    USAGE_FIELD_NUMBER: _ClassVar[int]
    SOURCES_FIELD_NUMBER: _ClassVar[int]
    LANGUAGES_FIELD_NUMBER: _ClassVar[int]
    BUCKETS_FIELD_NUMBER: _ClassVar[int]
    summary: AssistantDashboardSummary
    latency: AssistantDashboardLatency
    usage: AssistantDashboardUsage
    sources: _containers.RepeatedCompositeFieldContainer[AssistantDashboardDistribution]
    languages: _containers.RepeatedCompositeFieldContainer[AssistantDashboardDistribution]
    buckets: _containers.RepeatedCompositeFieldContainer[AssistantDashboardBucket]
    def __init__(self, summary: _Optional[_Union[AssistantDashboardSummary, _Mapping]] = ..., latency: _Optional[_Union[AssistantDashboardLatency, _Mapping]] = ..., usage: _Optional[_Union[AssistantDashboardUsage, _Mapping]] = ..., sources: _Optional[_Iterable[_Union[AssistantDashboardDistribution, _Mapping]]] = ..., languages: _Optional[_Iterable[_Union[AssistantDashboardDistribution, _Mapping]]] = ..., buckets: _Optional[_Iterable[_Union[AssistantDashboardBucket, _Mapping]]] = ...) -> None: ...

class AssistantDashboardSummary(_message.Message):
    __slots__ = ("totalSessions", "activeSessions", "completedSessions", "failedSessions", "totalMessages", "userMessages", "failureRate", "averageSessionDurationSeconds")
    TOTALSESSIONS_FIELD_NUMBER: _ClassVar[int]
    ACTIVESESSIONS_FIELD_NUMBER: _ClassVar[int]
    COMPLETEDSESSIONS_FIELD_NUMBER: _ClassVar[int]
    FAILEDSESSIONS_FIELD_NUMBER: _ClassVar[int]
    TOTALMESSAGES_FIELD_NUMBER: _ClassVar[int]
    USERMESSAGES_FIELD_NUMBER: _ClassVar[int]
    FAILURERATE_FIELD_NUMBER: _ClassVar[int]
    AVERAGESESSIONDURATIONSECONDS_FIELD_NUMBER: _ClassVar[int]
    totalSessions: int
    activeSessions: int
    completedSessions: int
    failedSessions: int
    totalMessages: int
    userMessages: int
    failureRate: float
    averageSessionDurationSeconds: float
    def __init__(self, totalSessions: _Optional[int] = ..., activeSessions: _Optional[int] = ..., completedSessions: _Optional[int] = ..., failedSessions: _Optional[int] = ..., totalMessages: _Optional[int] = ..., userMessages: _Optional[int] = ..., failureRate: _Optional[float] = ..., averageSessionDurationSeconds: _Optional[float] = ...) -> None: ...

class AssistantDashboardLatency(_message.Message):
    __slots__ = ("averageMs", "sttMs", "eosMs", "ttsMs", "llmMs")
    AVERAGEMS_FIELD_NUMBER: _ClassVar[int]
    STTMS_FIELD_NUMBER: _ClassVar[int]
    EOSMS_FIELD_NUMBER: _ClassVar[int]
    TTSMS_FIELD_NUMBER: _ClassVar[int]
    LLMMS_FIELD_NUMBER: _ClassVar[int]
    averageMs: float
    sttMs: float
    eosMs: float
    ttsMs: float
    llmMs: float
    def __init__(self, averageMs: _Optional[float] = ..., sttMs: _Optional[float] = ..., eosMs: _Optional[float] = ..., ttsMs: _Optional[float] = ..., llmMs: _Optional[float] = ...) -> None: ...

class AssistantDashboardUsage(_message.Message):
    __slots__ = ("totalTokens", "sttDurationSeconds", "ttsDurationSeconds", "totalDurationSeconds")
    TOTALTOKENS_FIELD_NUMBER: _ClassVar[int]
    STTDURATIONSECONDS_FIELD_NUMBER: _ClassVar[int]
    TTSDURATIONSECONDS_FIELD_NUMBER: _ClassVar[int]
    TOTALDURATIONSECONDS_FIELD_NUMBER: _ClassVar[int]
    totalTokens: float
    sttDurationSeconds: float
    ttsDurationSeconds: float
    totalDurationSeconds: float
    def __init__(self, totalTokens: _Optional[float] = ..., sttDurationSeconds: _Optional[float] = ..., ttsDurationSeconds: _Optional[float] = ..., totalDurationSeconds: _Optional[float] = ...) -> None: ...

class AssistantDashboardDistribution(_message.Message):
    __slots__ = ("name", "count", "percentage")
    NAME_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    PERCENTAGE_FIELD_NUMBER: _ClassVar[int]
    name: str
    count: int
    percentage: float
    def __init__(self, name: _Optional[str] = ..., count: _Optional[int] = ..., percentage: _Optional[float] = ...) -> None: ...

class AssistantDashboardBucket(_message.Message):
    __slots__ = ("startDate", "endDate", "messageCount", "sttLatencyMs", "eosLatencyMs", "ttsLatencyMs", "llmLatencyMs")
    STARTDATE_FIELD_NUMBER: _ClassVar[int]
    ENDDATE_FIELD_NUMBER: _ClassVar[int]
    MESSAGECOUNT_FIELD_NUMBER: _ClassVar[int]
    STTLATENCYMS_FIELD_NUMBER: _ClassVar[int]
    EOSLATENCYMS_FIELD_NUMBER: _ClassVar[int]
    TTSLATENCYMS_FIELD_NUMBER: _ClassVar[int]
    LLMLATENCYMS_FIELD_NUMBER: _ClassVar[int]
    startDate: _timestamp_pb2.Timestamp
    endDate: _timestamp_pb2.Timestamp
    messageCount: int
    sttLatencyMs: float
    eosLatencyMs: float
    ttsLatencyMs: float
    llmLatencyMs: float
    def __init__(self, startDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., endDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., messageCount: _Optional[int] = ..., sttLatencyMs: _Optional[float] = ..., eosLatencyMs: _Optional[float] = ..., ttsLatencyMs: _Optional[float] = ..., llmLatencyMs: _Optional[float] = ...) -> None: ...

class UpdateAssistantDetailRequest(_message.Message):
    __slots__ = ("assistantId", "name", "description")
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    assistantId: int
    name: str
    description: str
    def __init__(self, assistantId: _Optional[int] = ..., name: _Optional[str] = ..., description: _Optional[str] = ...) -> None: ...

class GetAssistantConversationRequest(_message.Message):
    __slots__ = ("assistantId", "id", "selectors")
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    SELECTORS_FIELD_NUMBER: _ClassVar[int]
    assistantId: int
    id: int
    selectors: _containers.RepeatedCompositeFieldContainer[_common_pb2.FieldSelector]
    def __init__(self, assistantId: _Optional[int] = ..., id: _Optional[int] = ..., selectors: _Optional[_Iterable[_Union[_common_pb2.FieldSelector, _Mapping]]] = ...) -> None: ...

class GetAssistantConversationResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: _common_pb2.AssistantConversation
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Union[_common_pb2.AssistantConversation, _Mapping]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...
