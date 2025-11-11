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
package types

import (
	"github.com/rapidaai/pkg/utils"
	lexatic_backend "github.com/rapidaai/protos"
)

type Content struct {
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// audio, image, video, text etc
	ContentType string `protobuf:"bytes,2,opt,name=contentType,proto3" json:"contentType,omitempty"`
	// from raw string to url all can be
	ContentFormat string `protobuf:"bytes,3,opt,name=contentFormat,proto3" json:"contentFormat,omitempty"`
	// actual content
	Content []byte `protobuf:"bytes,4,opt,name=content,proto3" json:"content,omitempty"`
	// added meta data incase you want to add something which is not supported in
	Meta map[string]interface{} `protobuf:"bytes,5,opt,name=meta,proto3" json:"meta,omitempty"`
}

func (c *Content) GetContent() []byte {
	return c.Content
}

func (c *Content) GetString() string {
	return string(c.Content)
}

func (c *Content) GetContentFormat() string {
	return string(c.ContentFormat)
}

func (c *Content) GetContentType() string {
	return string(c.ContentType)
}

func (c *Content) ToProto() *lexatic_backend.Content {
	protoC := &lexatic_backend.Content{}
	utils.Cast(c, protoC)
	return protoC
}

type Contents []*Content

func (m Contents) ToProto() []*lexatic_backend.Content {
	out := make([]*lexatic_backend.Content, len(m))
	for idx, k := range m {
		out[idx] = k.ToProto()
	}
	return out
}
