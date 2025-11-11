/*
 *  Copyright (c) 2024. Rapida
 *
 *  Permission is hereby granted, free of charge, to any person obtaining a copy
 *  of this software and associated documentation files (the "Software"), to deal
 *  in the Software without restriction, including without limitation the rights
 *  to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 *  copies of the Software, and to permit persons to whom the Software is
 *  furnished to do so, subject to the following conditions:
 *
 *  The above copyright notice and this permission notice shall be included in
 *  all copies or substantial portions of the Software.
 *
 *  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 *  IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 *  FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 *  AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 *  LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 *  OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 *  THE SOFTWARE.
 *
 *  Author: Prashant <prashant@rapida.ai>
 *
 */
package types

/*
Service scope
*/
type ServiceScope struct {
	UserId         *uint64 `json:"userId"`
	ProjectId      *uint64 `json:"projectId"`
	OrganizationId *uint64 `json:"organizationId"`
	CurrentToken   string  `json:"currentToken"`
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
	return (ss.HasUser() || ss.HasProject()) && ss.HasOrganization()
}

func (ss *ServiceScope) GetCurrentToken() string {
	return ss.CurrentToken
}

func (ss *ServiceScope) Type() string {
	return "service"
}
