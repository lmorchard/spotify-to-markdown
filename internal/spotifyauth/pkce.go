package spotifyauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// generateCodeVerifier returns a cryptographically random PKCE code verifier
// of the given byte length (raw entropy; base64url-encoded length will be larger).
// Spotify accepts 43-128 chars of [A-Za-z0-9-._~]; base64url encoding satisfies that.
func generateCodeVerifier(numBytes int) (string, error) {
	buf := make([]byte, numBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// codeChallengeS256 returns the S256-method PKCE challenge for the given verifier.
func codeChallengeS256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// randomState returns a short, URL-safe random string for the OAuth state param.
func randomState() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
