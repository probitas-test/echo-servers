package handlers

type AuthResponse struct {
	Authenticated bool   `json:"authenticated"`
	User          string `json:"user,omitempty"`
	Token         string `json:"token,omitempty"`
}
