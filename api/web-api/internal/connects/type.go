package internal_connects

type OpenID struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Verified bool   `json:"verified_email"`
	Id       string `json:"id"`
	Source   string `json:"source"`
	Token    string `json:"token"`
}
