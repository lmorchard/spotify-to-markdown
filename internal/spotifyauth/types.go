package spotifyauth

import "time"

// Token is the credential set we persist locally.
type Token struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	Scope        string
	ExpiresAt    time.Time
}

// IsExpired reports whether the access token is expired (or close enough that
// we should refresh before using it).
func (t Token) IsExpired() bool {
	// Refresh a minute early to avoid races against server-side clock skew.
	return time.Now().Add(60 * time.Second).After(t.ExpiresAt)
}

// tokenResponse mirrors the JSON body returned by Spotify's /api/token endpoint.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// errorResponse mirrors Spotify's auth-error JSON body.
type errorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}
