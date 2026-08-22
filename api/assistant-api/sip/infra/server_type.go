// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_infra

import (
	"fmt"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_core "github.com/rapidaai/api/assistant-api/sip/internal/core"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

type ServerState int32

const (
	ServerStateCreated ServerState = iota
	ServerStateRunning
	ServerStateStopped
)

type SIPRequestContext struct {
	Method       string
	CallID       string
	RequestURI   string
	FromIdentity string
	ToIdentity   string
	// Deprecated: use FromIdentity.
	FromURI string
	// Deprecated: use ToIdentity.
	ToURI           string
	APIKey          string
	AssistantID     string
	Auth            types.SimplePrinciple
	SDPInfo         *SDPMediaInfo
	Assistant       *internal_assistant_entity.Assistant
	VaultCredential *protos.VaultCredential
	Config          *Config
}

type SIPRequestIdentity struct {
	RequestURI   string
	FromIdentity string
	ToIdentity   string
}

type Middleware func(ctx *SIPRequestContext) error

type Server struct {
	inner *internal_core.Server
}

type ListenConfig struct {
	Address                 string    `json:"address" mapstructure:"address"`
	ExternalIP              string    `json:"external_ip" mapstructure:"external_ip"`
	AllowLoopbackExternalIP bool      `json:"allow_loopback_external_ip" mapstructure:"allow_loopback_external_ip"`
	Port                    int       `json:"port" mapstructure:"port"`
	Transport               Transport `json:"transport" mapstructure:"transport"`
}

func (c *ListenConfig) GetExternalIP() string {
	if c == nil {
		return ""
	}
	if c.ExternalIP != "" {
		return c.ExternalIP
	}
	return c.Address
}

func (c *ListenConfig) GetBindAddress() string {
	if c == nil {
		return ""
	}
	return c.Address
}

func (c *ListenConfig) GetListenAddr() string {
	if c == nil {
		return ""
	}
	return fmt.Sprintf("%s:%d", c.Address, c.Port)
}

func (c *ListenConfig) toCore() *internal_core.ListenConfig {
	if c == nil {
		return nil
	}
	return &internal_core.ListenConfig{
		Address:                 c.Address,
		ExternalIP:              c.ExternalIP,
		AllowLoopbackExternalIP: c.AllowLoopbackExternalIP,
		Port:                    c.Port,
		Transport:               internal_core.Transport(c.Transport),
	}
}

func listenConfigFromCore(config *internal_core.ListenConfig) *ListenConfig {
	if config == nil {
		return nil
	}
	return &ListenConfig{
		Address:                 config.Address,
		ExternalIP:              config.ExternalIP,
		AllowLoopbackExternalIP: config.AllowLoopbackExternalIP,
		Port:                    config.Port,
		Transport:               Transport(config.Transport),
	}
}

type ServerConfig struct {
	ListenConfig         *ListenConfig
	Middlewares          []Middleware
	Logger               commons.Logger
	RTPPortRangeStart    int
	RTPPortRangeEnd      int
	SymmetricRTP         bool
	IgnoreLocalAddrInSDP bool
}

func (c *ServerConfig) Validate() error {
	return c.toCore().Validate()
}

func (c *ServerConfig) toCore() *internal_core.ServerConfig {
	if c == nil {
		return nil
	}
	coreMiddlewares := make([]internal_core.Middleware, 0, len(c.Middlewares))
	for _, middleware := range c.Middlewares {
		current := middleware
		coreMiddlewares = append(coreMiddlewares, func(ctx *internal_core.SIPRequestContext) error {
			var config *Config
			if ctx.Config != nil {
				converted := configFromCore(ctx.Config)
				config = &converted
			}
			infraCtx := &SIPRequestContext{
				Method:          ctx.Method,
				CallID:          ctx.CallID,
				RequestURI:      ctx.RequestURI,
				FromIdentity:    ctx.FromIdentity,
				ToIdentity:      ctx.ToIdentity,
				FromURI:         ctx.FromIdentity,
				ToURI:           ctx.ToIdentity,
				SDPInfo:         sdpInfoFromCore(ctx.SDPInfo),
				APIKey:          ctx.APIKey,
				AssistantID:     ctx.AssistantID,
				Auth:            ctx.Auth,
				Assistant:       ctx.Assistant,
				VaultCredential: ctx.VaultCredential,
				Config:          config,
			}
			err := current(infraCtx)
			ctx.APIKey = infraCtx.APIKey
			ctx.AssistantID = infraCtx.AssistantID
			ctx.Auth = infraCtx.Auth
			ctx.Assistant = infraCtx.Assistant
			ctx.VaultCredential = infraCtx.VaultCredential
			if infraCtx.Config != nil {
				ctx.Config = infraCtx.Config.toCore()
			} else {
				ctx.Config = nil
			}
			return err
		})
	}
	return &internal_core.ServerConfig{
		ListenConfig:         c.ListenConfig.toCore(),
		Middlewares:          coreMiddlewares,
		Logger:               c.Logger,
		RTPPortRangeStart:    c.RTPPortRangeStart,
		RTPPortRangeEnd:      c.RTPPortRangeEnd,
		SymmetricRTP:         c.SymmetricRTP,
		IgnoreLocalAddrInSDP: c.IgnoreLocalAddrInSDP,
	}
}
