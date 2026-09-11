package internal_type

import (
	"testing"

	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestAssistantTalkPlaybackEnvelopeRoundTrip(t *testing.T) {
	messages := []proto.Message{
		&protos.AssistantTalkResponse{
			Code: 200, Success: true,
			Data: &protos.AssistantTalkResponse_PlaybackPause{PlaybackPause: &protos.ConversationPlaybackPause{Id: "response-1"}},
		},
		&protos.AssistantTalkResponse{
			Code: 200, Success: true,
			Data: &protos.AssistantTalkResponse_PlaybackContinue{PlaybackContinue: &protos.ConversationPlaybackContinue{Id: "response-1"}},
		},
		&protos.AssistantTalkResponse{
			Code: 200, Success: true,
			Data: &protos.AssistantTalkResponse_PlaybackFlush{PlaybackFlush: &protos.ConversationPlaybackFlush{Id: "response-1"}},
		},
		&protos.AssistantTalkRequest{
			Request: &protos.AssistantTalkRequest_PlaybackComplete{
				PlaybackComplete: &protos.ConversationPlaybackComplete{
					Id: "response-1", Time: &timestamppb.Timestamp{Seconds: 123},
				},
			},
		},
		&protos.AssistantTalkResponse{
			Data: &protos.AssistantTalkResponse_Assistant{
				Assistant: &protos.ConversationAssistantMessage{
					Id:        "response-1",
					Message:   &protos.ConversationAssistantMessage_Audio{Audio: []byte{1, 2}},
					Completed: true,
				},
			},
		},
	}
	for _, message := range messages {
		encoded, err := proto.Marshal(message)
		if err != nil {
			t.Fatalf("marshal %T: %v", message, err)
		}
		decoded := message.ProtoReflect().Type().New().Interface()
		if err := proto.Unmarshal(encoded, decoded); err != nil {
			t.Fatalf("unmarshal %T: %v", message, err)
		}
		if !proto.Equal(message, decoded) {
			t.Fatalf("wire round trip changed message: want %v, got %v", message, decoded)
		}
	}
}

func TestAssistantTalkPlaybackRejectsMalformedEnvelope(t *testing.T) {
	if err := proto.Unmarshal([]byte{0xff}, &protos.AssistantTalkRequest{}); err == nil {
		t.Fatal("malformed protobuf input must fail")
	}
}

func TestRetiredPlaybackFieldWireContract(t *testing.T) {
	for _, scenario := range []struct {
		message proto.Message
		retired protoreflect.FieldNumber
		fields  map[protoreflect.Name]protoreflect.FieldNumber
	}{
		{&protos.ConversationAssistantMessage{}, 5, map[protoreflect.Name]protoreflect.FieldNumber{"id": 2, "completed": 3, "time": 4, "audio": 10, "text": 11}},
		{&protos.ConversationPlaybackComplete{}, 3, map[protoreflect.Name]protoreflect.FieldNumber{"id": 1, "time": 2}},
	} {
		descriptor := scenario.message.ProtoReflect().Descriptor()
		if descriptor.Fields().ByName("playback_sequence") != nil || !descriptor.ReservedNames().Has("playback_sequence") || !descriptor.ReservedRanges().Has(scenario.retired) {
			t.Fatalf("%s must reserve the retired name and tag", descriptor.FullName())
		}
		for name, number := range scenario.fields {
			if field := descriptor.Fields().ByName(name); field == nil || field.Number() != number {
				t.Fatalf("%s field %s changed tag", descriptor.FullName(), name)
			}
		}
		encoded := protowire.AppendTag(nil, scenario.retired, protowire.VarintType)
		encoded = protowire.AppendVarint(encoded, 42)
		if err := proto.Unmarshal(encoded, scenario.message); err != nil {
			t.Fatal(err)
		}
		if len(scenario.message.ProtoReflect().GetUnknown()) == 0 {
			t.Fatal("old binary field must remain unknown, not authorize completion")
		}
		if err := protojson.Unmarshal([]byte(`{"playbackSequence":"42"}`), scenario.message); err == nil {
			t.Fatal("strict JSON decoding must reject the retired property")
		}
	}
	for _, control := range []proto.Message{&protos.ConversationPlaybackPause{}, &protos.ConversationPlaybackContinue{}, &protos.ConversationPlaybackFlush{}} {
		if control.ProtoReflect().Descriptor().Fields().ByName("playback_sequence") != nil {
			t.Fatalf("%T must remain message scoped", control)
		}
	}
}
