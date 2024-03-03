package client_evals

type perplexityEval struct {
}

func NewPerplexityEval() {
	return
}

func (perplexityEval) Calculate(prompt string) float64 {
	return 0
	// var nlls []float64
	// for _, negLogLikelihood := range tokenLogProbs {
	// 	if negLogLikelihood == nil { // default to -100, handles the initial token case
	// 		negLogLikelihood = -100
	// 	}
	// 	nlls = append(nlls, *negLogLikelihood)
	// }

	// var sum float64
	// for _, nll := range nlls {
	// 	sum += nll
	// }

	// perplexity := math.Exp(sum / float64(len(tokenLogProbs)))
	// return perplexity
}
