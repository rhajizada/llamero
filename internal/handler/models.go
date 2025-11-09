package handler

// tokenResponse represents the provider token exchange response.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	IDToken      string `json:"id_token"`
}

// userInfo carries normalized claims extracted from the provider userinfo payload.
type userInfo struct {
	Subject string
	Email   string
	Name    string
	Groups  []string
}
