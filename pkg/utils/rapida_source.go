package utils

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
	"encoding/json"
	"log"
	"strings"
)

type RapidaSource string

const (
	WebPlugin RapidaSource = "web-plugin"
	Debugger  RapidaSource = "debugger"
	SDK       RapidaSource = "sdk"
	PhoneCall RapidaSource = "phone-call"
	Whatsapp  RapidaSource = "whatsapp"
)

// Get returns the string value of the RapidaRegion
func (r RapidaSource) Get() string {
	return string(r)
}

// FromStr returns the corresponding RapidaSource for a given string,
// or WebPlugin if the string does not match any source.
func FromSourceStr(label string) RapidaSource {
	switch strings.ToLower(label) {
	case "web-plugin":
		return WebPlugin
	case "debugger":
		return Debugger
	case "sdk":
		return SDK
	case "phone-call":
		return PhoneCall
	case "whatsapp":
		return Whatsapp
	default:
		log.Printf("%s The source is not supported. Supported sources are 'web-plugin', 'debugger', 'sdk', 'phone-call', and 'whatsapp'.", label)
		return Debugger
	}
}

func (c RapidaSource) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(c))
}
