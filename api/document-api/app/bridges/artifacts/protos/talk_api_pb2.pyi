import datetime

from google.protobuf import any_pb2 as _any_pb2
import app.bridges.artifacts.protos.common_pb2 as _common_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ToolCallAction(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    TOOL_CALL_ACTION_UNSPECIFIED: _ClassVar[ToolCallAction]
    TOOL_CALL_ACTION_END_CONVERSATION: _ClassVar[ToolCallAction]
    TOOL_CALL_ACTION_TRANSFER_CONVERSATION: _ClassVar[ToolCallAction]

class StreamMode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    STREAM_MODE_UNSPECIFIED: _ClassVar[StreamMode]
    STREAM_MODE_TEXT: _ClassVar[StreamMode]
    STREAM_MODE_AUDIO: _ClassVar[StreamMode]
    STREAM_MODE_BOTH: _ClassVar[StreamMode]
TOOL_CALL_ACTION_UNSPECIFIED: ToolCallAction
TOOL_CALL_ACTION_END_CONVERSATION: ToolCallAction
TOOL_CALL_ACTION_TRANSFER_CONVERSATION: ToolCallAction
STREAM_MODE_UNSPECIFIED: StreamMode
STREAM_MODE_TEXT: StreamMode
STREAM_MODE_AUDIO: StreamMode
STREAM_MODE_BOTH: StreamMode

class ConversationToolCall(_message.Message):
    __slots__ = ("id", "toolId", "name", "action", "args", "time")
    class ArgsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    TOOLID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    ARGS_FIELD_NUMBER: _ClassVar[int]
    TIME_FIELD_NUMBER: _ClassVar[int]
    id: str
    toolId: str
    name: str
    action: ToolCallAction
    args: _containers.ScalarMap[str, str]
    time: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., toolId: _Optional[str] = ..., name: _Optional[str] = ..., action: _Optional[_Union[ToolCallAction, str]] = ..., args: _Optional[_Mapping[str, str]] = ..., time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ConversationToolCallResult(_message.Message):
    __slots__ = ("id", "toolId", "name", "action", "result", "time")
    class ResultEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    TOOLID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    TIME_FIELD_NUMBER: _ClassVar[int]
    id: str
    toolId: str
    name: str
    action: ToolCallAction
    result: _containers.ScalarMap[str, str]
    time: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., toolId: _Optional[str] = ..., name: _Optional[str] = ..., action: _Optional[_Union[ToolCallAction, str]] = ..., result: _Optional[_Mapping[str, str]] = ..., time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ConversationMetric(_message.Message):
    __slots__ = ("assistantConversationId", "metrics")
    ASSISTANTCONVERSATIONID_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    assistantConversationId: int
    metrics: _containers.RepeatedCompositeFieldContainer[_common_pb2.Metric]
    def __init__(self, assistantConversationId: _Optional[int] = ..., metrics: _Optional[_Iterable[_Union[_common_pb2.Metric, _Mapping]]] = ...) -> None: ...

class ConversationMetadata(_message.Message):
    __slots__ = ("assistantConversationId", "metadata")
    ASSISTANTCONVERSATIONID_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    assistantConversationId: int
    metadata: _containers.RepeatedCompositeFieldContainer[_common_pb2.Metadata]
    def __init__(self, assistantConversationId: _Optional[int] = ..., metadata: _Optional[_Iterable[_Union[_common_pb2.Metadata, _Mapping]]] = ...) -> None: ...

class ConversationBridgeUserAudio(_message.Message):
    __slots__ = ("audio", "time")
    AUDIO_FIELD_NUMBER: _ClassVar[int]
    TIME_FIELD_NUMBER: _ClassVar[int]
    audio: bytes
    time: _timestamp_pb2.Timestamp
    def __init__(self, audio: _Optional[bytes] = ..., time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ConversationBridgeOperatorAudio(_message.Message):
    __slots__ = ("audio", "time")
    AUDIO_FIELD_NUMBER: _ClassVar[int]
    TIME_FIELD_NUMBER: _ClassVar[int]
    audio: bytes
    time: _timestamp_pb2.Timestamp
    def __init__(self, audio: _Optional[bytes] = ..., time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ConversationError(_message.Message):
    __slots__ = ("assistantConversationId", "message", "details")
    class DetailsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ASSISTANTCONVERSATIONID_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    DETAILS_FIELD_NUMBER: _ClassVar[int]
    assistantConversationId: int
    message: str
    details: _containers.ScalarMap[str, str]
    def __init__(self, assistantConversationId: _Optional[int] = ..., message: _Optional[str] = ..., details: _Optional[_Mapping[str, str]] = ...) -> None: ...

class ConversationEvent(_message.Message):
    __slots__ = ("id", "name", "data", "time")
    class DataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    TIME_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    data: _containers.ScalarMap[str, str]
    time: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., data: _Optional[_Mapping[str, str]] = ..., time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

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

class StreamConfig(_message.Message):
    __slots__ = ("audio", "text")
    AUDIO_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    audio: AudioConfig
    text: TextConfig
    def __init__(self, audio: _Optional[_Union[AudioConfig, _Mapping]] = ..., text: _Optional[_Union[TextConfig, _Mapping]] = ...) -> None: ...

class WebIdentity(_message.Message):
    __slots__ = ("userId",)
    USERID_FIELD_NUMBER: _ClassVar[int]
    userId: str
    def __init__(self, userId: _Optional[str] = ...) -> None: ...

class PhoneIdentity(_message.Message):
    __slots__ = ("phoneNumber",)
    PHONENUMBER_FIELD_NUMBER: _ClassVar[int]
    phoneNumber: str
    def __init__(self, phoneNumber: _Optional[str] = ...) -> None: ...

class ConversationInitialization(_message.Message):
    __slots__ = ("assistantConversationId", "assistant", "time", "metadata", "args", "options", "streamMode", "phone", "web")
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
    STREAMMODE_FIELD_NUMBER: _ClassVar[int]
    PHONE_FIELD_NUMBER: _ClassVar[int]
    WEB_FIELD_NUMBER: _ClassVar[int]
    assistantConversationId: int
    assistant: _common_pb2.AssistantDefinition
    time: _timestamp_pb2.Timestamp
    metadata: _containers.MessageMap[str, _any_pb2.Any]
    args: _containers.MessageMap[str, _any_pb2.Any]
    options: _containers.MessageMap[str, _any_pb2.Any]
    streamMode: StreamMode
    phone: PhoneIdentity
    web: WebIdentity
    def __init__(self, assistantConversationId: _Optional[int] = ..., assistant: _Optional[_Union[_common_pb2.AssistantDefinition, _Mapping]] = ..., time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., metadata: _Optional[_Mapping[str, _any_pb2.Any]] = ..., args: _Optional[_Mapping[str, _any_pb2.Any]] = ..., options: _Optional[_Mapping[str, _any_pb2.Any]] = ..., streamMode: _Optional[_Union[StreamMode, str]] = ..., phone: _Optional[_Union[PhoneIdentity, _Mapping]] = ..., web: _Optional[_Union[WebIdentity, _Mapping]] = ...) -> None: ...

class ConversationConfiguration(_message.Message):
    __slots__ = ("streamMode",)
    STREAMMODE_FIELD_NUMBER: _ClassVar[int]
    streamMode: StreamMode
    def __init__(self, streamMode: _Optional[_Union[StreamMode, str]] = ...) -> None: ...

class ConversationInterruption(_message.Message):
    __slots__ = ("id", "type", "time")
    class InterruptionType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = ()
        INTERRUPTION_TYPE_UNSPECIFIED: _ClassVar[ConversationInterruption.InterruptionType]
        INTERRUPTION_TYPE_VAD: _ClassVar[ConversationInterruption.InterruptionType]
        INTERRUPTION_TYPE_WORD: _ClassVar[ConversationInterruption.InterruptionType]
    INTERRUPTION_TYPE_UNSPECIFIED: ConversationInterruption.InterruptionType
    INTERRUPTION_TYPE_VAD: ConversationInterruption.InterruptionType
    INTERRUPTION_TYPE_WORD: ConversationInterruption.InterruptionType
    ID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    TIME_FIELD_NUMBER: _ClassVar[int]
    id: str
    type: ConversationInterruption.InterruptionType
    time: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., type: _Optional[_Union[ConversationInterruption.InterruptionType, str]] = ..., time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ConversationPlaybackPause(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class ConversationPlaybackContinue(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class ConversationPlaybackFlush(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class ConversationPlaybackComplete(_message.Message):
    __slots__ = ("id", "time")
    ID_FIELD_NUMBER: _ClassVar[int]
    TIME_FIELD_NUMBER: _ClassVar[int]
    id: str
    time: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ConversationDisconnection(_message.Message):
    __slots__ = ("type", "time")
    class DisconnectionType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = ()
        DISCONNECTION_TYPE_UNSPECIFIED: _ClassVar[ConversationDisconnection.DisconnectionType]
        DISCONNECTION_TYPE_TOOL: _ClassVar[ConversationDisconnection.DisconnectionType]
        DISCONNECTION_TYPE_USER: _ClassVar[ConversationDisconnection.DisconnectionType]
        DISCONNECTION_TYPE_IDLE_TIMEOUT: _ClassVar[ConversationDisconnection.DisconnectionType]
        DISCONNECTION_TYPE_MAX_DURATION: _ClassVar[ConversationDisconnection.DisconnectionType]
        DISCONNECTION_TYPE_ERROR: _ClassVar[ConversationDisconnection.DisconnectionType]
    DISCONNECTION_TYPE_UNSPECIFIED: ConversationDisconnection.DisconnectionType
    DISCONNECTION_TYPE_TOOL: ConversationDisconnection.DisconnectionType
    DISCONNECTION_TYPE_USER: ConversationDisconnection.DisconnectionType
    DISCONNECTION_TYPE_IDLE_TIMEOUT: ConversationDisconnection.DisconnectionType
    DISCONNECTION_TYPE_MAX_DURATION: ConversationDisconnection.DisconnectionType
    DISCONNECTION_TYPE_ERROR: ConversationDisconnection.DisconnectionType
    TYPE_FIELD_NUMBER: _ClassVar[int]
    TIME_FIELD_NUMBER: _ClassVar[int]
    type: ConversationDisconnection.DisconnectionType
    time: _timestamp_pb2.Timestamp
    def __init__(self, type: _Optional[_Union[ConversationDisconnection.DisconnectionType, str]] = ..., time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ConversationAssistantMessage(_message.Message):
    __slots__ = ("audio", "text", "id", "completed", "time")
    AUDIO_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_FIELD_NUMBER: _ClassVar[int]
    TIME_FIELD_NUMBER: _ClassVar[int]
    audio: bytes
    text: str
    id: str
    completed: bool
    time: _timestamp_pb2.Timestamp
    def __init__(self, audio: _Optional[bytes] = ..., text: _Optional[str] = ..., id: _Optional[str] = ..., completed: bool = ..., time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ConversationUserMessage(_message.Message):
    __slots__ = ("audio", "text", "id", "completed", "time")
    AUDIO_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_FIELD_NUMBER: _ClassVar[int]
    TIME_FIELD_NUMBER: _ClassVar[int]
    audio: bytes
    text: str
    id: str
    completed: bool
    time: _timestamp_pb2.Timestamp
    def __init__(self, audio: _Optional[bytes] = ..., text: _Optional[str] = ..., id: _Optional[str] = ..., completed: bool = ..., time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ConversationModeChange(_message.Message):
    __slots__ = ("mode", "time")
    class ModeType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = ()
        MODE_TYPE_UNSPECIFIED: _ClassVar[ConversationModeChange.ModeType]
        MODE_TYPE_AUDIO: _ClassVar[ConversationModeChange.ModeType]
        MODE_TYPE_TEXT: _ClassVar[ConversationModeChange.ModeType]
    MODE_TYPE_UNSPECIFIED: ConversationModeChange.ModeType
    MODE_TYPE_AUDIO: ConversationModeChange.ModeType
    MODE_TYPE_TEXT: ConversationModeChange.ModeType
    MODE_FIELD_NUMBER: _ClassVar[int]
    TIME_FIELD_NUMBER: _ClassVar[int]
    mode: ConversationModeChange.ModeType
    time: _timestamp_pb2.Timestamp
    def __init__(self, mode: _Optional[_Union[ConversationModeChange.ModeType, str]] = ..., time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class AssistantTalkRequest(_message.Message):
    __slots__ = ("initialization", "configuration", "message", "metadata", "metric", "disconnection", "toolCallResult", "bridgeOperatorAudio", "bridgeUserAudio", "playbackComplete")
    INITIALIZATION_FIELD_NUMBER: _ClassVar[int]
    CONFIGURATION_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    METRIC_FIELD_NUMBER: _ClassVar[int]
    DISCONNECTION_FIELD_NUMBER: _ClassVar[int]
    TOOLCALLRESULT_FIELD_NUMBER: _ClassVar[int]
    BRIDGEOPERATORAUDIO_FIELD_NUMBER: _ClassVar[int]
    BRIDGEUSERAUDIO_FIELD_NUMBER: _ClassVar[int]
    PLAYBACKCOMPLETE_FIELD_NUMBER: _ClassVar[int]
    initialization: ConversationInitialization
    configuration: ConversationConfiguration
    message: ConversationUserMessage
    metadata: ConversationMetadata
    metric: ConversationMetric
    disconnection: ConversationDisconnection
    toolCallResult: ConversationToolCallResult
    bridgeOperatorAudio: ConversationBridgeOperatorAudio
    bridgeUserAudio: ConversationBridgeUserAudio
    playbackComplete: ConversationPlaybackComplete
    def __init__(self, initialization: _Optional[_Union[ConversationInitialization, _Mapping]] = ..., configuration: _Optional[_Union[ConversationConfiguration, _Mapping]] = ..., message: _Optional[_Union[ConversationUserMessage, _Mapping]] = ..., metadata: _Optional[_Union[ConversationMetadata, _Mapping]] = ..., metric: _Optional[_Union[ConversationMetric, _Mapping]] = ..., disconnection: _Optional[_Union[ConversationDisconnection, _Mapping]] = ..., toolCallResult: _Optional[_Union[ConversationToolCallResult, _Mapping]] = ..., bridgeOperatorAudio: _Optional[_Union[ConversationBridgeOperatorAudio, _Mapping]] = ..., bridgeUserAudio: _Optional[_Union[ConversationBridgeUserAudio, _Mapping]] = ..., playbackComplete: _Optional[_Union[ConversationPlaybackComplete, _Mapping]] = ...) -> None: ...

class AssistantTalkResponse(_message.Message):
    __slots__ = ("code", "success", "initialization", "configuration", "interruption", "user", "assistant", "toolCall", "toolCallResult", "metadata", "metric", "disconnection", "event", "playbackPause", "playbackContinue", "playbackFlush", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    INITIALIZATION_FIELD_NUMBER: _ClassVar[int]
    CONFIGURATION_FIELD_NUMBER: _ClassVar[int]
    INTERRUPTION_FIELD_NUMBER: _ClassVar[int]
    USER_FIELD_NUMBER: _ClassVar[int]
    ASSISTANT_FIELD_NUMBER: _ClassVar[int]
    TOOLCALL_FIELD_NUMBER: _ClassVar[int]
    TOOLCALLRESULT_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    METRIC_FIELD_NUMBER: _ClassVar[int]
    DISCONNECTION_FIELD_NUMBER: _ClassVar[int]
    EVENT_FIELD_NUMBER: _ClassVar[int]
    PLAYBACKPAUSE_FIELD_NUMBER: _ClassVar[int]
    PLAYBACKCONTINUE_FIELD_NUMBER: _ClassVar[int]
    PLAYBACKFLUSH_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    initialization: ConversationInitialization
    configuration: ConversationConfiguration
    interruption: ConversationInterruption
    user: ConversationUserMessage
    assistant: ConversationAssistantMessage
    toolCall: ConversationToolCall
    toolCallResult: ConversationToolCallResult
    metadata: ConversationMetadata
    metric: ConversationMetric
    disconnection: ConversationDisconnection
    event: ConversationEvent
    playbackPause: ConversationPlaybackPause
    playbackContinue: ConversationPlaybackContinue
    playbackFlush: ConversationPlaybackFlush
    error: ConversationError
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., initialization: _Optional[_Union[ConversationInitialization, _Mapping]] = ..., configuration: _Optional[_Union[ConversationConfiguration, _Mapping]] = ..., interruption: _Optional[_Union[ConversationInterruption, _Mapping]] = ..., user: _Optional[_Union[ConversationUserMessage, _Mapping]] = ..., assistant: _Optional[_Union[ConversationAssistantMessage, _Mapping]] = ..., toolCall: _Optional[_Union[ConversationToolCall, _Mapping]] = ..., toolCallResult: _Optional[_Union[ConversationToolCallResult, _Mapping]] = ..., metadata: _Optional[_Union[ConversationMetadata, _Mapping]] = ..., metric: _Optional[_Union[ConversationMetric, _Mapping]] = ..., disconnection: _Optional[_Union[ConversationDisconnection, _Mapping]] = ..., event: _Optional[_Union[ConversationEvent, _Mapping]] = ..., playbackPause: _Optional[_Union[ConversationPlaybackPause, _Mapping]] = ..., playbackContinue: _Optional[_Union[ConversationPlaybackContinue, _Mapping]] = ..., playbackFlush: _Optional[_Union[ConversationPlaybackFlush, _Mapping]] = ..., error: _Optional[_Union[ConversationError, _Mapping]] = ...) -> None: ...

class CreateMessageMetricRequest(_message.Message):
    __slots__ = ("assistantId", "assistantConversationId", "messageId", "metrics")
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTCONVERSATIONID_FIELD_NUMBER: _ClassVar[int]
    MESSAGEID_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    assistantId: int
    assistantConversationId: int
    messageId: str
    metrics: _containers.RepeatedCompositeFieldContainer[_common_pb2.Metric]
    def __init__(self, assistantId: _Optional[int] = ..., assistantConversationId: _Optional[int] = ..., messageId: _Optional[str] = ..., metrics: _Optional[_Iterable[_Union[_common_pb2.Metric, _Mapping]]] = ...) -> None: ...

class CreateMessageMetricResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: _containers.RepeatedCompositeFieldContainer[_common_pb2.Metric]
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Iterable[_Union[_common_pb2.Metric, _Mapping]]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class CreateConversationMetricRequest(_message.Message):
    __slots__ = ("assistantId", "assistantConversationId", "metrics")
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTCONVERSATIONID_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    assistantId: int
    assistantConversationId: int
    metrics: _containers.RepeatedCompositeFieldContainer[_common_pb2.Metric]
    def __init__(self, assistantId: _Optional[int] = ..., assistantConversationId: _Optional[int] = ..., metrics: _Optional[_Iterable[_Union[_common_pb2.Metric, _Mapping]]] = ...) -> None: ...

class CreateConversationMetricResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: _containers.RepeatedCompositeFieldContainer[_common_pb2.Metric]
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Iterable[_Union[_common_pb2.Metric, _Mapping]]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class CreatePhoneCallRequest(_message.Message):
    __slots__ = ("assistant", "metadata", "args", "options", "fromNumber", "toNumber")
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
    ASSISTANT_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    ARGS_FIELD_NUMBER: _ClassVar[int]
    OPTIONS_FIELD_NUMBER: _ClassVar[int]
    FROMNUMBER_FIELD_NUMBER: _ClassVar[int]
    TONUMBER_FIELD_NUMBER: _ClassVar[int]
    assistant: _common_pb2.AssistantDefinition
    metadata: _containers.MessageMap[str, _any_pb2.Any]
    args: _containers.MessageMap[str, _any_pb2.Any]
    options: _containers.MessageMap[str, _any_pb2.Any]
    fromNumber: str
    toNumber: str
    def __init__(self, assistant: _Optional[_Union[_common_pb2.AssistantDefinition, _Mapping]] = ..., metadata: _Optional[_Mapping[str, _any_pb2.Any]] = ..., args: _Optional[_Mapping[str, _any_pb2.Any]] = ..., options: _Optional[_Mapping[str, _any_pb2.Any]] = ..., fromNumber: _Optional[str] = ..., toNumber: _Optional[str] = ...) -> None: ...

class CreatePhoneCallResponse(_message.Message):
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

class CreateBulkPhoneCallRequest(_message.Message):
    __slots__ = ("phoneCalls",)
    PHONECALLS_FIELD_NUMBER: _ClassVar[int]
    phoneCalls: _containers.RepeatedCompositeFieldContainer[CreatePhoneCallRequest]
    def __init__(self, phoneCalls: _Optional[_Iterable[_Union[CreatePhoneCallRequest, _Mapping]]] = ...) -> None: ...

class CreateBulkPhoneCallResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: _containers.RepeatedCompositeFieldContainer[_common_pb2.AssistantConversation]
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Iterable[_Union[_common_pb2.AssistantConversation, _Mapping]]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...
