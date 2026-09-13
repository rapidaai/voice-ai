package internal_type

import (
	"testing"

	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/proto"
)

func TestOutputControlsUseProtobufMessages(t *testing.T) {
	controls := []proto.Message{
		&protos.ConversationPlaybackControl{Id: "response-1", Kind: protos.ConversationPlaybackControl_PAUSE},
		&protos.ConversationPlaybackControl{Id: "response-1", Kind: protos.ConversationPlaybackControl_CONTINUE},
		&protos.ConversationPlaybackControl{Id: "response-1", Kind: protos.ConversationPlaybackControl_FLUSH},
		&protos.ConversationPlaybackControl{Kind: protos.ConversationPlaybackControl_PAUSE},
		&protos.ConversationPlaybackControl{Kind: protos.ConversationPlaybackControl_CONTINUE},
		&protos.ConversationPlaybackControl{Kind: protos.ConversationPlaybackControl_FLUSH},
		&protos.ConversationPlaybackComplete{Id: "response-1"},
	}
	for _, control := range controls {
		encoded, err := proto.Marshal(control)
		if err != nil {
			t.Fatalf("marshal %T: %v", control, err)
		}
		decoded := control.ProtoReflect().Type().New().Interface()
		if err := proto.Unmarshal(encoded, decoded); err != nil {
			t.Fatalf("unmarshal %T: %v", control, err)
		}
		if !proto.Equal(control, decoded) {
			t.Fatalf("protobuf round trip changed %T", control)
		}
	}
}
