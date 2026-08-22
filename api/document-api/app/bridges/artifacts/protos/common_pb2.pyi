import datetime

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
    def __init__(self, id: _Optional[int] = ..., name: _Optional[str] = ..., email: _Optional[str] = ..., role: _Optional[str] = ..., createdDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., status: _Optional[str] = ...) -> None: ...

class AuditActor(_message.Message):
    __slots__ = ("type", "id", "displayName")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAYNAME_FIELD_NUMBER: _ClassVar[int]
    type: str
    id: int
    displayName: str
    def __init__(self, type: _Optional[str] = ..., id: _Optional[int] = ..., displayName: _Optional[str] = ...) -> None: ...

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

class AssistantMessage(_message.Message):
    __slots__ = ("contents", "toolCalls")
    CONTENTS_FIELD_NUMBER: _ClassVar[int]
    TOOLCALLS_FIELD_NUMBER: _ClassVar[int]
    contents: _containers.RepeatedScalarFieldContainer[str]
    toolCalls: _containers.RepeatedCompositeFieldContainer[ToolCall]
    def __init__(self, contents: _Optional[_Iterable[str]] = ..., toolCalls: _Optional[_Iterable[_Union[ToolCall, _Mapping]]] = ...) -> None: ...

class SystemMessage(_message.Message):
    __slots__ = ("content",)
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    content: str
    def __init__(self, content: _Optional[str] = ...) -> None: ...

class UserMessage(_message.Message):
    __slots__ = ("content",)
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    content: str
    def __init__(self, content: _Optional[str] = ...) -> None: ...

class ToolMessage(_message.Message):
    __slots__ = ("tools",)
    class Tool(_message.Message):
        __slots__ = ("name", "id", "content")
        NAME_FIELD_NUMBER: _ClassVar[int]
        ID_FIELD_NUMBER: _ClassVar[int]
        CONTENT_FIELD_NUMBER: _ClassVar[int]
        name: str
        id: str
        content: str
        def __init__(self, name: _Optional[str] = ..., id: _Optional[str] = ..., content: _Optional[str] = ...) -> None: ...
    TOOLS_FIELD_NUMBER: _ClassVar[int]
    tools: _containers.RepeatedCompositeFieldContainer[ToolMessage.Tool]
    def __init__(self, tools: _Optional[_Iterable[_Union[ToolMessage.Tool, _Mapping]]] = ...) -> None: ...

class Message(_message.Message):
    __slots__ = ("role", "assistant", "user", "tool", "system")
    ROLE_FIELD_NUMBER: _ClassVar[int]
    ASSISTANT_FIELD_NUMBER: _ClassVar[int]
    USER_FIELD_NUMBER: _ClassVar[int]
    TOOL_FIELD_NUMBER: _ClassVar[int]
    SYSTEM_FIELD_NUMBER: _ClassVar[int]
    role: str
    assistant: AssistantMessage
    user: UserMessage
    tool: ToolMessage
    system: SystemMessage
    def __init__(self, role: _Optional[str] = ..., assistant: _Optional[_Union[AssistantMessage, _Mapping]] = ..., user: _Optional[_Union[UserMessage, _Mapping]] = ..., tool: _Optional[_Union[ToolMessage, _Mapping]] = ..., system: _Optional[_Union[SystemMessage, _Mapping]] = ...) -> None: ...

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

class Knowledge(_message.Message):
    __slots__ = ("id", "name", "description", "visibility", "language", "embeddingModelProviderId", "embeddingModelProviderName", "knowledgeEmbeddingModelOptions", "status", "createdActor", "updatedActor", "createdDate", "updatedDate", "organizationId", "projectId", "organization", "knowledgeTag", "documentCount", "tokenCount", "wordCount")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    VISIBILITY_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    EMBEDDINGMODELPROVIDERID_FIELD_NUMBER: _ClassVar[int]
    EMBEDDINGMODELPROVIDERNAME_FIELD_NUMBER: _ClassVar[int]
    KNOWLEDGEEMBEDDINGMODELOPTIONS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CREATEDACTOR_FIELD_NUMBER: _ClassVar[int]
    UPDATEDACTOR_FIELD_NUMBER: _ClassVar[int]
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
    createdActor: AuditActor
    updatedActor: AuditActor
    createdDate: _timestamp_pb2.Timestamp
    updatedDate: _timestamp_pb2.Timestamp
    organizationId: int
    projectId: int
    organization: Organization
    knowledgeTag: Tag
    documentCount: int
    tokenCount: int
    wordCount: int
    def __init__(self, id: _Optional[int] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., visibility: _Optional[str] = ..., language: _Optional[str] = ..., embeddingModelProviderId: _Optional[int] = ..., embeddingModelProviderName: _Optional[str] = ..., knowledgeEmbeddingModelOptions: _Optional[_Iterable[_Union[Metadata, _Mapping]]] = ..., status: _Optional[str] = ..., createdActor: _Optional[_Union[AuditActor, _Mapping]] = ..., updatedActor: _Optional[_Union[AuditActor, _Mapping]] = ..., createdDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updatedDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., organizationId: _Optional[int] = ..., projectId: _Optional[int] = ..., organization: _Optional[_Union[Organization, _Mapping]] = ..., knowledgeTag: _Optional[_Union[Tag, _Mapping]] = ..., documentCount: _Optional[int] = ..., tokenCount: _Optional[int] = ..., wordCount: _Optional[int] = ...) -> None: ...

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
    __slots__ = ("id", "messageId", "assistantConversationId", "role", "body", "source", "metrics", "status", "createdActor", "updatedActor", "createdDate", "updatedDate", "assistantId", "assistantProviderModelId", "metadata")
    ID_FIELD_NUMBER: _ClassVar[int]
    MESSAGEID_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTCONVERSATIONID_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CREATEDACTOR_FIELD_NUMBER: _ClassVar[int]
    UPDATEDACTOR_FIELD_NUMBER: _ClassVar[int]
    CREATEDDATE_FIELD_NUMBER: _ClassVar[int]
    UPDATEDDATE_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTPROVIDERMODELID_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    id: int
    messageId: str
    assistantConversationId: int
    role: str
    body: str
    source: str
    metrics: _containers.RepeatedCompositeFieldContainer[Metric]
    status: str
    createdActor: AuditActor
    updatedActor: AuditActor
    createdDate: _timestamp_pb2.Timestamp
    updatedDate: _timestamp_pb2.Timestamp
    assistantId: int
    assistantProviderModelId: int
    metadata: _containers.RepeatedCompositeFieldContainer[Metadata]
    def __init__(self, id: _Optional[int] = ..., messageId: _Optional[str] = ..., assistantConversationId: _Optional[int] = ..., role: _Optional[str] = ..., body: _Optional[str] = ..., source: _Optional[str] = ..., metrics: _Optional[_Iterable[_Union[Metric, _Mapping]]] = ..., status: _Optional[str] = ..., createdActor: _Optional[_Union[AuditActor, _Mapping]] = ..., updatedActor: _Optional[_Union[AuditActor, _Mapping]] = ..., createdDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updatedDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., assistantId: _Optional[int] = ..., assistantProviderModelId: _Optional[int] = ..., metadata: _Optional[_Iterable[_Union[Metadata, _Mapping]]] = ...) -> None: ...

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
    __slots__ = ("id", "assistantRecordingUrl", "userRecordingUrl", "conversationRecordingUrl")
    ID_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTRECORDINGURL_FIELD_NUMBER: _ClassVar[int]
    USERRECORDINGURL_FIELD_NUMBER: _ClassVar[int]
    CONVERSATIONRECORDINGURL_FIELD_NUMBER: _ClassVar[int]
    id: int
    assistantRecordingUrl: str
    userRecordingUrl: str
    conversationRecordingUrl: str
    def __init__(self, id: _Optional[int] = ..., assistantRecordingUrl: _Optional[str] = ..., userRecordingUrl: _Optional[str] = ..., conversationRecordingUrl: _Optional[str] = ...) -> None: ...

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
    def __init__(self, id: _Optional[int] = ..., assistantConversationId: _Optional[int] = ..., provider: _Optional[str] = ..., eventType: _Optional[str] = ..., payload: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., createdDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updatedDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class AssistantConversation(_message.Message):
    __slots__ = ("id", "userId", "assistantId", "name", "projectId", "organizationId", "source", "createdActor", "updatedActor", "user", "assistantProviderModelId", "assistantConversationMessage", "identifier", "status", "createdDate", "updatedDate", "contexts", "metrics", "metadata", "arguments", "options", "direction", "recordings", "telephonyEvents")
    ID_FIELD_NUMBER: _ClassVar[int]
    USERID_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    PROJECTID_FIELD_NUMBER: _ClassVar[int]
    ORGANIZATIONID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    CREATEDACTOR_FIELD_NUMBER: _ClassVar[int]
    UPDATEDACTOR_FIELD_NUMBER: _ClassVar[int]
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
    createdActor: AuditActor
    updatedActor: AuditActor
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
    def __init__(self, id: _Optional[int] = ..., userId: _Optional[int] = ..., assistantId: _Optional[int] = ..., name: _Optional[str] = ..., projectId: _Optional[int] = ..., organizationId: _Optional[int] = ..., source: _Optional[str] = ..., createdActor: _Optional[_Union[AuditActor, _Mapping]] = ..., updatedActor: _Optional[_Union[AuditActor, _Mapping]] = ..., user: _Optional[_Union[User, _Mapping]] = ..., assistantProviderModelId: _Optional[int] = ..., assistantConversationMessage: _Optional[_Iterable[_Union[AssistantConversationMessage, _Mapping]]] = ..., identifier: _Optional[str] = ..., status: _Optional[str] = ..., createdDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updatedDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., contexts: _Optional[_Iterable[_Union[AssistantConversationContext, _Mapping]]] = ..., metrics: _Optional[_Iterable[_Union[Metric, _Mapping]]] = ..., metadata: _Optional[_Iterable[_Union[Metadata, _Mapping]]] = ..., arguments: _Optional[_Iterable[_Union[Argument, _Mapping]]] = ..., options: _Optional[_Iterable[_Union[Metadata, _Mapping]]] = ..., direction: _Optional[str] = ..., recordings: _Optional[_Iterable[_Union[AssistantConversationRecording, _Mapping]]] = ..., telephonyEvents: _Optional[_Iterable[_Union[AssistantConversationTelephonyEvent, _Mapping]]] = ...) -> None: ...

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
