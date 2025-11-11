package tokens

import (
	"github.com/rapidaai/pkg/types"
)

type ModelMetadata string

const (
	CONTEXT_WINDOW      ModelMetadata = "context_window"
	INPUT_COST_PER_MIL  ModelMetadata = "input_cost_per_mil"
	OUTPUT_COST_PER_MIL ModelMetadata = "output_cost_per_mil"
)

func (mm ModelMetadata) String() string {
	return string(mm)
}

type TokenCalculator interface {
	Token(in []*types.Message, out *types.Message) []*types.Metric
}
