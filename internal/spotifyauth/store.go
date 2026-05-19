package spotifyauth

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ErrNoToken indicates that no token has ever been stored (the user must run `auth`).
var ErrNoToken = errors.New("no spotify token stored; run `spotify-to-markdown auth` first")

// LoadToken reads the currently-persisted token from the auth_tokens table.
// Returns ErrNoToken if no row exists.
func LoadToken(db *sql.DB) (*Token, error) {
	var (
		t         Token
		expiresAt string
	)
	err := db.QueryRow(`
		SELECT access_token, refresh_token, token_type, scope, expires_at
		FROM auth_tokens
		WHERE id = 1
	`).Scan(&t.AccessToken, &t.RefreshToken, &t.TokenType, &t.Scope, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNoToken
	}
	if err != nil {
		return nil, fmt.Errorf("query auth_tokens: %w", err)
	}

	parsed, err := parseStoredTime(expiresAt)
	if err != nil {
		return nil, fmt.Errorf("parse expires_at %q: %w", expiresAt, err)
	}
	t.ExpiresAt = parsed
	return &t, nil
}

// SaveToken upserts the token into the auth_tokens table.
func SaveToken(db *sql.DB, t *Token) error {
	_, err := db.Exec(`
		INSERT INTO auth_tokens (id, access_token, refresh_token, token_type, scope, expires_at, updated_at)
		VALUES (1, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			access_token  = excluded.access_token,
			refresh_token = excluded.refresh_token,
			token_type    = excluded.token_type,
			scope         = excluded.scope,
			expires_at    = excluded.expires_at,
			updated_at    = CURRENT_TIMESTAMP
	`, t.AccessToken, t.RefreshToken, t.TokenType, t.Scope, t.ExpiresAt.UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("upsert auth_tokens: %w", err)
	}
	return nil
}

// parseStoredTime accepts RFC3339 (what we write) and SQLite's default
// "YYYY-MM-DD HH:MM:SS" format (in case of any drift).
func parseStoredTime(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("unrecognized timestamp format")
}
