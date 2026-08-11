package internal_sip_telephony

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rapidaai/api/assistant-api/config"
	internal_sip "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/sip/internal"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/structpb"
)

func vaultCredential(t *testing.T, values map[string]interface{}) *protos.VaultCredential {
	t.Helper()
	v, err := structpb.NewStruct(values)
	if err != nil {
		t.Fatalf("failed to create vault credential: %v", err)
	}
	return &protos.VaultCredential{Value: v}
}

func newSIPTelephonyForTest() *sipTelephony {
	logger, _ := commons.NewApplicationLogger()
	return &sipTelephony{
		logger: logger,
		appCfg: &config.AssistantConfig{
			SIPConfig: &config.SIPConfig{
				Port:                5060,
				Transport:           "udp",
				RTPPortRangeStart:   10000,
				RTPPortRangeEnd:     10100,
				RegisterTimeout:     5 * time.Second,
				InviteTimeout:       30 * time.Second,
				SessionTimeout:      45 * time.Minute,
				MediaTimeoutInitial: 20 * time.Second,
				MediaTimeout:        10 * time.Second,
			},
		},
	}
}

func TestStatusCallback_NormalizesCompletedJSONPayload(t *testing.T) {
	telephony := newSIPTelephonyForTest()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"event":"completed","call_id":"sip-call-1","duration_ms":1234,"price":"0.01"}`
	req := httptest.NewRequest("POST", "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	info, err := telephony.StatusCallback(c, nil, 42, 99)
	if err != nil {
		t.Fatalf("StatusCallback() error = %v", err)
	}

	if info.Event != internal_type.TelephonyEventCompleted {
		t.Fatalf("expected completed event, got %q", info.Event)
	}
	if !info.Completed {
		t.Fatal("expected completed callback")
	}
	if info.Error != nil {
		t.Fatalf("expected no error, got %+v", info.Error)
	}
	if info.ChannelUUID != "sip-call-1" {
		t.Fatalf("expected ChannelUUID sip-call-1, got %q", info.ChannelUUID)
	}
	if info.Duration == nil || info.Duration.Milliseconds() != 1234 {
		t.Fatalf("expected duration 1234ms, got %v", info.Duration)
	}
	if info.Price != "0.01" {
		t.Fatalf("expected price 0.01, got %q", info.Price)
	}
	if info.RawPayload != body {
		t.Fatalf("expected raw payload %q, got %q", body, info.RawPayload)
	}
}

func TestStatusCallback_NormalizesFailedFormPayload(t *testing.T) {
	telephony := newSIPTelephonyForTest()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := "event=failed&call_id=sip-call-2&reason=no_answer"
	req := httptest.NewRequest("POST", "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.Request = req

	info, err := telephony.StatusCallback(c, nil, 42, 99)
	if err != nil {
		t.Fatalf("StatusCallback() error = %v", err)
	}

	if info.Event != internal_type.TelephonyEventCompleted {
		t.Fatalf("expected failed event, got %q", info.Event)
	}
	if info.Error == nil {
		t.Fatal("expected failed status error")
	}
	if info.Error.Reason != "no_answer" {
		t.Fatalf("expected failure reason no_answer, got %q", info.Error.Reason)
	}
	if info.ChannelUUID != "sip-call-2" {
		t.Fatalf("expected ChannelUUID sip-call-2, got %q", info.ChannelUUID)
	}
}

func TestStatusCallback_NormalizesCancelledQueryPayload(t *testing.T) {
	telephony := newSIPTelephonyForTest()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/?event=cancelled&call_id=sip-call-3", nil)
	c.Request = req

	info, err := telephony.StatusCallback(c, nil, 42, 99)
	if err != nil {
		t.Fatalf("StatusCallback() error = %v", err)
	}

	if info.Event != internal_type.TelephonyEvent("cancelled") {
		t.Fatalf("expected cancelled event, got %q", info.Event)
	}
	if info.Completed {
		t.Fatal("cancelled callback must not be completed")
	}
	if info.Error != nil {
		t.Fatalf("cancelled callback should be reported as cancelled, not failed: %+v", info.Error)
	}
	if info.ChannelUUID != "sip-call-3" {
		t.Fatalf("expected ChannelUUID sip-call-3, got %q", info.ChannelUUID)
	}
}

func TestParseConfig_UsesPortFromSIPURI(t *testing.T) {
	telephony := newSIPTelephonyForTest()
	cred := vaultCredential(t, map[string]interface{}{
		"sip_uri":      "sip:example.org:5097",
		"sip_username": "user",
		"sip_password": "pass",
	})

	cfg, err := telephony.parseConfig(cred)
	if err != nil {
		t.Fatalf("parseConfig() error = %v", err)
	}

	if cfg.Port != 5097 {
		t.Fatalf("expected parsed SIP URI port 5097, got %d", cfg.Port)
	}
}

func TestParseConfig_UsesExplicitSIPPortFromVault(t *testing.T) {
	telephony := newSIPTelephonyForTest()
	cred := vaultCredential(t, map[string]interface{}{
		"sip_server":   "example.org",
		"sip_port":     5098,
		"sip_username": "user",
		"sip_password": "pass",
	})

	cfg, err := telephony.parseConfig(cred)
	if err != nil {
		t.Fatalf("parseConfig() error = %v", err)
	}

	if cfg.Port != 5098 {
		t.Fatalf("expected explicit vault sip_port 5098, got %d", cfg.Port)
	}
}

func TestParseConfig_DefaultsOutboundTo5060WhenVaultPortMissing(t *testing.T) {
	telephony := newSIPTelephonyForTest()
	cred := vaultCredential(t, map[string]interface{}{
		"sip_server":   "example.org",
		"sip_username": "user",
		"sip_password": "pass",
	})

	cfg, err := telephony.parseConfig(cred)
	if err != nil {
		t.Fatalf("parseConfig() error = %v", err)
	}

	if cfg.Port != internal_sip.DefaultOutboundSIPPort {
		t.Fatalf("expected default outbound SIP port %d, got %d", internal_sip.DefaultOutboundSIPPort, cfg.Port)
	}
}

func TestParseConfig_AllowsOutboundWithoutAuth(t *testing.T) {
	telephony := newSIPTelephonyForTest()
	cred := vaultCredential(t, map[string]interface{}{
		"host": "example.org:5060",
	})

	cfg, err := telephony.parseConfig(cred)
	if err != nil {
		t.Fatalf("parseConfig() error = %v", err)
	}

	if cfg.Server != "example.org" {
		t.Fatalf("expected server example.org, got %q", cfg.Server)
	}
	if cfg.Username != "" || cfg.Password != "" {
		t.Fatalf("expected empty auth, got username=%q password=%q", cfg.Username, cfg.Password)
	}
}

func TestParseConfig_AppliesPlatformTimeouts(t *testing.T) {
	telephony := newSIPTelephonyForTest()
	cred := vaultCredential(t, map[string]interface{}{
		"host": "example.org:5060",
	})

	cfg, err := telephony.parseConfig(cred)
	if err != nil {
		t.Fatalf("parseConfig() error = %v", err)
	}

	if cfg.RegisterTimeout != 5*time.Second {
		t.Fatalf("expected register timeout 5s, got %s", cfg.RegisterTimeout)
	}
	if cfg.InviteTimeout != 30*time.Second {
		t.Fatalf("expected invite timeout 30s, got %s", cfg.InviteTimeout)
	}
	if cfg.SessionTimeout != 45*time.Minute {
		t.Fatalf("expected session timeout 45m, got %s", cfg.SessionTimeout)
	}
	if cfg.MediaTimeoutInitial != 20*time.Second {
		t.Fatalf("expected initial media timeout 20s, got %s", cfg.MediaTimeoutInitial)
	}
	if cfg.MediaTimeout != 10*time.Second {
		t.Fatalf("expected media timeout 10s, got %s", cfg.MediaTimeout)
	}
}

func TestParseConfig_AppliesInboundAnswerPolicyDefaults(t *testing.T) {
	telephony := newSIPTelephonyForTest()
	telephony.appCfg.SIPConfig.Inbound = config.SIPInboundConfig{
		AnswerMode:      string(sip_infra.InboundAnswerModeAfterMinRingDuration),
		MinRingDuration: 50 * time.Millisecond,
		MaxRingDuration: 5 * time.Second,
		ACKTimeout:      2 * time.Second,
	}
	cred := vaultCredential(t, map[string]interface{}{
		"host": "example.org:5060",
	})

	cfg, err := telephony.parseConfig(cred)
	if err != nil {
		t.Fatalf("parseConfig() error = %v", err)
	}

	if cfg.InboundAnswerMode != sip_infra.InboundAnswerModeAfterMinRingDuration {
		t.Fatalf("expected inbound answer mode from app config, got %q", cfg.InboundAnswerMode)
	}
	if cfg.InboundMinRingDuration != 50*time.Millisecond ||
		cfg.InboundMaxRingDuration != 5*time.Second ||
		cfg.InboundACKTimeout != 2*time.Second {
		t.Fatalf("expected inbound answer policy defaults from app config, got %#v", cfg)
	}
}

func TestParseConfig_ParsesCustomHeaders(t *testing.T) {
	telephony := newSIPTelephonyForTest()
	cred := vaultCredential(t, map[string]interface{}{
		"sip_uri":      "sip:example.org:5060",
		"sip_username": "user",
		"sip_password": "pass",
		"sip_headers":  `{"X-Piopiy-Username":"Nitin","X-Custom":"value"}`,
	})

	cfg, err := telephony.parseConfig(cred)
	if err != nil {
		t.Fatalf("parseConfig() error = %v", err)
	}

	if len(cfg.CustomHeaders) != 2 {
		t.Fatalf("expected 2 custom headers, got %d", len(cfg.CustomHeaders))
	}
	if cfg.CustomHeaders["X-Piopiy-Username"] != "Nitin" {
		t.Fatalf("expected X-Piopiy-Username=Nitin, got %s", cfg.CustomHeaders["X-Piopiy-Username"])
	}
	if cfg.CustomHeaders["X-Custom"] != "value" {
		t.Fatalf("expected X-Custom=value, got %s", cfg.CustomHeaders["X-Custom"])
	}
}

func TestParseConfig_NoCustomHeadersWhenMissing(t *testing.T) {
	telephony := newSIPTelephonyForTest()
	cred := vaultCredential(t, map[string]interface{}{
		"sip_uri":      "sip:example.org:5060",
		"sip_username": "user",
		"sip_password": "pass",
	})

	cfg, err := telephony.parseConfig(cred)
	if err != nil {
		t.Fatalf("parseConfig() error = %v", err)
	}

	if cfg.CustomHeaders != nil {
		t.Fatalf("expected nil custom headers, got %v", cfg.CustomHeaders)
	}
}

func TestParseConfig_InvalidJSONHeadersIgnored(t *testing.T) {
	telephony := newSIPTelephonyForTest()
	cred := vaultCredential(t, map[string]interface{}{
		"sip_uri":      "sip:example.org:5060",
		"sip_username": "user",
		"sip_password": "pass",
		"sip_headers":  "not-json",
	})

	cfg, err := telephony.parseConfig(cred)
	if err != nil {
		t.Fatalf("parseConfig() error = %v", err)
	}

	if cfg.CustomHeaders != nil {
		t.Fatalf("expected nil custom headers for invalid JSON, got %v", cfg.CustomHeaders)
	}
}

func TestReceiveCall_PopulatesDialedNumberFromFallbackParams(t *testing.T) {
	telephony := newSIPTelephonyForTest()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/?caller=15551234567&destination=18005550100&call_id=sip-call-1", nil)
	c.Request = req

	info, err := telephony.ReceiveCall(c)
	if err != nil {
		t.Fatalf("ReceiveCall() error = %v", err)
	}

	if info.CallerNumber != "15551234567" {
		t.Fatalf("expected CallerNumber 15551234567, got %q", info.CallerNumber)
	}
	if info.FromNumber != "18005550100" {
		t.Fatalf("expected FromNumber from destination fallback, got %q", info.FromNumber)
	}
	if info.ChannelUUID != "sip-call-1" {
		t.Fatalf("expected ChannelUUID sip-call-1, got %q", info.ChannelUUID)
	}
	payload, ok := info.StatusInfo.Payload.(map[string]string)
	if !ok {
		t.Fatalf("expected map[string]string payload, got %T", info.StatusInfo.Payload)
	}
	if got := payload["destination"]; got != "18005550100" {
		t.Fatalf("expected status payload destination=18005550100, got %q", got)
	}
}
