from google.protobuf import any_pb2 as _any_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Source(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    WEB_PLUGIN: _ClassVar[Source]
    DEBUGGER: _ClassVar[Source]
    SDK: _ClassVar[Source]
    PHONE_CALL: _ClassVar[Source]
    WHATSAPP: _ClassVar[Source]
WEB_PLUGIN: Source
DEBUGGER: Source
SDK: Source
PHONE_CALL: Source
WHATSAPP: Source

class FieldSelector(_message.Message):
    __slots__ = ("field",)
    FIELD_FIELD_NUMBER: _ClassVar[int]
    field: str
    def __init__(self, field: _Optional[str] = ...) -> None: ...

class AssistantDefinition(_message.Message):
    __slots__ = ("assistantId", "version")
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    assistantId: int
    version: str
    def __init__(self, assistantId: _Optional[int] = ..., version: _Optional[str] = ...) -> None: ...

class Criteria(_message.Message):
    __slots__ = ("key", "value", "logic")
    KEY_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    LOGIC_FIELD_NUMBER: _ClassVar[int]
    key: str
    value: str
    logic: str
    def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ..., logic: _Optional[str] = ...) -> None: ...

class Error(_message.Message):
    __slots__ = ("errorCode", "errorMessage", "humanMessage")
    ERRORCODE_FIELD_NUMBER: _ClassVar[int]
    ERRORMESSAGE_FIELD_NUMBER: _ClassVar[int]
    HUMANMESSAGE_FIELD_NUMBER: _ClassVar[int]
    errorCode: int
    errorMessage: str
    humanMessage: str
    def __init__(self, errorCode: _Optional[int] = ..., errorMessage: _Optional[str] = ..., humanMessage: _Optional[str] = ...) -> None: ...

class Paginate(_message.Message):
    __slots__ = ("page", "pageSize")
    PAGE_FIELD_NUMBER: _ClassVar[int]
    PAGESIZE_FIELD_NUMBER: _ClassVar[int]
    page: int
    pageSize: int
    def __init__(self, page: _Optional[int] = ..., pageSize: _Optional[int] = ...) -> None: ...

class Paginated(_message.Message):
    __slots__ = ("currentPage", "totalItem")
    CURRENTPAGE_FIELD_NUMBER: _ClassVar[int]
    TOTALITEM_FIELD_NUMBER: _ClassVar[int]
    currentPage: int
    totalItem: int
    def __init__(self, currentPage: _Optional[int] = ..., totalItem: _Optional[int] = ...) -> None: ...

class Ordering(_message.Message):
    __slots__ = ("column", "order")
    COLUMN_FIELD_NUMBER: _ClassVar[int]
    ORDER_FIELD_NUMBER: _ClassVar[int]
    column: str
    order: str
    def __init__(self, column: _Optional[str] = ..., order: _Optional[str] = ...) -> None: ...

class User(_message.Message):
    __slots__ = ("id", "name", "email", "role", "createdDate", "status")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    CREATEDDATE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    id: int
    name: str
    email: str
    role: str
    createdDate: _timestamp_pb2.Timestamp
    status: str
    def __init__(self, id: _Optional[int] = ..., name: _Optional[str] = ..., email: _Optional[str] = ..., role: _Optional[str] = ..., createdDate: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., status: _Optional[str] = ...) -> None: ...

class BaseResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error")
    class DataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: _containers.ScalarMap[str, str]
    error: Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Mapping[str, str]] = ..., error: _Optional[_Union[Error, _Mapping]] = ...) -> None: ...

class Metadata(_message.Message):
    __slots__ = ("id", "key", "value")
    ID_FIELD_NUMBER: _ClassVar[int]
    KEY_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    id: int
    key: str
    value: str
    def __init__(self, id: _Optional[int] = ..., key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...

class Argument(_message.Message):
    __slots__ = ("id", "name", "value")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    id: int
    name: str
    value: str
    def __init__(self, id: _Optional[int] = ..., name: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...

class Variable(_message.Message):
    __slots__ = ("id", "name", "type", "defaultValue")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    DEFAULTVALUE_FIELD_NUMBER: _ClassVar[int]
    id: int
    name: str
    type: str
    defaultValue: str
    def __init__(self, id: _Optional[int] = ..., name: _Optional[str] = ..., type: _Optional[str] = ..., defaultValue: _Optional[str] = ...) -> None: ...

class Tag(_message.Message):
    __slots__ = ("id", "tag")
    ID_FIELD_NUMBER: _ClassVar[int]
    TAG_FIELD_NUMBER: _ClassVar[int]
    id: int
    tag: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[int] = ..., tag: _Optional[_Iterable[str]] = ...) -> None: ...

class Organization(_message.Message):
    __slots__ = ("id", "name", "description", "industry", "contact", "size")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    INDUSTRY_FIELD_NUMBER: _ClassVar[int]
    CONTACT_FIELD_NUMBER: _ClassVar[int]
    SIZE_FIELD_NUMBER: _ClassVar[int]
    id: int
    name: str
    description: str
    industry: str
    contact: str
    size: str
    def __init__(self, id: _Optional[int] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., industry: _Optional[str] = ..., contact: _Optional[str] = ..., size: _Optional[str] = ...) -> None: ...

class Metric(_message.Message):
    __slots__ = ("name", "value", "description")
    NAME_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    name: str
    value: str
    description: str
    def __init__(self, name: _Optional[str] = ..., value: _Optional[str] = ..., description: _Optional[str] = ...) -> None: ...

class Content(_message.Message):
    __slots__ = ("name", "contentType", "contentFormat", "content", "meta")
    NAME_FIELD_NUMBER: _ClassVar[int]
    CONTENTTYPE_FIELD_NUMBER: _ClassVar[int]
    CONTENTFORMAT_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    META_FIELD_NUMBER: _ClassVar[int]
    name: str
    contentType: str
    contentFormat: str
    content: bytes
    meta: _struct_pb2.Struct
    def __init__(self, name: _Optional[str] = ..., contentType: _Optional[str] = ..., contentFormat: _Optional[str] = ..., content: _Optional[bytes] = ..., meta: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class Message(_message.Message):
    __slots__ = ("role", "contents", "toolCalls")
    ROLE_FIELD_NUMBER: _ClassVar[int]
    CONTENTS_FIELD_NUMBER: _ClassVar[int]
    TOOLCALLS_FIELD_NUMBER: _ClassVar[int]
    role: str
    contents: _containers.RepeatedCompositeFieldContainer[Content]
    toolCalls: _containers.RepeatedCompositeFieldContainer[ToolCall]
    def __init__(self, role: _Optional[str] = ..., contents: _Optional[_Iterable[_Union[Content, _Mapping]]] = ..., toolCalls: _Optional[_Iterable[_Union[ToolCall, _Mapping]]] = ...) -> None: ...

class ToolCall(_message.Message):
    __slots__ = ("id", "type", "function")
    ID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    FUNCTION_FIELD_NUMBER: _ClassVar[int]
    id: str
    type: str
    function: FunctionCall
    def __init__(self, id: _Optional[str] = ..., type: _Optional[str] = ..., function: _Optional[_Union[FunctionCall, _Mapping]] = ...) -> None: ...

class FunctionCall(_message.Message):
    __slots__ = ("name", "arguments")
    NAME_FIELD_NUMBER: _ClassVar[int]
    ARGUMENTS_FIELD_NUMBER: _ClassVar[int]
    name: str
    arguments: str
    def __init__(self, name: _Optional[str] = ..., arguments: _Optional[str] = ...) -> None: ...

class Telemetry(_message.Message):
    __slots__ = ("stageName", "startTime", "endTime", "duration", "attributes", "spanID", "parentID")
    class AttributesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    STAGENAME_FIELD_NUMBER: _ClassVar[int]
    STARTTIME_FIELD_NUMBER: _ClassVar[int]
    ENDTIME_FIELD_NUMBER: _ClassVar[int]
    DURATION_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTES_FIELD_NUMBER: _ClassVar[int]
    SPANID_FIELD_NUMBER: _ClassVar[int]
    PARENTID_FIELD_NUMBER: _ClassVar[int]
    stageName: str
    startTime: _timestamp_pb2.Timestamp
    endTime: _timestamp_pb2.Timestamp
    duration: int
    attributes: _containers.ScalarMap[str, str]
    spanID: str
    parentID: str
    def __init__(self, stageName: _Optional[str] = ..., startTime: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., endTime: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., duration: _Optional[int] = ..., attributes: _Optional[_Mapping[str, str]] = ..., spanID: _Optional[str] = ..., parentID: _Optional[str] = ...) -> None: ...

class Knowledge(_message.Message):
    __slots__ = ("id", "name", "description", "visibility", "language", "embeddingModelProviderId", "embeddingModelProviderName", "knowledgeEmbeddingModelOptions", "status", "createdBy", "createdUser", "updatedBy", "updatedUser", "createdDate", "updatedDate", "organizationId", "projectId", "organization", "knowledgeTag", "documentCount", "tokenCount", "wordCount")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    VISIBILITY_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    EMBEDDINGMODELPROVIDERID_FIELD_NUMBER: _ClassVar[int]
    EMBEDDINGMODELPROVIDERNAME_FIELD_NUMBER: _ClassVar[int]
    KNOWLEDGEEMBEDDINGMODELOPTIONS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CREATEDBY_FIELD_NUMBER: _ClassVar[int]
    CREATEDUSER_FIELD_NUMBER: _ClassVar[int]
    UPDATEDBY_FIELD_NUMBER: _ClassVar[int]
    UPDATEDUSER_FIELD_NUMBER: _ClassVar[int]
    CREATEDDATE_FIELD_NUMBER: _ClassVar[int]
    UPDATEDDATE_FIELD_NUMBER: _ClassVar[int]
    ORGANIZATIONID_FIELD_NUMBER: _ClassVar[int]
    PROJECTID_FIELD_NUMBER: _ClassVar[int]
    ORGANIZATION_FIELD_NUMBER: _ClassVar[int]
    KNOWLEDGETAG_FIELD_NUMBER: _ClassVar[int]
    DOCUMENTCOUNT_FIELD_NUMBER: _ClassVar[int]
    TOKENCOUNT_FIELD_NUMBER: _ClassVar[int]
    WORDCOUNT_FIELD_NUMBER: _ClassVar[int]
    id: int
    name: str
    description: str
    visibility: str
    language: str
    embeddingModelProviderId: int
    embeddingModelProviderName: str
    knowledgeEmbeddingModelOptions: _containers.RepeatedCompositeFieldContainer[Metadata]
    status: str
    createdBy: int
    createdUser: User
    updatedBy: int
    updatedUser: User
    createdDate: _timestamp_pb2.Timestamp
    updatedDate: _timestamp_pb2.Timestamp
    organizationId: int
    projectId: int
    organization: Organization
    knowledgeTag: Tag
    documentCount: int
    tokenCount: int
    wordCount: int
    def __init__(self, id: _Optional[int] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., visibility: _Optional[str] = ..., language: _Optional[str] = ..., embeddingModelProviderId: _Optional[int] = ..., embeddingModelProviderName: _Optional[str] = ..., knowledgeEmbeddingModelOptions: _Optional[_Iterable[_Union[Metadata, _Mapping]]] = ..., status: _Optional[str] = ..., createdBy: _Optional[int] = ..., createdUser: _Optional[_Union[User, _Mapping]] = ..., updatedBy: _Optional[int] = ..., updatedUser: _Optional[_Union[User, _Mapping]] = ..., createdDate: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., updatedDate: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., organizationId: _Optional[int] = ..., projectId: _Optional[int] = ..., organization: _Optional[_Union[Organization, _Mapping]] = ..., knowledgeTag: _Optional[_Union[Tag, _Mapping]] = ..., documentCount: _Optional[int] = ..., tokenCount: _Optional[int] = ..., wordCount: _Optional[int] = ...) -> None: ...

class TextPrompt(_message.Message):
    __slots__ = ("role", "content")
    ROLE_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    role: str
    content: str
    def __init__(self, role: _Optional[str] = ..., content: _Optional[str] = ...) -> None: ...

class TextChatCompletePrompt(_message.Message):
    __slots__ = ("prompt", "promptVariables")
    PROMPT_FIELD_NUMBER: _ClassVar[int]
    PROMPTVARIABLES_FIELD_NUMBER: _ClassVar[int]
    prompt: _containers.RepeatedCompositeFieldContainer[TextPrompt]
    promptVariables: _containers.RepeatedCompositeFieldContainer[Variable]
    def __init__(self, prompt: _Optional[_Iterable[_Union[TextPrompt, _Mapping]]] = ..., promptVariables: _Optional[_Iterable[_Union[Variable, _Mapping]]] = ...) -> None: ...

class AssistantConversationMessage(_message.Message):
    __slots__ = ("id", "messageId", "assistantConversationId", "request", "response", "source", "metrics", "status", "createdBy", "updatedBy", "createdDate", "updatedDate", "assistantId", "assistantProviderModelId", "metadata")
    ID_FIELD_NUMBER: _ClassVar[int]
    MESSAGEID_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTCONVERSATIONID_FIELD_NUMBER: _ClassVar[int]
    REQUEST_FIELD_NUMBER: _ClassVar[int]
    RESPONSE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CREATEDBY_FIELD_NUMBER: _ClassVar[int]
    UPDATEDBY_FIELD_NUMBER: _ClassVar[int]
    CREATEDDATE_FIELD_NUMBER: _ClassVar[int]
    UPDATEDDATE_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTPROVIDERMODELID_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    id: int
    messageId: str
    assistantConversationId: int
    request: Message
    response: Message
    source: str
    metrics: _containers.RepeatedCompositeFieldContainer[Metric]
    status: str
    createdBy: int
    updatedBy: int
    createdDate: _timestamp_pb2.Timestamp
    updatedDate: _timestamp_pb2.Timestamp
    assistantId: int
    assistantProviderModelId: int
    metadata: _containers.RepeatedCompositeFieldContainer[Metadata]
    def __init__(self, id: _Optional[int] = ..., messageId: _Optional[str] = ..., assistantConversationId: _Optional[int] = ..., request: _Optional[_Union[Message, _Mapping]] = ..., response: _Optional[_Union[Message, _Mapping]] = ..., source: _Optional[str] = ..., metrics: _Optional[_Iterable[_Union[Metric, _Mapping]]] = ..., status: _Optional[str] = ..., createdBy: _Optional[int] = ..., updatedBy: _Optional[int] = ..., createdDate: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., updatedDate: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., assistantId: _Optional[int] = ..., assistantProviderModelId: _Optional[int] = ..., metadata: _Optional[_Iterable[_Union[Metadata, _Mapping]]] = ...) -> None: ...

class AssistantConversationContext(_message.Message):
    __slots__ = ("id", "metadata", "result", "query")
    ID_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    id: int
    metadata: _struct_pb2.Struct
    result: _struct_pb2.Struct
    query: _struct_pb2.Struct
    def __init__(self, id: _Optional[int] = ..., metadata: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., result: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., query: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class AssistantConversationRecording(_message.Message):
    __slots__ = ("recordingUrl",)
    RECORDINGURL_FIELD_NUMBER: _ClassVar[int]
    recordingUrl: str
    def __init__(self, recordingUrl: _Optional[str] = ...) -> None: ...

class AssistantConversationTelephonyEvent(_message.Message):
    __slots__ = ("id", "assistantConversationId", "provider", "eventType", "payload", "createdDate", "updatedDate")
    ID_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTCONVERSATIONID_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    EVENTTYPE_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    CREATEDDATE_FIELD_NUMBER: _ClassVar[int]
    UPDATEDDATE_FIELD_NUMBER: _ClassVar[int]
    id: int
    assistantConversationId: int
    provider: str
    eventType: str
    payload: _struct_pb2.Struct
    createdDate: _timestamp_pb2.Timestamp
    updatedDate: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[int] = ..., assistantConversationId: _Optional[int] = ..., provider: _Optional[str] = ..., eventType: _Optional[str] = ..., payload: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., createdDate: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., updatedDate: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class AssistantConversation(_message.Message):
    __slots__ = ("id", "userId", "assistantId", "name", "projectId", "organizationId", "source", "createdBy", "updatedBy", "user", "assistantProviderModelId", "assistantConversationMessage", "identifier", "status", "createdDate", "updatedDate", "contexts", "metrics", "metadata", "arguments", "options", "direction", "recordings", "telephonyEvents")
    ID_FIELD_NUMBER: _ClassVar[int]
    USERID_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    PROJECTID_FIELD_NUMBER: _ClassVar[int]
    ORGANIZATIONID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    CREATEDBY_FIELD_NUMBER: _ClassVar[int]
    UPDATEDBY_FIELD_NUMBER: _ClassVar[int]
    USER_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTPROVIDERMODELID_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTCONVERSATIONMESSAGE_FIELD_NUMBER: _ClassVar[int]
    IDENTIFIER_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CREATEDDATE_FIELD_NUMBER: _ClassVar[int]
    UPDATEDDATE_FIELD_NUMBER: _ClassVar[int]
    CONTEXTS_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    ARGUMENTS_FIELD_NUMBER: _ClassVar[int]
    OPTIONS_FIELD_NUMBER: _ClassVar[int]
    DIRECTION_FIELD_NUMBER: _ClassVar[int]
    RECORDINGS_FIELD_NUMBER: _ClassVar[int]
    TELEPHONYEVENTS_FIELD_NUMBER: _ClassVar[int]
    id: int
    userId: int
    assistantId: int
    name: str
    projectId: int
    organizationId: int
    source: str
    createdBy: int
    updatedBy: int
    user: User
    assistantProviderModelId: int
    assistantConversationMessage: _containers.RepeatedCompositeFieldContainer[AssistantConversationMessage]
    identifier: str
    status: str
    createdDate: _timestamp_pb2.Timestamp
    updatedDate: _timestamp_pb2.Timestamp
    contexts: _containers.RepeatedCompositeFieldContainer[AssistantConversationContext]
    metrics: _containers.RepeatedCompositeFieldContainer[Metric]
    metadata: _containers.RepeatedCompositeFieldContainer[Metadata]
    arguments: _containers.RepeatedCompositeFieldContainer[Argument]
    options: _containers.RepeatedCompositeFieldContainer[Metadata]
    direction: str
    recordings: _containers.RepeatedCompositeFieldContainer[AssistantConversationRecording]
    telephonyEvents: _containers.RepeatedCompositeFieldContainer[AssistantConversationTelephonyEvent]
    def __init__(self, id: _Optional[int] = ..., userId: _Optional[int] = ..., assistantId: _Optional[int] = ..., name: _Optional[str] = ..., projectId: _Optional[int] = ..., organizationId: _Optional[int] = ..., source: _Optional[str] = ..., createdBy: _Optional[int] = ..., updatedBy: _Optional[int] = ..., user: _Optional[_Union[User, _Mapping]] = ..., assistantProviderModelId: _Optional[int] = ..., assistantConversationMessage: _Optional[_Iterable[_Union[AssistantConversationMessage, _Mapping]]] = ..., identifier: _Optional[str] = ..., status: _Optional[str] = ..., createdDate: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., updatedDate: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., contexts: _Optional[_Iterable[_Union[AssistantConversationContext, _Mapping]]] = ..., metrics: _Optional[_Iterable[_Union[Metric, _Mapping]]] = ..., metadata: _Optional[_Iterable[_Union[Metadata, _Mapping]]] = ..., arguments: _Optional[_Iterable[_Union[Argument, _Mapping]]] = ..., options: _Optional[_Iterable[_Union[Metadata, _Mapping]]] = ..., direction: _Optional[str] = ..., recordings: _Optional[_Iterable[_Union[AssistantConversationRecording, _Mapping]]] = ..., telephonyEvents: _Optional[_Iterable[_Union[AssistantConversationTelephonyEvent, _Mapping]]] = ...) -> None: ...

class GetAllAssistantConversationRequest(_message.Message):
    __slots__ = ("assistantId", "paginate", "criterias", "source")
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    PAGINATE_FIELD_NUMBER: _ClassVar[int]
    CRITERIAS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    assistantId: int
    paginate: Paginate
    criterias: _containers.RepeatedCompositeFieldContainer[Criteria]
    source: Source
    def __init__(self, assistantId: _Optional[int] = ..., paginate: _Optional[_Union[Paginate, _Mapping]] = ..., criterias: _Optional[_Iterable[_Union[Criteria, _Mapping]]] = ..., source: _Optional[_Union[Source, str]] = ...) -> None: ...

class GetAllAssistantConversationResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error", "paginated")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    PAGINATED_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: _containers.RepeatedCompositeFieldContainer[AssistantConversation]
    error: Error
    paginated: Paginated
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Iterable[_Union[AssistantConversation, _Mapping]]] = ..., error: _Optional[_Union[Error, _Mapping]] = ..., paginated: _Optional[_Union[Paginated, _Mapping]] = ...) -> None: ...

class GetAllConversationMessageRequest(_message.Message):
    __slots__ = ("assistantId", "assistantConversationId", "paginate", "criterias", "order", "source")
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTCONVERSATIONID_FIELD_NUMBER: _ClassVar[int]
    PAGINATE_FIELD_NUMBER: _ClassVar[int]
    CRITERIAS_FIELD_NUMBER: _ClassVar[int]
    ORDER_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    assistantId: int
    assistantConversationId: int
    paginate: Paginate
    criterias: _containers.RepeatedCompositeFieldContainer[Criteria]
    order: Ordering
    source: Source
    def __init__(self, assistantId: _Optional[int] = ..., assistantConversationId: _Optional[int] = ..., paginate: _Optional[_Union[Paginate, _Mapping]] = ..., criterias: _Optional[_Iterable[_Union[Criteria, _Mapping]]] = ..., order: _Optional[_Union[Ordering, _Mapping]] = ..., source: _Optional[_Union[Source, str]] = ...) -> None: ...

class GetAllConversationMessageResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error", "paginated")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    PAGINATED_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: _containers.RepeatedCompositeFieldContainer[AssistantConversationMessage]
    error: Error
    paginated: Paginated
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Iterable[_Union[AssistantConversationMessage, _Mapping]]] = ..., error: _Optional[_Union[Error, _Mapping]] = ..., paginated: _Optional[_Union[Paginated, _Mapping]] = ...) -> None: ...

class AssistantConversationConfiguration(_message.Message):
    __slots__ = ("assistantConversationId", "assistant", "time", "metadata", "args", "options", "inputConfig", "outputConfig")
    class MetadataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _any_pb2.Any
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_any_pb2.Any, _Mapping]] = ...) -> None: ...
    class ArgsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _any_pb2.Any
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_any_pb2.Any, _Mapping]] = ...) -> None: ...
    class OptionsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _any_pb2.Any
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_any_pb2.Any, _Mapping]] = ...) -> None: ...
    ASSISTANTCONVERSATIONID_FIELD_NUMBER: _ClassVar[int]
    ASSISTANT_FIELD_NUMBER: _ClassVar[int]
    TIME_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    ARGS_FIELD_NUMBER: _ClassVar[int]
    OPTIONS_FIELD_NUMBER: _ClassVar[int]
    INPUTCONFIG_FIELD_NUMBER: _ClassVar[int]
    OUTPUTCONFIG_FIELD_NUMBER: _ClassVar[int]
    assistantConversationId: int
    assistant: AssistantDefinition
    time: _timestamp_pb2.Timestamp
    metadata: _containers.MessageMap[str, _any_pb2.Any]
    args: _containers.MessageMap[str, _any_pb2.Any]
    options: _containers.MessageMap[str, _any_pb2.Any]
    inputConfig: StreamConfig
    outputConfig: StreamConfig
    def __init__(self, assistantConversationId: _Optional[int] = ..., assistant: _Optional[_Union[AssistantDefinition, _Mapping]] = ..., time: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., metadata: _Optional[_Mapping[str, _any_pb2.Any]] = ..., args: _Optional[_Mapping[str, _any_pb2.Any]] = ..., options: _Optional[_Mapping[str, _any_pb2.Any]] = ..., inputConfig: _Optional[_Union[StreamConfig, _Mapping]] = ..., outputConfig: _Optional[_Union[StreamConfig, _Mapping]] = ...) -> None: ...

class AssistantConversationError(_message.Message):
    __slots__ = ("error",)
    ERROR_FIELD_NUMBER: _ClassVar[int]
    error: Error
    def __init__(self, error: _Optional[_Union[Error, _Mapping]] = ...) -> None: ...

class StreamConfig(_message.Message):
    __slots__ = ("audio", "text")
    AUDIO_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    audio: AudioConfig
    text: TextConfig
    def __init__(self, audio: _Optional[_Union[AudioConfig, _Mapping]] = ..., text: _Optional[_Union[TextConfig, _Mapping]] = ...) -> None: ...

class AudioConfig(_message.Message):
    __slots__ = ("sampleRate", "audioFormat", "channels")
    class AudioFormat(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = ()
        LINEAR16: _ClassVar[AudioConfig.AudioFormat]
        MuLaw8: _ClassVar[AudioConfig.AudioFormat]
    LINEAR16: AudioConfig.AudioFormat
    MuLaw8: AudioConfig.AudioFormat
    SAMPLERATE_FIELD_NUMBER: _ClassVar[int]
    AUDIOFORMAT_FIELD_NUMBER: _ClassVar[int]
    CHANNELS_FIELD_NUMBER: _ClassVar[int]
    sampleRate: int
    audioFormat: AudioConfig.AudioFormat
    channels: int
    def __init__(self, sampleRate: _Optional[int] = ..., audioFormat: _Optional[_Union[AudioConfig.AudioFormat, str]] = ..., channels: _Optional[int] = ...) -> None: ...

class TextConfig(_message.Message):
    __slots__ = ("charset",)
    CHARSET_FIELD_NUMBER: _ClassVar[int]
    charset: str
    def __init__(self, charset: _Optional[str] = ...) -> None: ...

class AssistantConversationAction(_message.Message):
    __slots__ = ("name", "action", "args")
    class ActionType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = ()
        ACTION_UNSPECIFIED: _ClassVar[AssistantConversationAction.ActionType]
        KNOWLEDGE_RETRIEVAL: _ClassVar[AssistantConversationAction.ActionType]
        API_REQUEST: _ClassVar[AssistantConversationAction.ActionType]
        ENDPOINT_CALL: _ClassVar[AssistantConversationAction.ActionType]
        PUT_ON_HOLD: _ClassVar[AssistantConversationAction.ActionType]
        END_CONVERSATION: _ClassVar[AssistantConversationAction.ActionType]
    ACTION_UNSPECIFIED: AssistantConversationAction.ActionType
    KNOWLEDGE_RETRIEVAL: AssistantConversationAction.ActionType
    API_REQUEST: AssistantConversationAction.ActionType
    ENDPOINT_CALL: AssistantConversationAction.ActionType
    PUT_ON_HOLD: AssistantConversationAction.ActionType
    END_CONVERSATION: AssistantConversationAction.ActionType
    class ArgsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _any_pb2.Any
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_any_pb2.Any, _Mapping]] = ...) -> None: ...
    NAME_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    ARGS_FIELD_NUMBER: _ClassVar[int]
    name: str
    action: AssistantConversationAction.ActionType
    args: _containers.MessageMap[str, _any_pb2.Any]
    def __init__(self, name: _Optional[str] = ..., action: _Optional[_Union[AssistantConversationAction.ActionType, str]] = ..., args: _Optional[_Mapping[str, _any_pb2.Any]] = ...) -> None: ...

class AssistantConversationInterruption(_message.Message):
    __slots__ = ("id", "type", "time")
    class InterruptionType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = ()
        INTERRUPTION_TYPE_UNSPECIFIED: _ClassVar[AssistantConversationInterruption.InterruptionType]
        INTERRUPTION_TYPE_VAD: _ClassVar[AssistantConversationInterruption.InterruptionType]
        INTERRUPTION_TYPE_WORD: _ClassVar[AssistantConversationInterruption.InterruptionType]
    INTERRUPTION_TYPE_UNSPECIFIED: AssistantConversationInterruption.InterruptionType
    INTERRUPTION_TYPE_VAD: AssistantConversationInterruption.InterruptionType
    INTERRUPTION_TYPE_WORD: AssistantConversationInterruption.InterruptionType
    ID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    TIME_FIELD_NUMBER: _ClassVar[int]
    id: str
    type: AssistantConversationInterruption.InterruptionType
    time: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., type: _Optional[_Union[AssistantConversationInterruption.InterruptionType, str]] = ..., time: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class AssistantConversationMessageTextContent(_message.Message):
    __slots__ = ("content",)
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    content: str
    def __init__(self, content: _Optional[str] = ...) -> None: ...

class AssistantConversationMessageAudioContent(_message.Message):
    __slots__ = ("content",)
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    content: bytes
    def __init__(self, content: _Optional[bytes] = ...) -> None: ...

class AssistantConversationUserMessage(_message.Message):
    __slots__ = ("audio", "text", "id", "completed", "time")
    AUDIO_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_FIELD_NUMBER: _ClassVar[int]
    TIME_FIELD_NUMBER: _ClassVar[int]
    audio: AssistantConversationMessageAudioContent
    text: AssistantConversationMessageTextContent
    id: str
    completed: bool
    time: _timestamp_pb2.Timestamp
    def __init__(self, audio: _Optional[_Union[AssistantConversationMessageAudioContent, _Mapping]] = ..., text: _Optional[_Union[AssistantConversationMessageTextContent, _Mapping]] = ..., id: _Optional[str] = ..., completed: bool = ..., time: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class AssistantConversationAssistantMessage(_message.Message):
    __slots__ = ("audio", "text", "id", "completed", "time")
    AUDIO_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_FIELD_NUMBER: _ClassVar[int]
    TIME_FIELD_NUMBER: _ClassVar[int]
    audio: AssistantConversationMessageAudioContent
    text: AssistantConversationMessageTextContent
    id: str
    completed: bool
    time: _timestamp_pb2.Timestamp
    def __init__(self, audio: _Optional[_Union[AssistantConversationMessageAudioContent, _Mapping]] = ..., text: _Optional[_Union[AssistantConversationMessageTextContent, _Mapping]] = ..., id: _Optional[str] = ..., completed: bool = ..., time: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
