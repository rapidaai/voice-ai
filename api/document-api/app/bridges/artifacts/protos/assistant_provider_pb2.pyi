import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf import struct_pb2 as _struct_pb2
import app.bridges.artifacts.protos.common_pb2 as _common_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class UpdateAssistantVersionRequest(_message.Message):
    __slots__ = ("assistantId", "assistantProviderId", "assistantProvider")
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTPROVIDERID_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTPROVIDER_FIELD_NUMBER: _ClassVar[int]
    assistantId: int
    assistantProviderId: int
    assistantProvider: str
    def __init__(self, assistantId: _Optional[int] = ..., assistantProviderId: _Optional[int] = ..., assistantProvider: _Optional[str] = ...) -> None: ...

class CreateAssistantProviderRequest(_message.Message):
    __slots__ = ("assistantId", "description", "model", "agentkit", "websocket", "agentflow")
    class CreateAssistantProviderModel(_message.Message):
        __slots__ = ("template", "modelProviderName", "assistantModelOptions")
        TEMPLATE_FIELD_NUMBER: _ClassVar[int]
        MODELPROVIDERNAME_FIELD_NUMBER: _ClassVar[int]
        ASSISTANTMODELOPTIONS_FIELD_NUMBER: _ClassVar[int]
        template: _common_pb2.TextChatCompletePrompt
        modelProviderName: str
        assistantModelOptions: _containers.RepeatedCompositeFieldContainer[_common_pb2.Metadata]
        def __init__(self, template: _Optional[_Union[_common_pb2.TextChatCompletePrompt, _Mapping]] = ..., modelProviderName: _Optional[str] = ..., assistantModelOptions: _Optional[_Iterable[_Union[_common_pb2.Metadata, _Mapping]]] = ...) -> None: ...
    class CreateAssistantProviderAgentkit(_message.Message):
        __slots__ = ("agentKitUrl", "certificate", "metadata", "transportSecurity", "tlsVerification", "tlsServerName", "connectTimeoutMs", "keepaliveTimeMs", "keepaliveTimeoutMs", "maxRecvMessageBytes", "maxSendMessageBytes")
        class MetadataEntry(_message.Message):
            __slots__ = ("key", "value")
            KEY_FIELD_NUMBER: _ClassVar[int]
            VALUE_FIELD_NUMBER: _ClassVar[int]
            key: str
            value: str
            def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
        AGENTKITURL_FIELD_NUMBER: _ClassVar[int]
        CERTIFICATE_FIELD_NUMBER: _ClassVar[int]
        METADATA_FIELD_NUMBER: _ClassVar[int]
        TRANSPORTSECURITY_FIELD_NUMBER: _ClassVar[int]
        TLSVERIFICATION_FIELD_NUMBER: _ClassVar[int]
        TLSSERVERNAME_FIELD_NUMBER: _ClassVar[int]
        CONNECTTIMEOUTMS_FIELD_NUMBER: _ClassVar[int]
        KEEPALIVETIMEMS_FIELD_NUMBER: _ClassVar[int]
        KEEPALIVETIMEOUTMS_FIELD_NUMBER: _ClassVar[int]
        MAXRECVMESSAGEBYTES_FIELD_NUMBER: _ClassVar[int]
        MAXSENDMESSAGEBYTES_FIELD_NUMBER: _ClassVar[int]
        agentKitUrl: str
        certificate: str
        metadata: _containers.ScalarMap[str, str]
        transportSecurity: str
        tlsVerification: str
        tlsServerName: str
        connectTimeoutMs: int
        keepaliveTimeMs: int
        keepaliveTimeoutMs: int
        maxRecvMessageBytes: int
        maxSendMessageBytes: int
        def __init__(self, agentKitUrl: _Optional[str] = ..., certificate: _Optional[str] = ..., metadata: _Optional[_Mapping[str, str]] = ..., transportSecurity: _Optional[str] = ..., tlsVerification: _Optional[str] = ..., tlsServerName: _Optional[str] = ..., connectTimeoutMs: _Optional[int] = ..., keepaliveTimeMs: _Optional[int] = ..., keepaliveTimeoutMs: _Optional[int] = ..., maxRecvMessageBytes: _Optional[int] = ..., maxSendMessageBytes: _Optional[int] = ...) -> None: ...
    class CreateAssistantProviderWebsocket(_message.Message):
        __slots__ = ("websocketUrl", "headers", "connectionParameters")
        class HeadersEntry(_message.Message):
            __slots__ = ("key", "value")
            KEY_FIELD_NUMBER: _ClassVar[int]
            VALUE_FIELD_NUMBER: _ClassVar[int]
            key: str
            value: str
            def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
        class ConnectionParametersEntry(_message.Message):
            __slots__ = ("key", "value")
            KEY_FIELD_NUMBER: _ClassVar[int]
            VALUE_FIELD_NUMBER: _ClassVar[int]
            key: str
            value: str
            def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
        WEBSOCKETURL_FIELD_NUMBER: _ClassVar[int]
        HEADERS_FIELD_NUMBER: _ClassVar[int]
        CONNECTIONPARAMETERS_FIELD_NUMBER: _ClassVar[int]
        websocketUrl: str
        headers: _containers.ScalarMap[str, str]
        connectionParameters: _containers.ScalarMap[str, str]
        def __init__(self, websocketUrl: _Optional[str] = ..., headers: _Optional[_Mapping[str, str]] = ..., connectionParameters: _Optional[_Mapping[str, str]] = ...) -> None: ...
    class CreateAssistantProviderAgentflow(_message.Message):
        __slots__ = ("schemaVersion", "definition")
        SCHEMAVERSION_FIELD_NUMBER: _ClassVar[int]
        DEFINITION_FIELD_NUMBER: _ClassVar[int]
        schemaVersion: str
        definition: _struct_pb2.Struct
        def __init__(self, schemaVersion: _Optional[str] = ..., definition: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    AGENTKIT_FIELD_NUMBER: _ClassVar[int]
    WEBSOCKET_FIELD_NUMBER: _ClassVar[int]
    AGENTFLOW_FIELD_NUMBER: _ClassVar[int]
    assistantId: int
    description: str
    model: CreateAssistantProviderRequest.CreateAssistantProviderModel
    agentkit: CreateAssistantProviderRequest.CreateAssistantProviderAgentkit
    websocket: CreateAssistantProviderRequest.CreateAssistantProviderWebsocket
    agentflow: CreateAssistantProviderRequest.CreateAssistantProviderAgentflow
    def __init__(self, assistantId: _Optional[int] = ..., description: _Optional[str] = ..., model: _Optional[_Union[CreateAssistantProviderRequest.CreateAssistantProviderModel, _Mapping]] = ..., agentkit: _Optional[_Union[CreateAssistantProviderRequest.CreateAssistantProviderAgentkit, _Mapping]] = ..., websocket: _Optional[_Union[CreateAssistantProviderRequest.CreateAssistantProviderWebsocket, _Mapping]] = ..., agentflow: _Optional[_Union[CreateAssistantProviderRequest.CreateAssistantProviderAgentflow, _Mapping]] = ...) -> None: ...

class AssistantProviderAgentkit(_message.Message):
    __slots__ = ("id", "description", "assistantId", "status", "url", "certificate", "metadata", "createdActor", "updatedActor", "createdDate", "updatedDate", "transportSecurity", "tlsVerification", "tlsServerName", "connectTimeoutMs", "keepaliveTimeMs", "keepaliveTimeoutMs", "maxRecvMessageBytes", "maxSendMessageBytes")
    class MetadataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    CERTIFICATE_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    CREATEDACTOR_FIELD_NUMBER: _ClassVar[int]
    UPDATEDACTOR_FIELD_NUMBER: _ClassVar[int]
    CREATEDDATE_FIELD_NUMBER: _ClassVar[int]
    UPDATEDDATE_FIELD_NUMBER: _ClassVar[int]
    TRANSPORTSECURITY_FIELD_NUMBER: _ClassVar[int]
    TLSVERIFICATION_FIELD_NUMBER: _ClassVar[int]
    TLSSERVERNAME_FIELD_NUMBER: _ClassVar[int]
    CONNECTTIMEOUTMS_FIELD_NUMBER: _ClassVar[int]
    KEEPALIVETIMEMS_FIELD_NUMBER: _ClassVar[int]
    KEEPALIVETIMEOUTMS_FIELD_NUMBER: _ClassVar[int]
    MAXRECVMESSAGEBYTES_FIELD_NUMBER: _ClassVar[int]
    MAXSENDMESSAGEBYTES_FIELD_NUMBER: _ClassVar[int]
    id: int
    description: str
    assistantId: int
    status: str
    url: str
    certificate: str
    metadata: _containers.ScalarMap[str, str]
    createdActor: _common_pb2.AuditActor
    updatedActor: _common_pb2.AuditActor
    createdDate: _timestamp_pb2.Timestamp
    updatedDate: _timestamp_pb2.Timestamp
    transportSecurity: str
    tlsVerification: str
    tlsServerName: str
    connectTimeoutMs: int
    keepaliveTimeMs: int
    keepaliveTimeoutMs: int
    maxRecvMessageBytes: int
    maxSendMessageBytes: int
    def __init__(self, id: _Optional[int] = ..., description: _Optional[str] = ..., assistantId: _Optional[int] = ..., status: _Optional[str] = ..., url: _Optional[str] = ..., certificate: _Optional[str] = ..., metadata: _Optional[_Mapping[str, str]] = ..., createdActor: _Optional[_Union[_common_pb2.AuditActor, _Mapping]] = ..., updatedActor: _Optional[_Union[_common_pb2.AuditActor, _Mapping]] = ..., createdDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updatedDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., transportSecurity: _Optional[str] = ..., tlsVerification: _Optional[str] = ..., tlsServerName: _Optional[str] = ..., connectTimeoutMs: _Optional[int] = ..., keepaliveTimeMs: _Optional[int] = ..., keepaliveTimeoutMs: _Optional[int] = ..., maxRecvMessageBytes: _Optional[int] = ..., maxSendMessageBytes: _Optional[int] = ...) -> None: ...

class AssistantProviderWebsocket(_message.Message):
    __slots__ = ("id", "description", "assistantId", "url", "headers", "parameters", "status", "createdActor", "updatedActor", "createdDate", "updatedDate")
    class HeadersEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    class ParametersEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    HEADERS_FIELD_NUMBER: _ClassVar[int]
    PARAMETERS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CREATEDACTOR_FIELD_NUMBER: _ClassVar[int]
    UPDATEDACTOR_FIELD_NUMBER: _ClassVar[int]
    CREATEDDATE_FIELD_NUMBER: _ClassVar[int]
    UPDATEDDATE_FIELD_NUMBER: _ClassVar[int]
    id: int
    description: str
    assistantId: int
    url: str
    headers: _containers.ScalarMap[str, str]
    parameters: _containers.ScalarMap[str, str]
    status: str
    createdActor: _common_pb2.AuditActor
    updatedActor: _common_pb2.AuditActor
    createdDate: _timestamp_pb2.Timestamp
    updatedDate: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[int] = ..., description: _Optional[str] = ..., assistantId: _Optional[int] = ..., url: _Optional[str] = ..., headers: _Optional[_Mapping[str, str]] = ..., parameters: _Optional[_Mapping[str, str]] = ..., status: _Optional[str] = ..., createdActor: _Optional[_Union[_common_pb2.AuditActor, _Mapping]] = ..., updatedActor: _Optional[_Union[_common_pb2.AuditActor, _Mapping]] = ..., createdDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updatedDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class AssistantProviderModel(_message.Message):
    __slots__ = ("id", "template", "description", "assistantId", "modelProviderName", "assistantModelOptions", "status", "createdActor", "updatedActor", "createdDate", "updatedDate")
    ID_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    MODELPROVIDERNAME_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTMODELOPTIONS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CREATEDACTOR_FIELD_NUMBER: _ClassVar[int]
    UPDATEDACTOR_FIELD_NUMBER: _ClassVar[int]
    CREATEDDATE_FIELD_NUMBER: _ClassVar[int]
    UPDATEDDATE_FIELD_NUMBER: _ClassVar[int]
    id: int
    template: _common_pb2.TextChatCompletePrompt
    description: str
    assistantId: int
    modelProviderName: str
    assistantModelOptions: _containers.RepeatedCompositeFieldContainer[_common_pb2.Metadata]
    status: str
    createdActor: _common_pb2.AuditActor
    updatedActor: _common_pb2.AuditActor
    createdDate: _timestamp_pb2.Timestamp
    updatedDate: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[int] = ..., template: _Optional[_Union[_common_pb2.TextChatCompletePrompt, _Mapping]] = ..., description: _Optional[str] = ..., assistantId: _Optional[int] = ..., modelProviderName: _Optional[str] = ..., assistantModelOptions: _Optional[_Iterable[_Union[_common_pb2.Metadata, _Mapping]]] = ..., status: _Optional[str] = ..., createdActor: _Optional[_Union[_common_pb2.AuditActor, _Mapping]] = ..., updatedActor: _Optional[_Union[_common_pb2.AuditActor, _Mapping]] = ..., createdDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updatedDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class AssistantProviderAgentflow(_message.Message):
    __slots__ = ("id", "description", "assistantId", "schemaVersion", "definition", "status", "createdActor", "updatedActor", "createdDate", "updatedDate")
    ID_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    SCHEMAVERSION_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CREATEDACTOR_FIELD_NUMBER: _ClassVar[int]
    UPDATEDACTOR_FIELD_NUMBER: _ClassVar[int]
    CREATEDDATE_FIELD_NUMBER: _ClassVar[int]
    UPDATEDDATE_FIELD_NUMBER: _ClassVar[int]
    id: int
    description: str
    assistantId: int
    schemaVersion: str
    definition: _struct_pb2.Struct
    status: str
    createdActor: _common_pb2.AuditActor
    updatedActor: _common_pb2.AuditActor
    createdDate: _timestamp_pb2.Timestamp
    updatedDate: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[int] = ..., description: _Optional[str] = ..., assistantId: _Optional[int] = ..., schemaVersion: _Optional[str] = ..., definition: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., status: _Optional[str] = ..., createdActor: _Optional[_Union[_common_pb2.AuditActor, _Mapping]] = ..., updatedActor: _Optional[_Union[_common_pb2.AuditActor, _Mapping]] = ..., createdDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updatedDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetAllAssistantProviderRequest(_message.Message):
    __slots__ = ("paginate", "criterias", "assistantId")
    PAGINATE_FIELD_NUMBER: _ClassVar[int]
    CRITERIAS_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    paginate: _common_pb2.Paginate
    criterias: _containers.RepeatedCompositeFieldContainer[_common_pb2.Criteria]
    assistantId: int
    def __init__(self, paginate: _Optional[_Union[_common_pb2.Paginate, _Mapping]] = ..., criterias: _Optional[_Iterable[_Union[_common_pb2.Criteria, _Mapping]]] = ..., assistantId: _Optional[int] = ...) -> None: ...

class GetAssistantProviderResponse(_message.Message):
    __slots__ = ("code", "success", "assistantProviderModel", "assistantProviderAgentkit", "assistantProviderWebsocket", "assistantProviderAgentflow", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTPROVIDERMODEL_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTPROVIDERAGENTKIT_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTPROVIDERWEBSOCKET_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTPROVIDERAGENTFLOW_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    assistantProviderModel: AssistantProviderModel
    assistantProviderAgentkit: AssistantProviderAgentkit
    assistantProviderWebsocket: AssistantProviderWebsocket
    assistantProviderAgentflow: AssistantProviderAgentflow
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., assistantProviderModel: _Optional[_Union[AssistantProviderModel, _Mapping]] = ..., assistantProviderAgentkit: _Optional[_Union[AssistantProviderAgentkit, _Mapping]] = ..., assistantProviderWebsocket: _Optional[_Union[AssistantProviderWebsocket, _Mapping]] = ..., assistantProviderAgentflow: _Optional[_Union[AssistantProviderAgentflow, _Mapping]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class GetAllAssistantProviderResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error", "paginated")
    class AssistantProvider(_message.Message):
        __slots__ = ("assistantProviderModel", "assistantProviderAgentkit", "assistantProviderWebsocket", "assistantProviderAgentflow")
        ASSISTANTPROVIDERMODEL_FIELD_NUMBER: _ClassVar[int]
        ASSISTANTPROVIDERAGENTKIT_FIELD_NUMBER: _ClassVar[int]
        ASSISTANTPROVIDERWEBSOCKET_FIELD_NUMBER: _ClassVar[int]
        ASSISTANTPROVIDERAGENTFLOW_FIELD_NUMBER: _ClassVar[int]
        assistantProviderModel: AssistantProviderModel
        assistantProviderAgentkit: AssistantProviderAgentkit
        assistantProviderWebsocket: AssistantProviderWebsocket
        assistantProviderAgentflow: AssistantProviderAgentflow
        def __init__(self, assistantProviderModel: _Optional[_Union[AssistantProviderModel, _Mapping]] = ..., assistantProviderAgentkit: _Optional[_Union[AssistantProviderAgentkit, _Mapping]] = ..., assistantProviderWebsocket: _Optional[_Union[AssistantProviderWebsocket, _Mapping]] = ..., assistantProviderAgentflow: _Optional[_Union[AssistantProviderAgentflow, _Mapping]] = ...) -> None: ...
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    PAGINATED_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: _containers.RepeatedCompositeFieldContainer[GetAllAssistantProviderResponse.AssistantProvider]
    error: _common_pb2.Error
    paginated: _common_pb2.Paginated
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Iterable[_Union[GetAllAssistantProviderResponse.AssistantProvider, _Mapping]]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ..., paginated: _Optional[_Union[_common_pb2.Paginated, _Mapping]] = ...) -> None: ...
