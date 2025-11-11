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
	"context"
	"encoding/json"
	"strconv"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/metadata"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

var (
	CLIENT_CTX_KEY CTX_KEY = "__client"
	REQUEST_ID_KEY         = "x-request-id"
)

type ClientInfo struct {
	UserAgent               string  `json:"user_agent"`
	Language                string  `json:"language"`
	Platform                string  `json:"platform"`
	ScreenWidth             int     `json:"screen_width"`
	ScreenHeight            int     `json:"screen_height"`
	WindowWidth             int     `json:"window_width"`
	WindowHeight            int     `json:"window_height"`
	Timezone                string  `json:"timezone"`
	ColorDepth              int     `json:"color_depth"`
	DeviceMemory            float64 `json:"device_memory,omitempty"`
	HardwareConcurrency     int64   `json:"hardware_concurrency,omitempty"`
	ConnectionType          string  `json:"connection_type,omitempty"`
	ConnectionEffectiveType string  `json:"connection_effective_type,omitempty"`
	CookiesEnabled          bool    `json:"cookies_enabled"`
	DoNotTrack              string  `json:"do_not_track,omitempty"`
	Referrer                string  `json:"referrer"`
	RemoteURL               string  `json:"remote_url"`
	Latitude                float64 `json:"latitude,omitempty"`
	Longitude               float64 `json:"longitude,omitempty"`
}

func (e *ClientInfo) ToJson() (string, error) {
	jsonData, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}

func NewClientInfoFromContext(ctx context.Context, logger commons.Logger) *ClientInfo {
	md := metadata.ExtractIncoming(ctx)
	clientInfo := &ClientInfo{}
	clientInfo.UserAgent = md.Get(utils.HEADER_USER_AGENT)
	clientInfo.Language = md.Get(utils.HEADER_LANGUAGE)
	clientInfo.Platform = md.Get(utils.HEADER_PLATFORM)
	clientInfo.ScreenWidth, _ = strconv.Atoi(md.Get(utils.HEADER_SCREEN_WIDTH))
	clientInfo.ScreenHeight, _ = strconv.Atoi(md.Get(utils.HEADER_SCREEN_HEIGHT))
	clientInfo.WindowWidth, _ = strconv.Atoi(md.Get(utils.HEADER_WINDOW_WIDTH))
	clientInfo.WindowHeight, _ = strconv.Atoi(md.Get(utils.HEADER_WINDOW_HEIGHT))
	clientInfo.Timezone = md.Get(utils.HEADER_TIMEZONE)
	clientInfo.ColorDepth, _ = strconv.Atoi(md.Get(utils.HEADER_COLOR_DEPTH))
	if memory, ok := strconv.ParseFloat(md.Get(utils.HEADER_DEVICE_MEMORY), 64); ok != nil {
		clientInfo.DeviceMemory = memory
	}

	if con, ok := strconv.ParseInt(md.Get(utils.HEADER_HARDWARE_CONCURRENCY), 10, 64); ok != nil {
		clientInfo.HardwareConcurrency = con
	}

	clientInfo.ConnectionType = md.Get(utils.HEADER_CONNECTION_TYPE)
	clientInfo.ConnectionEffectiveType = md.Get(utils.HEADER_CONNECTION_EFFECTIVE_TYPE)
	clientInfo.CookiesEnabled, _ = strconv.ParseBool(md.Get(utils.HEADER_COOKIES_ENABLED))
	clientInfo.DoNotTrack = md.Get(utils.HEADER_DO_NOT_TRACK)
	clientInfo.Referrer = md.Get(utils.HEADER_REFERRER)
	clientInfo.RemoteURL = md.Get(utils.HEADER_REMOTE_URL)

	if lat, ok := strconv.ParseFloat(md.Get(utils.HEADER_LATITUDE), 64); ok != nil {
		clientInfo.Latitude = lat
	}

	if lng, ok := strconv.ParseFloat(md.Get(utils.HEADER_LONGITUDE), 64); ok != nil {
		clientInfo.Longitude = lng
	}
	return clientInfo
}

func GetClientInfoFromGrpcContext(ctx context.Context) *ClientInfo {
	clt := ctx.Value(CLIENT_CTX_KEY)
	switch md := clt.(type) {
	case *ClientInfo:
		return md
	default:
		return nil
	}
}
