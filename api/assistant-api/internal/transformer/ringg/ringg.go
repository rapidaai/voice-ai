// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_ringg

import (
	"fmt"
	"strings"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

const (
	RINGG_STT_URL          = "wss://prod-api.ringg.ai/stt/v1/stream"
	RINGG_DEFAULT_LANGUAGE = "en"
)

type ringgOption struct {
	key     string
	logger  commons.Logger
	mdlOpts utils.Option
}

func NewRinggOption(
	logger commons.Logger,
	vaultCredential *protos.VaultCredential,
	opts utils.Option,
) (*ringgOption, error) {
	if vaultCredential == nil || vaultCredential.GetValue() == nil {
		return nil, fmt.Errorf("ringg: illegal vault config - missing key")
	}

	vaultMap := vaultCredential.GetValue().AsMap()
	rawKey, ok := vaultMap["key"]
	if !ok {
		return nil, fmt.Errorf("ringg: illegal vault config - missing key")
	}

	key, ok := rawKey.(string)
	if !ok || strings.TrimSpace(key) == "" {
		return nil, fmt.Errorf("ringg: illegal vault config - invalid key")
	}

	return &ringgOption{
		key:     key,
		mdlOpts: opts,
		logger:  logger,
	}, nil
}

func (co *ringgOption) GetKey() string {
	return co.key
}

func (co *ringgOption) GetLanguage() string {
	if lang, err := co.mdlOpts.GetString("listen.language"); err == nil && strings.TrimSpace(lang) != "" {
		return lang
	}
	return RINGG_DEFAULT_LANGUAGE
}
