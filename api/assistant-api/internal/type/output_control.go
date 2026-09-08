package internal_type

type PauseOutput struct{}
type ContinueOutput struct{}
type FlushOutput struct{}

func (PauseOutput) ProtoMessage()    {}
func (ContinueOutput) ProtoMessage() {}
func (FlushOutput) ProtoMessage()    {}
