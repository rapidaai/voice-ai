package internal_adapter_request_customizers

/*
 *  Copyright (c) 2024. Rapida
 *
 *  Permission is hereby granted, free of charge, to any person obtaining a copy
 *  of this software and associated documentation files (the "Software"), to deal
 *  in the Software without restriction, including without limitation the rights
 *  to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 *  copies of the Software, and to permit persons to whom the Software is
 *  furnished to do so, subject to the following conditions:
 *
 *  The above copyright notice and this permission notice shall be included in
 *  all copies or substantial portions of the Software.
 *
 *  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 *  IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 *  FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 *  AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 *  LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 *  OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 *  THE SOFTWARE.
 *
 *  Author: Prashant <prashant@rapida.ai>
 *
 */
import (
	internal_adapter_requests "github.com/rapidaai/api/assistant-api/internal/adapters/requests"
	"github.com/rapidaai/pkg/utils"
	lexatic_backend "github.com/rapidaai/protos"
)

type customizer struct {
	args     map[string]interface{}
	options  map[string]interface{}
	metadata map[string]interface{}
}

func NewRequestBaseCustomizer(req *lexatic_backend.AssistantConversationConfiguration) (internal_adapter_requests.Customization, error) {
	arg, err := utils.AnyMapToInterfaceMap(req.GetArgs())
	if err != nil {
		return nil, err
	}
	opts, err := utils.AnyMapToInterfaceMap(req.GetOptions())
	if err != nil {
		return nil, err
	}
	mtd, err := utils.AnyMapToInterfaceMap(req.GetMetadata())
	if err != nil {
		return nil, err
	}
	return &customizer{
		metadata: mtd,
		options:  opts,
		args:     arg,
	}, nil

}

func (ctmzr *customizer) GetMetadata() map[string]interface{} {
	return ctmzr.metadata
}

func (ctmzr *customizer) GetOptions() map[string]interface{} {
	return ctmzr.options
}

func (ctmzr *customizer) GetArgs() map[string]interface{} {
	return ctmzr.args
}
