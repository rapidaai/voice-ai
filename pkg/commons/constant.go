package commons

import (
	"fmt"
	"strings"

	lexatic_backend "github.com/rapidaai/protos"
)

var (
	// 10mb
	MaxRecvMsgSize = 1024 * 1024 * 10
	MaxSendMsgSize = 1024 * 1024 * 10

	SEPARATOR = "<|||>"

	//
	ENDPOINT_INDEX  = "endpoint-latest"
	ASSISTANT_INDEX = "assistant-latest"
	TELEMETRY_INDEX = "rapida-telemetry-latest"

	VERSION_PREFIX = "vrsn_"
	//

	DEVELOPMENT_TELEMETRY_INDEX = "rapida-telemetry-20250811"
	DEVELOPMENT_ASSISTANT_INDEX = "assistant-20240619"
	DEVELOPMENT_ENDPOINT_INDEX  = "endpoint-20240628"
)

// traceIndex
func TelemetryIndex(developement bool) string {
	if developement {
		return DEVELOPMENT_TELEMETRY_INDEX
	}
	return TELEMETRY_INDEX
}

// endpoint opensearch index
func EndpointIndex(developement bool) string {
	if developement {
		return DEVELOPMENT_ENDPOINT_INDEX
	}
	return ENDPOINT_INDEX
}

// assistant opensearch index
func AssistantIndex(developement bool) string {
	if developement {
		return DEVELOPMENT_ASSISTANT_INDEX
	}
	return ASSISTANT_INDEX
}

// knowledge opensearch index
func KnowledgeIndex(developement bool, org, prjm, kn uint64) string {
	if developement {
		return fmt.Sprintf("dev__vs__%d__%d__%d", org, prjm, kn)
	}
	return fmt.Sprintf("prod__vs__%d__%d__%d", org, prjm, kn)
}

// al
type ResponseContentType string
type ResponseContentFormat string

func (rct ResponseContentType) String() string {
	return string(rct)
}

func (rct ResponseContentFormat) String() string {
	return string(rct)
}

type MessageContent lexatic_backend.Message

func (mc *MessageContent) StringContent() string {
	var builder strings.Builder
	if len(mc.Contents) == 0 {
		return ""
	}
	for _, c := range mc.Contents {
		if ResponseContentType(c.GetContentType()) == TEXT_CONTENT {
			if ResponseContentFormat(c.GetContentFormat()) == TEXT_CONTENT_FORMAT_RAW {
				builder.Write(c.Content)
			}
		}
	}
	return builder.String()
}

func ToMessageContent(msg *lexatic_backend.Message) *MessageContent {
	// copy the message avoid locking
	return &MessageContent{
		Role:     msg.GetRole(),
		Contents: msg.GetContents(),
	}
}
