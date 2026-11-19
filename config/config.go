// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package config

import (
	"crypto/hmac"
	"crypto/sha256"
)

const (
	serviceAuthenticationLabel             = "rapida/service-authentication/v1"
	organizationCredentialFingerprintLabel = "rapida/organization-credential-fingerprint/v1"
)

type RegionalHost struct {
	Region string `mapstructure:"region" validate:"required"`
	Host   string `mapstructure:"host" validate:"required"`
}

type ServiceHostConfig struct {
	Host    string         `mapstructure:"host" validate:"required"`
	Public  string         `mapstructure:"public"`
	Regions []RegionalHost `mapstructure:"regions"`
}

type AppConfig struct {
	//
	Name      string `mapstructure:"service_name" validate:"required"`
	ServiceID uint64 `mapstructure:"service_id" validate:"required"`
	Version   string `mapstructure:"version"`
	Host      string `mapstructure:"host" validate:"required"`
	Env       string `mapstructure:"env" validate:"required"`
	Port      int    `mapstructure:"port" validate:"required"`
	LogLevel  string `mapstructure:"log_level" validate:"required"`
	Secret    string `mapstructure:"secret" validate:"required"`

	Integration ServiceHostConfig `mapstructure:"integration"`
	Endpoint    ServiceHostConfig `mapstructure:"endpoint"`
	Assistant   ServiceHostConfig `mapstructure:"assistant"`
	Web         ServiceHostConfig `mapstructure:"web"`
	Document    ServiceHostConfig `mapstructure:"document"`
	Ui          ServiceHostConfig `mapstructure:"ui"`
}

func (cfg *AppConfig) IsDevelopment() bool {
	return cfg.Env != "production"
}

func (cfg *AppConfig) BaseUrl() string {
	return cfg.Ui.Host
}

// ServiceAuthenticationKey derives the internal assertion key.
func (cfg *AppConfig) ServiceAuthenticationKey() []byte {
	mac := hmac.New(sha256.New, []byte(cfg.Secret))
	_, _ = mac.Write([]byte(serviceAuthenticationLabel))
	return mac.Sum(nil)
}

// OrganizationCredentialFingerprintKey derives the Organization credential HMAC key.
func (cfg *AppConfig) OrganizationCredentialFingerprintKey() []byte {
	mac := hmac.New(sha256.New, []byte(cfg.Secret))
	_, _ = mac.Write([]byte(organizationCredentialFingerprintLabel))
	return mac.Sum(nil)
}
