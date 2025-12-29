// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package gorm_types

import (
	"database/sql/driver"
	"encoding/json"
)

type RetrievalMethod string

const (
	RETRIEVAL_METHOD_SEMANTIC      RetrievalMethod = "semantic-search"
	RETRIEVAL_METHOD_FULLTEXT      RetrievalMethod = "full-text-search"
	RETRIEVAL_METHOD_HYBRID        RetrievalMethod = "hybrid-search"
	RETRIEVAL_METHOD_INVERTEDINDEX RetrievalMethod = "inverted-index"
)

func (c RetrievalMethod) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(c))
}

func (c RetrievalMethod) Value() (driver.Value, error) {
	return string(c), nil
}

type DocumentSource string
type ManualDocumentSource string

const (
	DOCUMENT_SOURCE_MANUAL DocumentSource = "manual"
	// all manual upload thigns
	DOCUMENT_SOURCE_MANUAL_FILE ManualDocumentSource = "manual-file"
	DOCUMENT_SOURCE_MANUAL_ZIP  ManualDocumentSource = "manual-zip"
	DOCUMENT_SOURCE_MANUAL_URL  ManualDocumentSource = "manual-url"
	//
	// DOCUMENT_SOURCE_GITHUB               DocumentSource = "github-code"
	// DOCUMENT_SOURCE_GOOGLE_DRIVE         DocumentSource = "google-drive"
	// DOCUMENT_SOURCE_MICROSOFT_SHAREPOINT DocumentSource = "microsoft-sharepoint"
	// DOCUMENT_SOURCE_CONFLUENCE           DocumentSource = "atlasian-confluence"
	// DOCUMENT_SOURCE_NOTION               DocumentSource = "notion"
)

type DocumentType string

const (
	DOCUMENT_TYPE_PDF     = "pdf"
	DOCUMENT_TYPE_CSV     = "pdf"
	DOCUMENT_TYPE_XLS     = "pdf"
	DOCUMENT_TYPE_TXT     = "txt"
	DOCUMENT_TYPE_WEB_URL = "web-url"
	DOCUMENT_TYPE_UNKNOWN = "unknown"
	DOCUMENT_TYPE_CODE    = "source-code"
)
