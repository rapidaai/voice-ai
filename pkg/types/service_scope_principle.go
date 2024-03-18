package types

/*
Service scope
*/
type ServiceScope struct {
	UserId         *uint64 `json:"userId"`
	ProjectId      *uint64 `json:"projectId"`
	OrganizationId *uint64 `json:"organizationId"`
}

func (ss *ServiceScope) GetUserId() *uint64 {
	return ss.UserId
}
func (ss *ServiceScope) GetCurrentProjectId() *uint64 {
	return ss.ProjectId
}
func (ss *ServiceScope) GetCurrentOrganizationId() *uint64 {
	return ss.OrganizationId
}

func (ss *ServiceScope) HasOrganization() bool {
	return ss.GetCurrentOrganizationId() != nil
}

func (ss *ServiceScope) HasUser() bool {
	return ss.GetUserId() != nil
}

func (ss *ServiceScope) HasProject() bool {
	return ss.GetCurrentProjectId() != nil
}

func (ss *ServiceScope) IsAuthenticated() bool {
	// org scope is already to have only org
	return ss.HasUser() && ss.HasOrganization()
}
