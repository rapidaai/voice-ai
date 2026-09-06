// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

//go:build sipintegration && freeswitch

package freeswitch_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
	"github.com/rapidaai/pkg/commons"
	"github.com/stretchr/testify/require"
)

const (
	freeSWITCHEnabledEnv = "RAPIDA_SIP_FREESWITCH_ENABLE"

	defaultFreeSWITCHCLI      = "/opt/homebrew/opt/freeswitch/bin/fs_cli"
	defaultFreeSWITCHHost     = "127.0.0.1"
	defaultFreeSWITCHCLIPort  = "8022"
	defaultFreeSWITCHPassword = "ClueCon"
	defaultFreeSWITCHSIPPort  = 5060
	defaultSIPListenAddress   = "127.0.0.1"
	defaultSIPRTPPortStart    = 21000
	defaultSIPRTPPortEnd      = 21200

	freeSWITCHCommandTimeout = 5 * time.Second
	callSetupTimeout         = 20 * time.Second
	callTeardownTimeout      = 10 * time.Second
)

type freeSWITCHIntegrationConfig struct {
	fsCLI       string
	fsHost      string
	fsCLIPort   string
	fsPassword  string
	fsSIPHost   string
	fsSIPPort   int
	listenHost  string
	externalIP  string
	listenPort  int
	rtpPortFrom int
	rtpPortTo   int
}

type freeSWITCHHarness struct {
	t          *testing.T
	config     freeSWITCHIntegrationConfig
	logger     commons.Logger
	server     *sip_runtime.Server
	sipConfig  *sip_runtime.Config
	cancelFunc context.CancelFunc
}

func newFreeSWITCHHarness(t *testing.T, credentials sipCredentialConfig) *freeSWITCHHarness {
	t.Helper()

	config := loadFreeSWITCHIntegrationConfig(t)
	logger, err := commons.NewApplicationLogger(
		commons.Name("sip-freeswitch-integration"),
		commons.Level("debug"),
	)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	sipConfig := &sip_runtime.Config{
		Server:            config.fsSIPHost,
		Username:          credentials.username,
		Password:          credentials.password,
		Realm:             credentials.realm,
		Domain:            credentials.domain,
		Port:              config.fsSIPPort,
		Transport:         sip_runtime.TransportUDP,
		RTPPortRangeStart: config.rtpPortFrom,
		RTPPortRangeEnd:   config.rtpPortTo,
		InviteTimeout:     callSetupTimeout,
		SessionTimeout:    time.Minute,
	}

	server, err := sip_runtime.NewServer(ctx, &sip_runtime.ServerConfig{
		ListenConfig: &sip_runtime.ListenConfig{
			Address:                 config.listenHost,
			ExternalIP:              config.externalIP,
			AllowLoopbackExternalIP: true,
			Port:                    config.listenPort,
			Transport:               sip_runtime.TransportUDP,
		},
		Middlewares: []sip_runtime.Middleware{
			func(ctx *sip_runtime.SIPRequestContext) error {
				ctx.Config = sipConfig
				return nil
			},
		},
		Logger:            logger,
		RTPPortRangeStart: config.rtpPortFrom,
		RTPPortRangeEnd:   config.rtpPortTo,
	})
	require.NoError(t, err)

	require.NoError(t, server.Start())

	harness := &freeSWITCHHarness{
		t:          t,
		config:     config,
		logger:     logger,
		server:     server,
		sipConfig:  sipConfig,
		cancelFunc: cancel,
	}
	t.Cleanup(harness.close)
	harness.requireFreeSWITCHReady()
	return harness
}

func loadFreeSWITCHIntegrationConfig(t *testing.T) freeSWITCHIntegrationConfig {
	t.Helper()
	if os.Getenv(freeSWITCHEnabledEnv) != "1" {
		t.Skipf("set %s=1 to run FreeSWITCH SIP integration tests", freeSWITCHEnabledEnv)
	}

	listenPort := integrationEnvInt(t, "RAPIDA_SIP_LISTEN_PORT", 0)
	if listenPort == 0 {
		listenPort = reserveUDPPort(t, defaultSIPListenAddress)
	}

	return freeSWITCHIntegrationConfig{
		fsCLI:       integrationEnv("FREESWITCH_CLI", defaultFreeSWITCHCLI),
		fsHost:      integrationEnv("FREESWITCH_HOST", defaultFreeSWITCHHost),
		fsCLIPort:   integrationEnv("FREESWITCH_CLI_PORT", defaultFreeSWITCHCLIPort),
		fsPassword:  integrationEnv("FREESWITCH_PASSWORD", defaultFreeSWITCHPassword),
		fsSIPHost:   integrationEnv("FREESWITCH_SIP_HOST", defaultFreeSWITCHHost),
		fsSIPPort:   integrationEnvInt(t, "FREESWITCH_SIP_PORT", defaultFreeSWITCHSIPPort),
		listenHost:  integrationEnv("RAPIDA_SIP_LISTEN_ADDRESS", defaultSIPListenAddress),
		externalIP:  integrationEnv("RAPIDA_SIP_EXTERNAL_IP", defaultSIPListenAddress),
		listenPort:  listenPort,
		rtpPortFrom: integrationEnvInt(t, "RAPIDA_SIP_RTP_PORT_START", defaultSIPRTPPortStart),
		rtpPortTo:   integrationEnvInt(t, "RAPIDA_SIP_RTP_PORT_END", defaultSIPRTPPortEnd),
	}
}

func (h *freeSWITCHHarness) requireFreeSWITCHReady() {
	h.t.Helper()
	output, err := h.runFreeSWITCHCommand("status")
	require.NoErrorf(h.t, err, "FreeSWITCH status failed: %s", output)
}

func (h *freeSWITCHHarness) runFreeSWITCHCommand(command string) (string, error) {
	h.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), freeSWITCHCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(
		ctx,
		h.config.fsCLI,
		"-H", h.config.fsHost,
		"-P", h.config.fsCLIPort,
		"-p", h.config.fsPassword,
		"-x", command,
	)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (h *freeSWITCHHarness) registrationClient() *sip_runtime.RegistrationClient {
	return sip_runtime.NewRegistrationClient(h.server.Client(), h.server.GetListenConfig(), h.logger)
}

func (h *freeSWITCHHarness) originateRegisteredInboundCall(registeredDID, callerUser string) string {
	h.t.Helper()
	endpoint := "user/" + freeSWITCHUser(registeredDID)
	command := fmt.Sprintf(
		"originate {origination_caller_id_number=%s,origination_caller_id_name=RapidaIntegration}%s &park()",
		callerUser,
		endpoint,
	)
	output, err := h.runFreeSWITCHCommand(command)
	require.NoErrorf(h.t, err, "FreeSWITCH originate failed: %s", output)
	return parseFreeSWITCHOKUUID(h.t, output)
}

func (h *freeSWITCHHarness) originateTwilioElasticInboundCall(profile twilioElasticTrunkConfig) string {
	h.t.Helper()
	endpoint := fmt.Sprintf("sofia/internal/%s@%s:%d", freeSWITCHUser(profile.inboundDID), h.config.externalIP, h.config.listenPort)
	command := fmt.Sprintf(
		"originate {origination_caller_id_number=%s,origination_caller_id_name=TwilioElastic,sip_h_X-Twilio-CallSid=%s,sip_h_X-Twilio-AccountSid=%s,sip_h_X-Twilio-Elastic-Trunk-SID=%s}%s &park()",
		profile.callerUser,
		profile.callSID,
		profile.accountSID,
		profile.trunkSID,
		endpoint,
	)
	output, err := h.runFreeSWITCHCommand(command)
	require.NoErrorf(h.t, err, "FreeSWITCH Twilio-style originate failed: %s", output)
	return parseFreeSWITCHOKUUID(h.t, output)
}

func (h *freeSWITCHHarness) originateDirectInboundCall(routeUser, callerUser string, headers map[string]string) string {
	h.t.Helper()

	channelVariables := []string{
		"origination_caller_id_number=" + callerUser,
		"origination_caller_id_name=RapidaIntegration",
	}
	headerNames := make([]string, 0, len(headers))
	for name := range headers {
		headerNames = append(headerNames, name)
	}
	sort.Strings(headerNames)
	for _, name := range headerNames {
		channelVariables = append(channelVariables, "sip_h_"+name+"="+headers[name])
	}

	endpoint := fmt.Sprintf("sofia/internal/sip:%s@%s:%d", routeUser, h.config.externalIP, h.config.listenPort)
	command := fmt.Sprintf("originate {%s}%s &park()", strings.Join(channelVariables, ","), endpoint)
	output, err := h.runFreeSWITCHCommand(command)
	require.NoErrorf(h.t, err, "FreeSWITCH direct originate failed: %s", output)
	return parseFreeSWITCHOKUUID(h.t, output)
}

func (h *freeSWITCHHarness) hangupFreeSWITCHCall(channelUUID string) {
	h.t.Helper()
	h.hangupFreeSWITCHCallWithCause(channelUUID, "")
}

func (h *freeSWITCHHarness) hangupFreeSWITCHCallWithCause(channelUUID string, cause string) {
	h.t.Helper()
	if strings.TrimSpace(channelUUID) == "" {
		return
	}
	command := "uuid_kill " + channelUUID
	if strings.TrimSpace(cause) != "" {
		command += " " + strings.TrimSpace(cause)
	}
	output, err := h.runFreeSWITCHCommand(command)
	require.NoErrorf(h.t, err, "FreeSWITCH uuid_kill failed: %s", output)
}

func (h *freeSWITCHHarness) freeSWITCHCallExists(channelUUID string) bool {
	h.t.Helper()
	if strings.TrimSpace(channelUUID) == "" {
		return false
	}
	output, err := h.runFreeSWITCHCommand("uuid_exists " + channelUUID)
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(output), "true")
}

func (h *freeSWITCHHarness) close() {
	if h.server != nil {
		h.server.Stop()
	}
	if h.cancelFunc != nil {
		h.cancelFunc()
	}
}

func reserveUDPPort(t *testing.T, address string) int {
	t.Helper()
	conn, err := net.ListenPacket("udp", net.JoinHostPort(address, "0"))
	require.NoError(t, err)
	defer conn.Close()

	udpAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	require.True(t, ok)
	return udpAddr.Port
}

func requiredIntegrationEnv(t *testing.T, name string) string {
	t.Helper()
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		t.Skipf("%s is required for this FreeSWITCH SIP integration test", name)
	}
	return value
}

func integrationEnv(name string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func integrationEnvInt(t *testing.T, name string, fallback int) int {
	t.Helper()
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	require.NoErrorf(t, err, "%s must be an integer", name)
	return parsed
}

func freeSWITCHOutboundTargetDescription(config freeSWITCHIntegrationConfig, user string) string {
	return fmt.Sprintf("sip:%s@%s:%d", user, config.fsSIPHost, config.fsSIPPort)
}

func freeSWITCHUser(user string) string {
	return strings.TrimPrefix(strings.TrimSpace(user), "+")
}

func parseFreeSWITCHOKUUID(t *testing.T, output string) string {
	t.Helper()
	fields := strings.Fields(strings.TrimSpace(output))
	require.GreaterOrEqualf(t, len(fields), 2, "FreeSWITCH response did not include UUID: %s", output)
	require.Equalf(t, "+OK", fields[0], "unexpected FreeSWITCH response: %s", output)
	return fields[1]
}
