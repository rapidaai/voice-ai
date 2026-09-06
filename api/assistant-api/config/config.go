// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package config

import (
	"log"
	"os"
	"time"

	validator "github.com/go-playground/validator/v10"
	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/configs"
	"github.com/spf13/viper"
)

type SIPInboundConfig struct {
	// Supported answer modes:
	// - answer_immediately: no mode-specific argument is needed.
	// - answer_after_min_ring_ms: requires min_ring_duration.
	// max_ring_duration bounds pre-answer runtime readiness independently of answer_mode.
	// ack_timeout bounds the ACK wait after 200 OK independently of answer_mode.
	AnswerMode      string        `mapstructure:"answer_mode"`
	MinRingDuration time.Duration `mapstructure:"min_ring_duration"`
	MaxRingDuration time.Duration `mapstructure:"max_ring_duration"`
	ACKTimeout      time.Duration `mapstructure:"ack_timeout"`
}

// SIPConfig holds the SIP server configuration.
type SIPConfig struct {
	Server                  string           `mapstructure:"server"`
	InstanceID              string           `mapstructure:"instance_id" validate:"required"` // Unique identifier for this SIP server instance
	ExternalIP              string           `mapstructure:"external_ip"`                     // Public/reachable IP for outbound SDP and SIP Contact headers
	AllowLoopbackExternalIP bool             `mapstructure:"allow_loopback_external_ip"`
	Port                    int              `mapstructure:"port"`
	Transport               string           `mapstructure:"transport"`
	RTPPortRangeStart       int              `mapstructure:"rtp_port_range_start"`
	RTPPortRangeEnd         int              `mapstructure:"rtp_port_range_end"`
	SymmetricRTP            bool             `mapstructure:"symmetric_rtp"`
	IgnoreLocalAddrInSDP    bool             `mapstructure:"ignore_local_addr_in_sdp"`
	RegisterTimeout         time.Duration    `mapstructure:"register_timeout"`
	InviteTimeout           time.Duration    `mapstructure:"invite_timeout"`
	SessionTimeout          time.Duration    `mapstructure:"session_timeout"`
	MediaTimeoutInitial     time.Duration    `mapstructure:"media_timeout_initial"`
	MediaTimeout            time.Duration    `mapstructure:"media_timeout"`
	OutboundHealthGate      *bool            `mapstructure:"outbound_health_gate"`
	MaxConcurrentCalls      int              `mapstructure:"max_concurrent_calls"`
	CallAdmissionCPS        int              `mapstructure:"call_admission_cps"`
	CallAdmissionBurst      int              `mapstructure:"call_admission_burst"`
	Inbound                 SIPInboundConfig `mapstructure:"inbound"`
}

// WebRTCConfig holds WebRTC ICE configuration for production cloud deployments.
// On EC2, Pion only sees the private IP; set ExternalIP to the public/elastic IP
// so Pion advertises it in host candidates instead of the unreachable private IP.
type WebRTCConfig struct {
	ExternalIP         string            `mapstructure:"external_ip"`
	UDPPortRangeStart  int               `mapstructure:"udp_port_range_start"`
	UDPPortRangeEnd    int               `mapstructure:"udp_port_range_end"`
	ICEServers         []WebRTCICEServer `mapstructure:"ice_servers"`
	ICETransportPolicy string            `mapstructure:"ice_transport_policy"`
}

type WebRTCICEServer struct {
	URLs       []string `mapstructure:"urls"`
	Username   string   `mapstructure:"username"`
	Credential string   `mapstructure:"credential"`
}

type AudioSocketConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type AssistantConfig struct {
	config.AppConfig  `mapstructure:",squash"`
	PostgresConfig    configs.PostgresConfig    `mapstructure:"postgres" validate:"required"`
	RedisConfig       configs.RedisConfig       `mapstructure:"redis" validate:"required"`
	OpenSearchConfig  *configs.OpenSearchConfig `mapstructure:"opensearch"`
	TelemetryConfig   *configs.TelemetryConfig  `mapstructure:"telemetry"`
	WeaviateConfig    configs.WeaviateConfig    `mapstructure:"weaviate"`
	AssetStoreConfig  configs.AssetStoreConfig  `mapstructure:"asset_store" validate:"required"`
	SIPConfig         *SIPConfig                `mapstructure:"sip"`
	AudioSocketConfig *AudioSocketConfig        `mapstructure:"audiosocket"`
	WebRTCConfig      *WebRTCConfig             `mapstructure:"webrtc"`
}

// reading config and intializing configs for application
func InitConfig() (*viper.Viper, error) {
	vConfig := viper.New()

	path := os.Getenv("ENV_PATH")
	if path != "" {
		log.Printf("config path %v", path)
		vConfig.SetConfigFile(path)
	} else {
		vConfig.AddConfigPath("./env/")
		vConfig.SetConfigName("assistant")
		vConfig.SetConfigType("yaml")
	}

	if err := vConfig.ReadInConfig(); err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Error while reading the config: %v", err)
		}
	}

	return vConfig, nil
}

// Getting application config from viper
func GetApplicationConfig(v *viper.Viper) (*AssistantConfig, error) {
	var config AssistantConfig
	err := v.Unmarshal(&config)
	if err != nil {
		log.Printf("%+v\n", err)
		return nil, err
	}

	// valdating the app config
	validate := validator.New()
	err = validate.Struct(&config)
	if err != nil {
		log.Printf("%+v\n", err)
		return nil, err
	}
	return &config, nil
}
