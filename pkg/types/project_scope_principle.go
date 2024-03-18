package types

type ProjectScope struct {
	ProjectId      *uint64 `json:"project_id"`
	OrganizationId *uint64 `json:"organization_id"`
	Status         string  `json:"status"`
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
	return ss.Status == "active"
}

func (ss *ProjectScope) IsAuthenticated() bool {
	// org scope is already to have only org
	return ss.HasProject() && ss.IsActive() && ss.HasOrganization()
}
