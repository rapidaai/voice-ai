import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
import app.bridges.artifacts.protos.common_pb2 as _common_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DeploymentAudioProvider(_message.Message):
    __slots__ = ("id", "audioProvider", "audioOptions", "status", "audioType")
    ID_FIELD_NUMBER: _ClassVar[int]
    AUDIOPROVIDER_FIELD_NUMBER: _ClassVar[int]
    AUDIOOPTIONS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    AUDIOTYPE_FIELD_NUMBER: _ClassVar[int]
    id: int
    audioProvider: str
    audioOptions: _containers.RepeatedCompositeFieldContainer[_common_pb2.Metadata]
    status: str
    audioType: str
    def __init__(self, id: _Optional[int] = ..., audioProvider: _Optional[str] = ..., audioOptions: _Optional[_Iterable[_Union[_common_pb2.Metadata, _Mapping]]] = ..., status: _Optional[str] = ..., audioType: _Optional[str] = ...) -> None: ...

class AssistantWebpluginDeployment(_message.Message):
    __slots__ = ("id", "assistantId", "name", "greeting", "mistake", "inputAudio", "outputAudio", "suggestion", "helpCenterEnabled", "productCatalogEnabled", "articleCatalogEnabled", "uploadFileEnabled", "createdDate", "updatedDate", "status", "maxSessionDuration", "idealTimeout", "idealTimeoutMessage", "idealTimeoutBackoff", "greetingInterruptible", "unclearInputTimeout", "unclearInputMessage")
    ID_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    GREETING_FIELD_NUMBER: _ClassVar[int]
    MISTAKE_FIELD_NUMBER: _ClassVar[int]
    INPUTAUDIO_FIELD_NUMBER: _ClassVar[int]
    OUTPUTAUDIO_FIELD_NUMBER: _ClassVar[int]
    SUGGESTION_FIELD_NUMBER: _ClassVar[int]
    HELPCENTERENABLED_FIELD_NUMBER: _ClassVar[int]
    PRODUCTCATALOGENABLED_FIELD_NUMBER: _ClassVar[int]
    ARTICLECATALOGENABLED_FIELD_NUMBER: _ClassVar[int]
    UPLOADFILEENABLED_FIELD_NUMBER: _ClassVar[int]
    CREATEDDATE_FIELD_NUMBER: _ClassVar[int]
    UPDATEDDATE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    MAXSESSIONDURATION_FIELD_NUMBER: _ClassVar[int]
    IDEALTIMEOUT_FIELD_NUMBER: _ClassVar[int]
    IDEALTIMEOUTMESSAGE_FIELD_NUMBER: _ClassVar[int]
    IDEALTIMEOUTBACKOFF_FIELD_NUMBER: _ClassVar[int]
    GREETINGINTERRUPTIBLE_FIELD_NUMBER: _ClassVar[int]
    UNCLEARINPUTTIMEOUT_FIELD_NUMBER: _ClassVar[int]
    UNCLEARINPUTMESSAGE_FIELD_NUMBER: _ClassVar[int]
    id: int
    assistantId: int
    name: str
    greeting: str
    mistake: str
    inputAudio: DeploymentAudioProvider
    outputAudio: DeploymentAudioProvider
    suggestion: _containers.RepeatedScalarFieldContainer[str]
    helpCenterEnabled: bool
    productCatalogEnabled: bool
    articleCatalogEnabled: bool
    uploadFileEnabled: bool
    createdDate: _timestamp_pb2.Timestamp
    updatedDate: _timestamp_pb2.Timestamp
    status: str
    maxSessionDuration: int
    idealTimeout: int
    idealTimeoutMessage: str
    idealTimeoutBackoff: int
    greetingInterruptible: bool
    unclearInputTimeout: float
    unclearInputMessage: str
    def __init__(self, id: _Optional[int] = ..., assistantId: _Optional[int] = ..., name: _Optional[str] = ..., greeting: _Optional[str] = ..., mistake: _Optional[str] = ..., inputAudio: _Optional[_Union[DeploymentAudioProvider, _Mapping]] = ..., outputAudio: _Optional[_Union[DeploymentAudioProvider, _Mapping]] = ..., suggestion: _Optional[_Iterable[str]] = ..., helpCenterEnabled: bool = ..., productCatalogEnabled: bool = ..., articleCatalogEnabled: bool = ..., uploadFileEnabled: bool = ..., createdDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updatedDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., status: _Optional[str] = ..., maxSessionDuration: _Optional[int] = ..., idealTimeout: _Optional[int] = ..., idealTimeoutMessage: _Optional[str] = ..., idealTimeoutBackoff: _Optional[int] = ..., greetingInterruptible: bool = ..., unclearInputTimeout: _Optional[float] = ..., unclearInputMessage: _Optional[str] = ...) -> None: ...

class AssistantPhoneDeployment(_message.Message):
    __slots__ = ("id", "assistantId", "greeting", "mistake", "inputAudio", "outputAudio", "phoneProviderName", "phoneOptions", "createdDate", "updatedDate", "status", "maxSessionDuration", "idealTimeout", "idealTimeoutMessage", "idealTimeoutBackoff", "greetingInterruptible", "unclearInputTimeout", "unclearInputMessage")
    ID_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    GREETING_FIELD_NUMBER: _ClassVar[int]
    MISTAKE_FIELD_NUMBER: _ClassVar[int]
    INPUTAUDIO_FIELD_NUMBER: _ClassVar[int]
    OUTPUTAUDIO_FIELD_NUMBER: _ClassVar[int]
    PHONEPROVIDERNAME_FIELD_NUMBER: _ClassVar[int]
    PHONEOPTIONS_FIELD_NUMBER: _ClassVar[int]
    CREATEDDATE_FIELD_NUMBER: _ClassVar[int]
    UPDATEDDATE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    MAXSESSIONDURATION_FIELD_NUMBER: _ClassVar[int]
    IDEALTIMEOUT_FIELD_NUMBER: _ClassVar[int]
    IDEALTIMEOUTMESSAGE_FIELD_NUMBER: _ClassVar[int]
    IDEALTIMEOUTBACKOFF_FIELD_NUMBER: _ClassVar[int]
    GREETINGINTERRUPTIBLE_FIELD_NUMBER: _ClassVar[int]
    UNCLEARINPUTTIMEOUT_FIELD_NUMBER: _ClassVar[int]
    UNCLEARINPUTMESSAGE_FIELD_NUMBER: _ClassVar[int]
    id: int
    assistantId: int
    greeting: str
    mistake: str
    inputAudio: DeploymentAudioProvider
    outputAudio: DeploymentAudioProvider
    phoneProviderName: str
    phoneOptions: _containers.RepeatedCompositeFieldContainer[_common_pb2.Metadata]
    createdDate: _timestamp_pb2.Timestamp
    updatedDate: _timestamp_pb2.Timestamp
    status: str
    maxSessionDuration: int
    idealTimeout: int
    idealTimeoutMessage: str
    idealTimeoutBackoff: int
    greetingInterruptible: bool
    unclearInputTimeout: float
    unclearInputMessage: str
    def __init__(self, id: _Optional[int] = ..., assistantId: _Optional[int] = ..., greeting: _Optional[str] = ..., mistake: _Optional[str] = ..., inputAudio: _Optional[_Union[DeploymentAudioProvider, _Mapping]] = ..., outputAudio: _Optional[_Union[DeploymentAudioProvider, _Mapping]] = ..., phoneProviderName: _Optional[str] = ..., phoneOptions: _Optional[_Iterable[_Union[_common_pb2.Metadata, _Mapping]]] = ..., createdDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updatedDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., status: _Optional[str] = ..., maxSessionDuration: _Optional[int] = ..., idealTimeout: _Optional[int] = ..., idealTimeoutMessage: _Optional[str] = ..., idealTimeoutBackoff: _Optional[int] = ..., greetingInterruptible: bool = ..., unclearInputTimeout: _Optional[float] = ..., unclearInputMessage: _Optional[str] = ...) -> None: ...

class AssistantWhatsappDeployment(_message.Message):
    __slots__ = ("id", "assistantId", "name", "greeting", "mistake", "inputAudio", "outputAudio", "whatsappProviderName", "whatsappOptions", "createdDate", "updatedDate", "status", "maxSessionDuration", "idealTimeout", "idealTimeoutMessage", "idealTimeoutBackoff", "greetingInterruptible", "unclearInputTimeout", "unclearInputMessage")
    ID_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    GREETING_FIELD_NUMBER: _ClassVar[int]
    MISTAKE_FIELD_NUMBER: _ClassVar[int]
    INPUTAUDIO_FIELD_NUMBER: _ClassVar[int]
    OUTPUTAUDIO_FIELD_NUMBER: _ClassVar[int]
    WHATSAPPPROVIDERNAME_FIELD_NUMBER: _ClassVar[int]
    WHATSAPPOPTIONS_FIELD_NUMBER: _ClassVar[int]
    CREATEDDATE_FIELD_NUMBER: _ClassVar[int]
    UPDATEDDATE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    MAXSESSIONDURATION_FIELD_NUMBER: _ClassVar[int]
    IDEALTIMEOUT_FIELD_NUMBER: _ClassVar[int]
    IDEALTIMEOUTMESSAGE_FIELD_NUMBER: _ClassVar[int]
    IDEALTIMEOUTBACKOFF_FIELD_NUMBER: _ClassVar[int]
    GREETINGINTERRUPTIBLE_FIELD_NUMBER: _ClassVar[int]
    UNCLEARINPUTTIMEOUT_FIELD_NUMBER: _ClassVar[int]
    UNCLEARINPUTMESSAGE_FIELD_NUMBER: _ClassVar[int]
    id: int
    assistantId: int
    name: str
    greeting: str
    mistake: str
    inputAudio: DeploymentAudioProvider
    outputAudio: DeploymentAudioProvider
    whatsappProviderName: str
    whatsappOptions: _containers.RepeatedCompositeFieldContainer[_common_pb2.Metadata]
    createdDate: _timestamp_pb2.Timestamp
    updatedDate: _timestamp_pb2.Timestamp
    status: str
    maxSessionDuration: int
    idealTimeout: int
    idealTimeoutMessage: str
    idealTimeoutBackoff: int
    greetingInterruptible: bool
    unclearInputTimeout: float
    unclearInputMessage: str
    def __init__(self, id: _Optional[int] = ..., assistantId: _Optional[int] = ..., name: _Optional[str] = ..., greeting: _Optional[str] = ..., mistake: _Optional[str] = ..., inputAudio: _Optional[_Union[DeploymentAudioProvider, _Mapping]] = ..., outputAudio: _Optional[_Union[DeploymentAudioProvider, _Mapping]] = ..., whatsappProviderName: _Optional[str] = ..., whatsappOptions: _Optional[_Iterable[_Union[_common_pb2.Metadata, _Mapping]]] = ..., createdDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updatedDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., status: _Optional[str] = ..., maxSessionDuration: _Optional[int] = ..., idealTimeout: _Optional[int] = ..., idealTimeoutMessage: _Optional[str] = ..., idealTimeoutBackoff: _Optional[int] = ..., greetingInterruptible: bool = ..., unclearInputTimeout: _Optional[float] = ..., unclearInputMessage: _Optional[str] = ...) -> None: ...

class AssistantDebuggerDeployment(_message.Message):
    __slots__ = ("id", "assistantId", "name", "greeting", "mistake", "inputAudio", "outputAudio", "createdDate", "updatedDate", "status", "maxSessionDuration", "idealTimeout", "idealTimeoutMessage", "idealTimeoutBackoff", "greetingInterruptible", "unclearInputTimeout", "unclearInputMessage")
    ID_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    GREETING_FIELD_NUMBER: _ClassVar[int]
    MISTAKE_FIELD_NUMBER: _ClassVar[int]
    INPUTAUDIO_FIELD_NUMBER: _ClassVar[int]
    OUTPUTAUDIO_FIELD_NUMBER: _ClassVar[int]
    CREATEDDATE_FIELD_NUMBER: _ClassVar[int]
    UPDATEDDATE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    MAXSESSIONDURATION_FIELD_NUMBER: _ClassVar[int]
    IDEALTIMEOUT_FIELD_NUMBER: _ClassVar[int]
    IDEALTIMEOUTMESSAGE_FIELD_NUMBER: _ClassVar[int]
    IDEALTIMEOUTBACKOFF_FIELD_NUMBER: _ClassVar[int]
    GREETINGINTERRUPTIBLE_FIELD_NUMBER: _ClassVar[int]
    UNCLEARINPUTTIMEOUT_FIELD_NUMBER: _ClassVar[int]
    UNCLEARINPUTMESSAGE_FIELD_NUMBER: _ClassVar[int]
    id: int
    assistantId: int
    name: str
    greeting: str
    mistake: str
    inputAudio: DeploymentAudioProvider
    outputAudio: DeploymentAudioProvider
    createdDate: _timestamp_pb2.Timestamp
    updatedDate: _timestamp_pb2.Timestamp
    status: str
    maxSessionDuration: int
    idealTimeout: int
    idealTimeoutMessage: str
    idealTimeoutBackoff: int
    greetingInterruptible: bool
    unclearInputTimeout: float
    unclearInputMessage: str
    def __init__(self, id: _Optional[int] = ..., assistantId: _Optional[int] = ..., name: _Optional[str] = ..., greeting: _Optional[str] = ..., mistake: _Optional[str] = ..., inputAudio: _Optional[_Union[DeploymentAudioProvider, _Mapping]] = ..., outputAudio: _Optional[_Union[DeploymentAudioProvider, _Mapping]] = ..., createdDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updatedDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., status: _Optional[str] = ..., maxSessionDuration: _Optional[int] = ..., idealTimeout: _Optional[int] = ..., idealTimeoutMessage: _Optional[str] = ..., idealTimeoutBackoff: _Optional[int] = ..., greetingInterruptible: bool = ..., unclearInputTimeout: _Optional[float] = ..., unclearInputMessage: _Optional[str] = ...) -> None: ...

class AssistantApiDeployment(_message.Message):
    __slots__ = ("id", "assistantId", "greeting", "mistake", "inputAudio", "outputAudio", "createdDate", "updatedDate", "status", "maxSessionDuration", "idealTimeout", "idealTimeoutMessage", "idealTimeoutBackoff", "greetingInterruptible", "unclearInputTimeout", "unclearInputMessage")
    ID_FIELD_NUMBER: _ClassVar[int]
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    GREETING_FIELD_NUMBER: _ClassVar[int]
    MISTAKE_FIELD_NUMBER: _ClassVar[int]
    INPUTAUDIO_FIELD_NUMBER: _ClassVar[int]
    OUTPUTAUDIO_FIELD_NUMBER: _ClassVar[int]
    CREATEDDATE_FIELD_NUMBER: _ClassVar[int]
    UPDATEDDATE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    MAXSESSIONDURATION_FIELD_NUMBER: _ClassVar[int]
    IDEALTIMEOUT_FIELD_NUMBER: _ClassVar[int]
    IDEALTIMEOUTMESSAGE_FIELD_NUMBER: _ClassVar[int]
    IDEALTIMEOUTBACKOFF_FIELD_NUMBER: _ClassVar[int]
    GREETINGINTERRUPTIBLE_FIELD_NUMBER: _ClassVar[int]
    UNCLEARINPUTTIMEOUT_FIELD_NUMBER: _ClassVar[int]
    UNCLEARINPUTMESSAGE_FIELD_NUMBER: _ClassVar[int]
    id: int
    assistantId: int
    greeting: str
    mistake: str
    inputAudio: DeploymentAudioProvider
    outputAudio: DeploymentAudioProvider
    createdDate: _timestamp_pb2.Timestamp
    updatedDate: _timestamp_pb2.Timestamp
    status: str
    maxSessionDuration: int
    idealTimeout: int
    idealTimeoutMessage: str
    idealTimeoutBackoff: int
    greetingInterruptible: bool
    unclearInputTimeout: float
    unclearInputMessage: str
    def __init__(self, id: _Optional[int] = ..., assistantId: _Optional[int] = ..., greeting: _Optional[str] = ..., mistake: _Optional[str] = ..., inputAudio: _Optional[_Union[DeploymentAudioProvider, _Mapping]] = ..., outputAudio: _Optional[_Union[DeploymentAudioProvider, _Mapping]] = ..., createdDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updatedDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., status: _Optional[str] = ..., maxSessionDuration: _Optional[int] = ..., idealTimeout: _Optional[int] = ..., idealTimeoutMessage: _Optional[str] = ..., idealTimeoutBackoff: _Optional[int] = ..., greetingInterruptible: bool = ..., unclearInputTimeout: _Optional[float] = ..., unclearInputMessage: _Optional[str] = ...) -> None: ...

class CreateAssistantDeploymentRequest(_message.Message):
    __slots__ = ("api", "debugger", "whatsapp", "phone", "plugin")
    API_FIELD_NUMBER: _ClassVar[int]
    DEBUGGER_FIELD_NUMBER: _ClassVar[int]
    WHATSAPP_FIELD_NUMBER: _ClassVar[int]
    PHONE_FIELD_NUMBER: _ClassVar[int]
    PLUGIN_FIELD_NUMBER: _ClassVar[int]
    api: AssistantApiDeployment
    debugger: AssistantDebuggerDeployment
    whatsapp: AssistantWhatsappDeployment
    phone: AssistantPhoneDeployment
    plugin: AssistantWebpluginDeployment
    def __init__(self, api: _Optional[_Union[AssistantApiDeployment, _Mapping]] = ..., debugger: _Optional[_Union[AssistantDebuggerDeployment, _Mapping]] = ..., whatsapp: _Optional[_Union[AssistantWhatsappDeployment, _Mapping]] = ..., phone: _Optional[_Union[AssistantPhoneDeployment, _Mapping]] = ..., plugin: _Optional[_Union[AssistantWebpluginDeployment, _Mapping]] = ...) -> None: ...

class GetAssistantApiDeploymentResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: AssistantApiDeployment
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Union[AssistantApiDeployment, _Mapping]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class GetAssistantPhoneDeploymentResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: AssistantPhoneDeployment
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Union[AssistantPhoneDeployment, _Mapping]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class GetAssistantWhatsappDeploymentResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: AssistantWhatsappDeployment
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Union[AssistantWhatsappDeployment, _Mapping]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class GetAssistantDebuggerDeploymentResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: AssistantDebuggerDeployment
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Union[AssistantDebuggerDeployment, _Mapping]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class GetAssistantWebpluginDeploymentResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: AssistantWebpluginDeployment
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Union[AssistantWebpluginDeployment, _Mapping]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class GetAllAssistantApiDeploymentResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error", "paginated")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    PAGINATED_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: _containers.RepeatedCompositeFieldContainer[AssistantApiDeployment]
    error: _common_pb2.Error
    paginated: _common_pb2.Paginated
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Iterable[_Union[AssistantApiDeployment, _Mapping]]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ..., paginated: _Optional[_Union[_common_pb2.Paginated, _Mapping]] = ...) -> None: ...

class GetAllAssistantPhoneDeploymentResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error", "paginated")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    PAGINATED_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: _containers.RepeatedCompositeFieldContainer[AssistantPhoneDeployment]
    error: _common_pb2.Error
    paginated: _common_pb2.Paginated
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Iterable[_Union[AssistantPhoneDeployment, _Mapping]]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ..., paginated: _Optional[_Union[_common_pb2.Paginated, _Mapping]] = ...) -> None: ...

class GetAllAssistantWhatsappDeploymentResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error", "paginated")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    PAGINATED_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: _containers.RepeatedCompositeFieldContainer[AssistantWhatsappDeployment]
    error: _common_pb2.Error
    paginated: _common_pb2.Paginated
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Iterable[_Union[AssistantWhatsappDeployment, _Mapping]]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ..., paginated: _Optional[_Union[_common_pb2.Paginated, _Mapping]] = ...) -> None: ...

class GetAllAssistantDebuggerDeploymentResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error", "paginated")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    PAGINATED_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: _containers.RepeatedCompositeFieldContainer[AssistantDebuggerDeployment]
    error: _common_pb2.Error
    paginated: _common_pb2.Paginated
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Iterable[_Union[AssistantDebuggerDeployment, _Mapping]]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ..., paginated: _Optional[_Union[_common_pb2.Paginated, _Mapping]] = ...) -> None: ...

class GetAllAssistantWebpluginDeploymentResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error", "paginated")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    PAGINATED_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: _containers.RepeatedCompositeFieldContainer[AssistantWebpluginDeployment]
    error: _common_pb2.Error
    paginated: _common_pb2.Paginated
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Iterable[_Union[AssistantWebpluginDeployment, _Mapping]]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ..., paginated: _Optional[_Union[_common_pb2.Paginated, _Mapping]] = ...) -> None: ...

class GetAssistantDeploymentRequest(_message.Message):
    __slots__ = ("assistantId",)
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    assistantId: int
    def __init__(self, assistantId: _Optional[int] = ...) -> None: ...

class GetAllAssistantDeploymentRequest(_message.Message):
    __slots__ = ("assistantId", "paginate", "criterias")
    ASSISTANTID_FIELD_NUMBER: _ClassVar[int]
    PAGINATE_FIELD_NUMBER: _ClassVar[int]
    CRITERIAS_FIELD_NUMBER: _ClassVar[int]
    assistantId: int
    paginate: _common_pb2.Paginate
    criterias: _containers.RepeatedCompositeFieldContainer[_common_pb2.Criteria]
    def __init__(self, assistantId: _Optional[int] = ..., paginate: _Optional[_Union[_common_pb2.Paginate, _Mapping]] = ..., criterias: _Optional[_Iterable[_Union[_common_pb2.Criteria, _Mapping]]] = ...) -> None: ...
