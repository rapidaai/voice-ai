package internal_connects

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"github.com/lexatic/web-backend/config"
	internal_gorm "github.com/lexatic/web-backend/internal/gorm"
	"github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	gorm_types "github.com/lexatic/web-backend/pkg/models/gorm/types"
	"github.com/lexatic/web-backend/pkg/types"
	"golang.org/x/oauth2"
)

type ExternalConnect struct {
	cfg      *config.AppConfig
	log      commons.Logger
	postgres connectors.PostgresConnector
}

func NewExternalConnect(cfg *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector) ExternalConnect {
	return ExternalConnect{
		cfg:      cfg,
		log:      logger,
		postgres: postgres,
	}
}

// return id that will use as state
func (ec *ExternalConnect) EncodeState(ctx context.Context,
	toolId uint64,
	toolConnect string,
	linker gorm_types.VaultLevel, linkId uint64, redirect string) (string, error) {
	db := ec.postgres.DB(ctx)
	identifier := uuid.NewString()

	eConnect := &internal_gorm.OAuthExternalConnect{
		Identifier:  identifier,
		ToolConnect: toolConnect,
		ToolId:      toolId,
		Linker:      linker,
		LinkerId:    linkId,
		RedirectTo:  redirect,
	}
	tx := db.Save(eConnect)
	if err := tx.Error; err != nil {
		return identifier, err
	}
	return identifier, nil
}

func (ec *ExternalConnect) DecodeState(ctx context.Context, auth types.SimplePrinciple, identifier string) (*internal_gorm.OAuthExternalConnect, error) {
	db := ec.postgres.DB(ctx)
	var ct internal_gorm.OAuthExternalConnect
	tx := db.Last(&ct, "identifier = ?", identifier)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return &ct, nil
}

func (ec *ExternalConnect) ToToken(value map[string]interface{}) (*oauth2.Token, string, error) {
	// Extract access token

	connect, ok := value["connect"].(string)
	if !ok {
		return nil, connect, errors.New("connect not found or invalid type, how will identify what purpose your code will be used")
	}

	token := oauth2.Token{}
	accessToken, ok := value["accessToken"].(string)
	if !ok {
		return nil, connect, errors.New("access_token not found or invalid type")
	}
	token.AccessToken = accessToken
	// Extract token type
	tokenType, ok := value["tokenType"].(string)
	if !ok {
		return nil, connect, errors.New("token_type not found or invalid type")
	}
	token.TokenType = tokenType

	// Extract refresh token, if present
	refreshToken, ok := value["refreshToken"].(string)
	if !ok {
		return nil, connect, errors.New("refresh_token not found or invalid type")
	}
	token.RefreshToken = refreshToken

	if expiryStr, ok := value["expiry"].(string); ok {
		expiry, err := time.Parse(time.RFC3339, expiryStr)
		if err != nil {
			ec.log.Errorf("not able to parse expire time: %v time = %v", err, expiryStr)
			return nil, connect, fmt.Errorf("error parsing expiry time: %v", err)
		}
		token.Expiry = expiry
	}

	ec.log.Debugf("to_token returning the respo. s, keep in mind that expire is something need to be very precies %+v", token)
	// Construct and return the token
	return &token, connect, nil
}

func (ec *ExternalConnect) NewHttpClient() *resty.Client {
	return ec.GetClient(nil)
}

func (ec *ExternalConnect) GetClient(hc *http.Client) *resty.Client {
	ct := resty.New()
	if hc != nil {
		ct = resty.NewWithClient(hc)
	}

	if ec.cfg.IsDevelopment() {
		ct.SetDebug(true)
	}
	return ct
}
