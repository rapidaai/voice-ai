package internal_telephony

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	lexatic_backend "github.com/rapidaai/protos"
)

type exotelTelephony struct {
	logger commons.Logger
	cfg    utils.Option

	AccountSid   string
	ClientID     string
	ClientSecret string
}

func NewExotelTelephony(logger commons.Logger, vaultCredential *lexatic_backend.VaultCredential, cfg utils.Option) (Telephony, error) {
	accountSid, ok := vaultCredential.GetValue().AsMap()["account_sid"]
	if !ok {
		return nil, fmt.Errorf("illegal vault config accountSid is not found")
	}
	clientId, ok := vaultCredential.GetValue().AsMap()["client_id"]
	if !ok {
		return nil, fmt.Errorf("illegal vault config client_id not found")
	}
	authToken, ok := vaultCredential.GetValue().AsMap()["client_secret"]
	if !ok {
		return nil, fmt.Errorf("illegal vault config")
	}
	return &exotelTelephony{
		logger:       logger,
		cfg:          cfg,
		AccountSid:   accountSid.(string),
		ClientID:     clientId.(string),
		ClientSecret: authToken.(string),
	}, nil
}

func (tpc *exotelTelephony) ClientUrl() string {
	return fmt.Sprintf("https://%s:%s@api.exotel.com/v1/Accounts/%s/Calls/connect.json",
		tpc.ClientID, tpc.ClientSecret, tpc.AccountSid)
}

func (tpc *exotelTelephony) CreateCall(
	auth types.SimplePrinciple,
	toPhone string,
	fromPhone string,
	assistantId, sessionId uint64) (map[string]interface{}, error) {
	redirectUrl := "assistant-01.rapida.ai"
	// redirectUrl = "integral-presently-cub.ngrok-free.app"
	formData := url.Values{}
	formData.Set("From", toPhone)
	formData.Set("CallerId", fromPhone)
	formData.Set("Url", fmt.Sprintf("wss://%s/v1/talk/exotel/stream/%d/%s/%d/%s",
		redirectUrl,
		assistantId,
		toPhone,
		sessionId,
		auth.GetCurrentToken()))

	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest("POST", tpc.ClientUrl(), strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var jsonResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&jsonResponse); err != nil {
		return nil, err
	}
	return jsonResponse, nil
}
