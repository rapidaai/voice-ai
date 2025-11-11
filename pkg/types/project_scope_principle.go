package types

import type_enums "github.com/rapidaai/pkg/types/enums"

type ProjectScope struct {
	ProjectId      *uint64 `json:"projectId"`
	OrganizationId *uint64 `json:"organizationId"`
	Status         string  `json:"status"`
	CurrentToken   string  `json:"currentToken"`
}

func (ss *ProjectScope) GetUserId() *uint64 {
	return nil
}
func (ss *ProjectScope) GetCurrentProjectId() *uint64 {
	return ss.ProjectId
}
func (ss *ProjectScope) GetCurrentOrganizationId() *uint64 {
	return ss.OrganizationId
}

func (ss *ProjectScope) HasOrganization() bool {
	return ss.GetCurrentOrganizationId() != nil
}

func (ss *ProjectScope) HasUser() bool {
	return ss.GetUserId() != nil
}

func (ss *ProjectScope) HasProject() bool {
	return ss.GetCurrentProjectId() != nil
}

func (ss *ProjectScope) IsActive() bool {
	return ss.Status == type_enums.RECORD_ACTIVE.String()
}

func (ss *ProjectScope) IsAuthenticated() bool {
	// org scope is already to have only org
	return ss.HasProject() && ss.IsActive() && ss.HasOrganization()
}

func (ss *ProjectScope) GetCurrentToken() string {
	return ss.CurrentToken
}

func (aP *ProjectScope) Type() string {
	return "project"
}
