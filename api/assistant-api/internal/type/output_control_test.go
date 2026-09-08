package internal_type

import "testing"

func TestOutputControlsUseInternalStreamContract(t *testing.T) {
	controls := []Stream{PauseOutput{}, ContinueOutput{}, FlushOutput{}}
	for _, control := range controls {
		control.ProtoMessage()
		if _, public := control.(interface{ Reset() }); public {
			t.Fatalf("internal control %T must not implement the public protobuf contract", control)
		}
	}
}
