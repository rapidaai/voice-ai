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
			Data: &protos.AssistantTalkResponse_PlaybackControl{PlaybackControl: &protos.ConversationPlaybackControl{Id: "response-1", Kind: protos.ConversationPlaybackControl_PAUSE}},
		},
		&protos.AssistantTalkResponse{
			Code: 200, Success: true,
			Data: &protos.AssistantTalkResponse_PlaybackControl{PlaybackControl: &protos.ConversationPlaybackControl{Id: "response-1", Kind: protos.ConversationPlaybackControl_CONTINUE}},
		},
		&protos.AssistantTalkResponse{
			Code: 200, Success: true,
			Data: &protos.AssistantTalkResponse_PlaybackControl{PlaybackControl: &protos.ConversationPlaybackControl{Id: "response-1", Kind: protos.ConversationPlaybackControl_FLUSH}},
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

func TestPlaybackControlDescriptorAndRoundTrip(t *testing.T) {
	descriptor := (&protos.ConversationPlaybackControl{}).ProtoReflect().Descriptor()
	if descriptor.Fields().Len() != 2 {
		t.Fatalf("control must contain only id and kind: %v", descriptor.Fields())
	}
	id := descriptor.Fields().ByName("id")
	if id == nil || id.Number() != 1 || id.Kind() != protoreflect.StringKind {
		t.Fatal("control id must be a string at tag 1")
	}
	kind := descriptor.Fields().ByName("kind")
	if kind == nil || kind.Number() != 2 || kind.Kind() != protoreflect.EnumKind {
		t.Fatal("control kind must be an enum at tag 2")
	}
	enum := kind.Enum()
	if enum.Name() != "Kind" || enum.Parent().FullName() != descriptor.FullName() || enum.Values().Len() != 4 {
		t.Fatal("control must own the four-value nested Kind enum")
	}
	for name, number := range map[protoreflect.Name]protoreflect.EnumNumber{
		"KIND_UNSPECIFIED": 0, "PAUSE": 1, "CONTINUE": 2, "FLUSH": 3,
	} {
		if value := enum.Values().ByName(name); value == nil || value.Number() != number {
			t.Fatalf("control kind %s must have value %d", name, number)
		}
	}
	responseDescriptor := (&protos.AssistantTalkResponse{}).ProtoReflect().Descriptor()
	field := responseDescriptor.Fields().ByName("playbackControl")
	if field == nil || field.Number() != 24 || field.Message().FullName() != descriptor.FullName() || field.ContainingOneof() == nil || field.ContainingOneof().Name() != "data" {
		t.Fatal("response playbackControl must occupy data oneof tag 24")
	}
	completion := (&protos.AssistantTalkRequest{}).ProtoReflect().Descriptor().Fields().ByName("playbackComplete")
	if completion == nil || completion.Number() != 10 || completion.Message().FullName() != (&protos.ConversationPlaybackComplete{}).ProtoReflect().Descriptor().FullName() {
		t.Fatal("request playbackComplete must retain tag 10 and its message type")
	}
	for _, value := range []protos.ConversationPlaybackControl_Kind{
		protos.ConversationPlaybackControl_KIND_UNSPECIFIED, protos.ConversationPlaybackControl_PAUSE,
		protos.ConversationPlaybackControl_CONTINUE, protos.ConversationPlaybackControl_FLUSH,
		protos.ConversationPlaybackControl_Kind(99),
	} {
		t.Run(value.String(), func(t *testing.T) {
			message := &protos.AssistantTalkResponse{
				Data: &protos.AssistantTalkResponse_PlaybackControl{PlaybackControl: &protos.ConversationPlaybackControl{Id: "response-1", Kind: value}},
			}
			for _, format := range []string{"binary", "json"} {
				t.Run(format, func(t *testing.T) {
					var encoded []byte
					var err error
					decoded := &protos.AssistantTalkResponse{}
					if format == "binary" {
						encoded, err = proto.Marshal(message)
					} else {
						encoded, err = protojson.Marshal(message)
					}
					if err != nil {
						t.Fatal(err)
					}
					if format == "binary" {
						err = proto.Unmarshal(encoded, decoded)
					} else {
						err = protojson.Unmarshal(encoded, decoded)
					}
					if err != nil {
						t.Fatal(err)
					}
					if !proto.Equal(message, decoded) || decoded.GetPlaybackControl().GetKind() != value || decoded.GetPlaybackControl().GetId() != "response-1" {
						t.Fatalf("control round trip changed kind or id: %v", decoded)
					}
				})
			}
		})
	}
}

func TestRetiredPlaybackControlEnvelopeFieldsRemainReserved(t *testing.T) {
	descriptor := (&protos.AssistantTalkResponse{}).ProtoReflect().Descriptor()
	for name, number := range map[protoreflect.Name]protoreflect.FieldNumber{
		"playbackPause": 21, "playbackContinue": 22, "playbackFlush": 23,
	} {
		if descriptor.Fields().ByName(name) != nil || descriptor.Fields().ByNumber(number) != nil || !descriptor.ReservedNames().Has(name) || !descriptor.ReservedRanges().Has(number) {
			t.Fatalf("response must reserve retired field %s at tag %d", name, number)
		}
		payload := protowire.AppendTag(nil, 1, protowire.BytesType)
		payload = protowire.AppendString(payload, "response-1")
		encoded := protowire.AppendTag(nil, number, protowire.BytesType)
		encoded = protowire.AppendBytes(encoded, payload)
		decoded := &protos.AssistantTalkResponse{}
		if err := proto.Unmarshal(encoded, decoded); err != nil {
			t.Fatal(err)
		}
		if decoded.GetData() != nil || len(decoded.ProtoReflect().GetUnknown()) == 0 {
			t.Fatalf("retired tag %d must remain unknown, not become a control", number)
		}
		if err := protojson.Unmarshal([]byte(`{"`+string(name)+`":{"id":"response-1"}}`), &protos.AssistantTalkResponse{}); err == nil {
			t.Fatalf("strict JSON decoding must reject %s", name)
		}
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
		// The source intentionally removed these reservations; retired fields must still stay absent.
		if descriptor.Fields().ByName("playback_sequence") != nil || descriptor.Fields().ByNumber(scenario.retired) != nil {
			t.Fatalf("%s must not reuse the retired name or tag", descriptor.FullName())
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
	if (&protos.ConversationPlaybackControl{}).ProtoReflect().Descriptor().Fields().ByName("playback_sequence") != nil {
		t.Fatal("playback control must remain message scoped")
	}
}
