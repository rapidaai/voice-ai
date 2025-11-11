package token_tiktoken_calculators

import (
	"strings"

	"github.com/pkoukk/tiktoken-go"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/tokens"
	"github.com/rapidaai/pkg/types"
)

type tikTokenCostCalculator struct {
	logger commons.Logger
	model  string
}

func NewTikTokenCostCalculator(
	logger commons.Logger,
	providerModel string) tokens.TokenCalculator {
	return &tikTokenCostCalculator{
		logger: logger,
		model:  providerModel,
	}
}

func (occ *tikTokenCostCalculator) Token(in []*types.Message, out *types.Message) []*types.Metric {
	mt := make([]*types.Metric, 0)
	ti, to := occ.token(occ.model, in, out)
	mt = append(mt, types.NewInputTokenMetric(ti))
	mt = append(mt, types.NewOutputTokenMetric(to))
	// If you want to add the total token count as well
	totalTokens := ti + to
	mt = append(mt, types.NewTotalTokenMetric(totalTokens))

	// If you have cost information, you can add those metrics too
	inputCost := float64(ti) * (occ.InputCost() / 1_000_000)
	outputCost := float64(to) * (occ.OutputCost() / 1_000_000)
	totalCost := inputCost + outputCost

	mt = append(mt, types.NewInputCostMetric(inputCost))
	mt = append(mt, types.NewOutputCostMetric(outputCost))
	mt = append(mt, types.NewTotalCostMetric(totalCost))
	return mt
}

func (occ *tikTokenCostCalculator) token(name string,
	in []*types.Message, out *types.Message) (int, int) {
	tkm, err := tiktoken.EncodingForModel(name)
	if err != nil {
		return 0, 0
	}

	var tokensPerMessage = 0
	switch name {
	case "gpt-3.5-turbo-0613",
		"gpt-3.5-turbo-16k-0613",
		"gpt-4-0314",
		"gpt-4-32k-0314",
		"gpt-4-0613",
		"gpt-4-32k-0613":
		tokensPerMessage = 3
	case "gpt-3.5-turbo-0301":
		tokensPerMessage = 4 // every message follows <|start|>{role/name}\n{content}<|end|>\n
	default:
		if strings.Contains(name, "gpt-3.5-turbo") {
			return occ.token("gpt-3.5-turbo-0613", in, out)
		} else if strings.Contains(name, "gpt-4") {
			return occ.token("gpt-4-0613", in, out)
		} else {
			return 0, 0
		}
	}
	inTokenCount := 0
	for _, message := range in {
		inTokenCount += tokensPerMessage
		inTokenCount += len(tkm.Encode(types.OnlyStringContent(message.GetContents()), nil, nil))
		inTokenCount += len(tkm.Encode(message.GetRole(), nil, nil))

	}
	// every reply is primed with <|start|>assistant<|message|>
	inTokenCount += 3
	outputToken := 0
	outputToken += len(tkm.Encode(types.OnlyStringContent(out.GetContents()), nil, nil))
	outputToken += len(tkm.Encode("assistant", nil, nil))

	return inTokenCount, outputToken
}

func (occ *tikTokenCostCalculator) InputCost() float64 {
	// for _, meta := range occ.providerModel.GetMetadatas() {
	// 	if meta.GetKey() == tokens.INPUT_COST_PER_MIL.String() {
	// 		if v, err := strconv.ParseFloat(meta.GetValue(), 64); err == nil {
	// 			return v
	// 		}
	// 	}
	// }
	return 0
}

func (occ *tikTokenCostCalculator) OutputCost() float64 {
	// for _, meta := range occ.providerModel.GetMetadatas() {
	// 	if meta.GetKey() == tokens.OUTPUT_COST_PER_MIL.String() {
	// 		if v, err := strconv.ParseFloat(meta.GetValue(), 64); err == nil {
	// 			return v
	// 		}
	// 	}
	// }
	return 0
}
