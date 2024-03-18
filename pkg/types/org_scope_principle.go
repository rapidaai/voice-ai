package types

type OrganizationScope struct {
	OrganizationId *uint64 `json:"organization_id"`
	Status         string  `json:"status"`
}

func (ss *OrganizationScope) GetUserId() *uint64 {
	// hard coding this
	return nil
}
func (ss *OrganizationScope) GetCurrentProjectId() *uint64 {
	return nil
}
func (ss *OrganizationScope) GetCurrentOrganizationId() *uint64 {
	return ss.OrganizationId
}

func (ss *OrganizationScope) HasOrganization() bool {
	return ss.GetCurrentOrganizationId() != nil
}

func (ss *OrganizationScope) HasUser() bool {
	return ss.GetUserId() != nil
}

func (ss *OrganizationScope) HasProject() bool {
	return ss.GetCurrentProjectId() != nil
}

func (ss *OrganizationScope) IsActive() bool {
	return ss.Status == "active"
}

func (ss *OrganizationScope) IsAuthenticated() bool {
	// org scope is already to have only org
	return ss.HasOrganization() && ss.IsActive()
}
