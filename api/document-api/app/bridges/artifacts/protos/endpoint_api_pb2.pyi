import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
import app.bridges.artifacts.protos.common_pb2 as _common_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class EndpointAttribute(_message.Message):
    __slots__ = ("source", "sourceIdentifier", "visibility", "language", "name", "description")
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    SOURCEIDENTIFIER_FIELD_NUMBER: _ClassVar[int]
    VISIBILITY_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    source: str
    sourceIdentifier: int
    visibility: str
    language: str
    name: str
    description: str
    def __init__(self, source: _Optional[str] = ..., sourceIdentifier: _Optional[int] = ..., visibility: _Optional[str] = ..., language: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ...) -> None: ...

class EndpointProviderModelAttribute(_message.Message):
    __slots__ = ("description", "chatCompletePrompt", "modelProviderName", "endpointModelOptions")
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CHATCOMPLETEPROMPT_FIELD_NUMBER: _ClassVar[int]
    MODELPROVIDERNAME_FIELD_NUMBER: _ClassVar[int]
    ENDPOINTMODELOPTIONS_FIELD_NUMBER: _ClassVar[int]
    description: str
    chatCompletePrompt: _common_pb2.TextChatCompletePrompt
    modelProviderName: str
    endpointModelOptions: _containers.RepeatedCompositeFieldContainer[_common_pb2.Metadata]
    def __init__(self, description: _Optional[str] = ..., chatCompletePrompt: _Optional[_Union[_common_pb2.TextChatCompletePrompt, _Mapping]] = ..., modelProviderName: _Optional[str] = ..., endpointModelOptions: _Optional[_Iterable[_Union[_common_pb2.Metadata, _Mapping]]] = ...) -> None: ...

class CreateEndpointRequest(_message.Message):
    __slots__ = ("endpointProviderModelAttribute", "endpointAttribute", "retryConfiguration", "cacheConfiguration", "tags")
    ENDPOINTPROVIDERMODELATTRIBUTE_FIELD_NUMBER: _ClassVar[int]
    ENDPOINTATTRIBUTE_FIELD_NUMBER: _ClassVar[int]
    RETRYCONFIGURATION_FIELD_NUMBER: _ClassVar[int]
    CACHECONFIGURATION_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    endpointProviderModelAttribute: EndpointProviderModelAttribute
    endpointAttribute: EndpointAttribute
    retryConfiguration: EndpointRetryConfiguration
    cacheConfiguration: EndpointCacheConfiguration
    tags: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, endpointProviderModelAttribute: _Optional[_Union[EndpointProviderModelAttribute, _Mapping]] = ..., endpointAttribute: _Optional[_Union[EndpointAttribute, _Mapping]] = ..., retryConfiguration: _Optional[_Union[EndpointRetryConfiguration, _Mapping]] = ..., cacheConfiguration: _Optional[_Union[EndpointCacheConfiguration, _Mapping]] = ..., tags: _Optional[_Iterable[str]] = ...) -> None: ...

class CreateEndpointResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: Endpoint
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Union[Endpoint, _Mapping]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class EndpointProviderModel(_message.Message):
    __slots__ = ("id", "chatCompletePrompt", "modelProviderName", "endpointModelOptions", "status", "createdActor", "updatedActor", "createdDate", "updatedDate", "endpointId", "description")
    ID_FIELD_NUMBER: _ClassVar[int]
    CHATCOMPLETEPROMPT_FIELD_NUMBER: _ClassVar[int]
    MODELPROVIDERNAME_FIELD_NUMBER: _ClassVar[int]
    ENDPOINTMODELOPTIONS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CREATEDACTOR_FIELD_NUMBER: _ClassVar[int]
    UPDATEDACTOR_FIELD_NUMBER: _ClassVar[int]
    CREATEDDATE_FIELD_NUMBER: _ClassVar[int]
    UPDATEDDATE_FIELD_NUMBER: _ClassVar[int]
    ENDPOINTID_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    id: int
    chatCompletePrompt: _common_pb2.TextChatCompletePrompt
    modelProviderName: str
    endpointModelOptions: _containers.RepeatedCompositeFieldContainer[_common_pb2.Metadata]
    status: str
    createdActor: _common_pb2.AuditActor
    updatedActor: _common_pb2.AuditActor
    createdDate: _timestamp_pb2.Timestamp
    updatedDate: _timestamp_pb2.Timestamp
    endpointId: int
    description: str
    def __init__(self, id: _Optional[int] = ..., chatCompletePrompt: _Optional[_Union[_common_pb2.TextChatCompletePrompt, _Mapping]] = ..., modelProviderName: _Optional[str] = ..., endpointModelOptions: _Optional[_Iterable[_Union[_common_pb2.Metadata, _Mapping]]] = ..., status: _Optional[str] = ..., createdActor: _Optional[_Union[_common_pb2.AuditActor, _Mapping]] = ..., updatedActor: _Optional[_Union[_common_pb2.AuditActor, _Mapping]] = ..., createdDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updatedDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., endpointId: _Optional[int] = ..., description: _Optional[str] = ...) -> None: ...

class AggregatedEndpointAnalytics(_message.Message):
    __slots__ = ("count", "totalInputCost", "totalOutputCost", "totalToken", "successCount", "errorCount", "p50Latency", "p99Latency", "lastActivity")
    COUNT_FIELD_NUMBER: _ClassVar[int]
    TOTALINPUTCOST_FIELD_NUMBER: _ClassVar[int]
    TOTALOUTPUTCOST_FIELD_NUMBER: _ClassVar[int]
    TOTALTOKEN_FIELD_NUMBER: _ClassVar[int]
    SUCCESSCOUNT_FIELD_NUMBER: _ClassVar[int]
    ERRORCOUNT_FIELD_NUMBER: _ClassVar[int]
    P50LATENCY_FIELD_NUMBER: _ClassVar[int]
    P99LATENCY_FIELD_NUMBER: _ClassVar[int]
    LASTACTIVITY_FIELD_NUMBER: _ClassVar[int]
    count: int
    totalInputCost: float
    totalOutputCost: float
    totalToken: int
    successCount: int
    errorCount: int
    p50Latency: float
    p99Latency: float
    lastActivity: _timestamp_pb2.Timestamp
    def __init__(self, count: _Optional[int] = ..., totalInputCost: _Optional[float] = ..., totalOutputCost: _Optional[float] = ..., totalToken: _Optional[int] = ..., successCount: _Optional[int] = ..., errorCount: _Optional[int] = ..., p50Latency: _Optional[float] = ..., p99Latency: _Optional[float] = ..., lastActivity: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class Endpoint(_message.Message):
    __slots__ = ("id", "status", "visibility", "source", "sourceIdentifier", "projectId", "organizationId", "endpointProviderModelId", "endpointProviderModel", "endpointAnalytics", "endpointRetry", "endpointCaching", "endpointTag", "language", "organization", "name", "description", "createdDate", "updatedDate", "createdActor", "updatedActor")
    ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    VISIBILITY_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    SOURCEIDENTIFIER_FIELD_NUMBER: _ClassVar[int]
    PROJECTID_FIELD_NUMBER: _ClassVar[int]
    ORGANIZATIONID_FIELD_NUMBER: _ClassVar[int]
    ENDPOINTPROVIDERMODELID_FIELD_NUMBER: _ClassVar[int]
    ENDPOINTPROVIDERMODEL_FIELD_NUMBER: _ClassVar[int]
    ENDPOINTANALYTICS_FIELD_NUMBER: _ClassVar[int]
    ENDPOINTRETRY_FIELD_NUMBER: _ClassVar[int]
    ENDPOINTCACHING_FIELD_NUMBER: _ClassVar[int]
    ENDPOINTTAG_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    ORGANIZATION_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CREATEDDATE_FIELD_NUMBER: _ClassVar[int]
    UPDATEDDATE_FIELD_NUMBER: _ClassVar[int]
    CREATEDACTOR_FIELD_NUMBER: _ClassVar[int]
    UPDATEDACTOR_FIELD_NUMBER: _ClassVar[int]
    id: int
    status: str
    visibility: str
    source: str
    sourceIdentifier: int
    projectId: int
    organizationId: int
    endpointProviderModelId: int
    endpointProviderModel: EndpointProviderModel
    endpointAnalytics: AggregatedEndpointAnalytics
    endpointRetry: EndpointRetryConfiguration
    endpointCaching: EndpointCacheConfiguration
    endpointTag: _common_pb2.Tag
    language: str
    organization: _common_pb2.Organization
    name: str
    description: str
    createdDate: _timestamp_pb2.Timestamp
    updatedDate: _timestamp_pb2.Timestamp
    createdActor: _common_pb2.AuditActor
    updatedActor: _common_pb2.AuditActor
    def __init__(self, id: _Optional[int] = ..., status: _Optional[str] = ..., visibility: _Optional[str] = ..., source: _Optional[str] = ..., sourceIdentifier: _Optional[int] = ..., projectId: _Optional[int] = ..., organizationId: _Optional[int] = ..., endpointProviderModelId: _Optional[int] = ..., endpointProviderModel: _Optional[_Union[EndpointProviderModel, _Mapping]] = ..., endpointAnalytics: _Optional[_Union[AggregatedEndpointAnalytics, _Mapping]] = ..., endpointRetry: _Optional[_Union[EndpointRetryConfiguration, _Mapping]] = ..., endpointCaching: _Optional[_Union[EndpointCacheConfiguration, _Mapping]] = ..., endpointTag: _Optional[_Union[_common_pb2.Tag, _Mapping]] = ..., language: _Optional[str] = ..., organization: _Optional[_Union[_common_pb2.Organization, _Mapping]] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., createdDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updatedDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., createdActor: _Optional[_Union[_common_pb2.AuditActor, _Mapping]] = ..., updatedActor: _Optional[_Union[_common_pb2.AuditActor, _Mapping]] = ...) -> None: ...

class CreateEndpointProviderModelRequest(_message.Message):
    __slots__ = ("endpointId", "endpointProviderModelAttribute")
    ENDPOINTID_FIELD_NUMBER: _ClassVar[int]
    ENDPOINTPROVIDERMODELATTRIBUTE_FIELD_NUMBER: _ClassVar[int]
    endpointId: int
    endpointProviderModelAttribute: EndpointProviderModelAttribute
    def __init__(self, endpointId: _Optional[int] = ..., endpointProviderModelAttribute: _Optional[_Union[EndpointProviderModelAttribute, _Mapping]] = ...) -> None: ...

class CreateEndpointProviderModelResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: EndpointProviderModel
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Union[EndpointProviderModel, _Mapping]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class GetEndpointRequest(_message.Message):
    __slots__ = ("id", "endpointProviderModelId")
    ID_FIELD_NUMBER: _ClassVar[int]
    ENDPOINTPROVIDERMODELID_FIELD_NUMBER: _ClassVar[int]
    id: int
    endpointProviderModelId: int
    def __init__(self, id: _Optional[int] = ..., endpointProviderModelId: _Optional[int] = ...) -> None: ...

class GetEndpointResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: Endpoint
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Union[Endpoint, _Mapping]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class GetAllEndpointRequest(_message.Message):
    __slots__ = ("paginate", "criterias")
    PAGINATE_FIELD_NUMBER: _ClassVar[int]
    CRITERIAS_FIELD_NUMBER: _ClassVar[int]
    paginate: _common_pb2.Paginate
    criterias: _containers.RepeatedCompositeFieldContainer[_common_pb2.Criteria]
    def __init__(self, paginate: _Optional[_Union[_common_pb2.Paginate, _Mapping]] = ..., criterias: _Optional[_Iterable[_Union[_common_pb2.Criteria, _Mapping]]] = ...) -> None: ...

class GetAllEndpointResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error", "paginated")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    PAGINATED_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: _containers.RepeatedCompositeFieldContainer[Endpoint]
    error: _common_pb2.Error
    paginated: _common_pb2.Paginated
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Iterable[_Union[Endpoint, _Mapping]]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ..., paginated: _Optional[_Union[_common_pb2.Paginated, _Mapping]] = ...) -> None: ...

class GetAllEndpointProviderModelRequest(_message.Message):
    __slots__ = ("paginate", "criterias", "endpointId")
    PAGINATE_FIELD_NUMBER: _ClassVar[int]
    CRITERIAS_FIELD_NUMBER: _ClassVar[int]
    ENDPOINTID_FIELD_NUMBER: _ClassVar[int]
    paginate: _common_pb2.Paginate
    criterias: _containers.RepeatedCompositeFieldContainer[_common_pb2.Criteria]
    endpointId: int
    def __init__(self, paginate: _Optional[_Union[_common_pb2.Paginate, _Mapping]] = ..., criterias: _Optional[_Iterable[_Union[_common_pb2.Criteria, _Mapping]]] = ..., endpointId: _Optional[int] = ...) -> None: ...

class GetAllEndpointProviderModelResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error", "paginated")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    PAGINATED_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: _containers.RepeatedCompositeFieldContainer[EndpointProviderModel]
    error: _common_pb2.Error
    paginated: _common_pb2.Paginated
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Iterable[_Union[EndpointProviderModel, _Mapping]]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ..., paginated: _Optional[_Union[_common_pb2.Paginated, _Mapping]] = ...) -> None: ...

class UpdateEndpointVersionRequest(_message.Message):
    __slots__ = ("endpointId", "endpointProviderModelId")
    ENDPOINTID_FIELD_NUMBER: _ClassVar[int]
    ENDPOINTPROVIDERMODELID_FIELD_NUMBER: _ClassVar[int]
    endpointId: int
    endpointProviderModelId: int
    def __init__(self, endpointId: _Optional[int] = ..., endpointProviderModelId: _Optional[int] = ...) -> None: ...

class UpdateEndpointVersionResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: Endpoint
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Union[Endpoint, _Mapping]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class EndpointRetryConfiguration(_message.Message):
    __slots__ = ("retryType", "maxAttempts", "delaySeconds", "exponentialBackoff", "retryables", "createdActor", "updatedActor")
    RETRYTYPE_FIELD_NUMBER: _ClassVar[int]
    MAXATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    DELAYSECONDS_FIELD_NUMBER: _ClassVar[int]
    EXPONENTIALBACKOFF_FIELD_NUMBER: _ClassVar[int]
    RETRYABLES_FIELD_NUMBER: _ClassVar[int]
    CREATEDACTOR_FIELD_NUMBER: _ClassVar[int]
    UPDATEDACTOR_FIELD_NUMBER: _ClassVar[int]
    retryType: str
    maxAttempts: int
    delaySeconds: int
    exponentialBackoff: bool
    retryables: _containers.RepeatedScalarFieldContainer[str]
    createdActor: _common_pb2.AuditActor
    updatedActor: _common_pb2.AuditActor
    def __init__(self, retryType: _Optional[str] = ..., maxAttempts: _Optional[int] = ..., delaySeconds: _Optional[int] = ..., exponentialBackoff: bool = ..., retryables: _Optional[_Iterable[str]] = ..., createdActor: _Optional[_Union[_common_pb2.AuditActor, _Mapping]] = ..., updatedActor: _Optional[_Union[_common_pb2.AuditActor, _Mapping]] = ...) -> None: ...

class EndpointCacheConfiguration(_message.Message):
    __slots__ = ("cacheType", "expiryInterval", "matchThreshold", "createdActor", "updatedActor")
    CACHETYPE_FIELD_NUMBER: _ClassVar[int]
    EXPIRYINTERVAL_FIELD_NUMBER: _ClassVar[int]
    MATCHTHRESHOLD_FIELD_NUMBER: _ClassVar[int]
    CREATEDACTOR_FIELD_NUMBER: _ClassVar[int]
    UPDATEDACTOR_FIELD_NUMBER: _ClassVar[int]
    cacheType: str
    expiryInterval: int
    matchThreshold: float
    createdActor: _common_pb2.AuditActor
    updatedActor: _common_pb2.AuditActor
    def __init__(self, cacheType: _Optional[str] = ..., expiryInterval: _Optional[int] = ..., matchThreshold: _Optional[float] = ..., createdActor: _Optional[_Union[_common_pb2.AuditActor, _Mapping]] = ..., updatedActor: _Optional[_Union[_common_pb2.AuditActor, _Mapping]] = ...) -> None: ...

class CreateEndpointRetryConfigurationRequest(_message.Message):
    __slots__ = ("endpointId", "data")
    ENDPOINTID_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    endpointId: int
    data: EndpointRetryConfiguration
    def __init__(self, endpointId: _Optional[int] = ..., data: _Optional[_Union[EndpointRetryConfiguration, _Mapping]] = ...) -> None: ...

class CreateEndpointRetryConfigurationResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: EndpointRetryConfiguration
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Union[EndpointRetryConfiguration, _Mapping]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class CreateEndpointCacheConfigurationRequest(_message.Message):
    __slots__ = ("endpointId", "data")
    ENDPOINTID_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    endpointId: int
    data: EndpointCacheConfiguration
    def __init__(self, endpointId: _Optional[int] = ..., data: _Optional[_Union[EndpointCacheConfiguration, _Mapping]] = ...) -> None: ...

class CreateEndpointCacheConfigurationResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: EndpointCacheConfiguration
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Union[EndpointCacheConfiguration, _Mapping]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class CreateEndpointTagRequest(_message.Message):
    __slots__ = ("endpointId", "tags")
    ENDPOINTID_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    endpointId: int
    tags: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, endpointId: _Optional[int] = ..., tags: _Optional[_Iterable[str]] = ...) -> None: ...

class ForkEndpointRequest(_message.Message):
    __slots__ = ("endpointId", "endpointProviderId")
    ENDPOINTID_FIELD_NUMBER: _ClassVar[int]
    ENDPOINTPROVIDERID_FIELD_NUMBER: _ClassVar[int]
    endpointId: int
    endpointProviderId: int
    def __init__(self, endpointId: _Optional[int] = ..., endpointProviderId: _Optional[int] = ...) -> None: ...

class UpdateEndpointDetailRequest(_message.Message):
    __slots__ = ("endpointId", "name", "description")
    ENDPOINTID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    endpointId: int
    name: str
    description: str
    def __init__(self, endpointId: _Optional[int] = ..., name: _Optional[str] = ..., description: _Optional[str] = ...) -> None: ...

class EndpointLog(_message.Message):
    __slots__ = ("id", "endpointId", "source", "status", "projectId", "organizationId", "endpointProviderModelId", "timeTaken", "createdDate", "updatedDate", "metrics", "metadata", "arguments", "options")
    ID_FIELD_NUMBER: _ClassVar[int]
    ENDPOINTID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PROJECTID_FIELD_NUMBER: _ClassVar[int]
    ORGANIZATIONID_FIELD_NUMBER: _ClassVar[int]
    ENDPOINTPROVIDERMODELID_FIELD_NUMBER: _ClassVar[int]
    TIMETAKEN_FIELD_NUMBER: _ClassVar[int]
    CREATEDDATE_FIELD_NUMBER: _ClassVar[int]
    UPDATEDDATE_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    ARGUMENTS_FIELD_NUMBER: _ClassVar[int]
    OPTIONS_FIELD_NUMBER: _ClassVar[int]
    id: int
    endpointId: int
    source: str
    status: str
    projectId: int
    organizationId: int
    endpointProviderModelId: int
    timeTaken: int
    createdDate: _timestamp_pb2.Timestamp
    updatedDate: _timestamp_pb2.Timestamp
    metrics: _containers.RepeatedCompositeFieldContainer[_common_pb2.Metric]
    metadata: _containers.RepeatedCompositeFieldContainer[_common_pb2.Metadata]
    arguments: _containers.RepeatedCompositeFieldContainer[_common_pb2.Argument]
    options: _containers.RepeatedCompositeFieldContainer[_common_pb2.Metadata]
    def __init__(self, id: _Optional[int] = ..., endpointId: _Optional[int] = ..., source: _Optional[str] = ..., status: _Optional[str] = ..., projectId: _Optional[int] = ..., organizationId: _Optional[int] = ..., endpointProviderModelId: _Optional[int] = ..., timeTaken: _Optional[int] = ..., createdDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updatedDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., metrics: _Optional[_Iterable[_Union[_common_pb2.Metric, _Mapping]]] = ..., metadata: _Optional[_Iterable[_Union[_common_pb2.Metadata, _Mapping]]] = ..., arguments: _Optional[_Iterable[_Union[_common_pb2.Argument, _Mapping]]] = ..., options: _Optional[_Iterable[_Union[_common_pb2.Metadata, _Mapping]]] = ...) -> None: ...

class GetAllEndpointLogRequest(_message.Message):
    __slots__ = ("paginate", "criterias", "endpointId")
    PAGINATE_FIELD_NUMBER: _ClassVar[int]
    CRITERIAS_FIELD_NUMBER: _ClassVar[int]
    ENDPOINTID_FIELD_NUMBER: _ClassVar[int]
    paginate: _common_pb2.Paginate
    criterias: _containers.RepeatedCompositeFieldContainer[_common_pb2.Criteria]
    endpointId: int
    def __init__(self, paginate: _Optional[_Union[_common_pb2.Paginate, _Mapping]] = ..., criterias: _Optional[_Iterable[_Union[_common_pb2.Criteria, _Mapping]]] = ..., endpointId: _Optional[int] = ...) -> None: ...

class GetAllEndpointLogResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error", "paginated")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    PAGINATED_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: _containers.RepeatedCompositeFieldContainer[EndpointLog]
    error: _common_pb2.Error
    paginated: _common_pb2.Paginated
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Iterable[_Union[EndpointLog, _Mapping]]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ..., paginated: _Optional[_Union[_common_pb2.Paginated, _Mapping]] = ...) -> None: ...

class GetEndpointLogRequest(_message.Message):
    __slots__ = ("endpointId", "id")
    ENDPOINTID_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    endpointId: int
    id: int
    def __init__(self, endpointId: _Optional[int] = ..., id: _Optional[int] = ...) -> None: ...

class GetEndpointLogResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: EndpointLog
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Union[EndpointLog, _Mapping]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...
