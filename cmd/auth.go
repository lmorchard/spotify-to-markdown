package cmd

import (
	"fmt"

	"github.com/lmorchard/spotify-to-markdown/internal/database"
	"github.com/lmorchard/spotify-to-markdown/internal/spotifyauth"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authorize with Spotify (one-time interactive OAuth flow)",
	Long: `Runs the Spotify OAuth2 Authorization Code (PKCE) flow:

  1. Opens your default browser pointed at Spotify's consent screen.
  2. Listens for the callback on 127.0.0.1 (port from config).
  3. Exchanges the authorization code for access + refresh tokens.
  4. Persists those tokens in the local SQLite database.

After this completes once, ` + "`fetch`" + ` and ` + "`render`" + ` can run unattended; the
refresh token is used silently to mint fresh access tokens as needed.

You must have a Spotify Developer app registered with the redirect URI
http://127.0.0.1:<port>/callback added to its Redirect URIs list, and the
matching client_id set in your config file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log := GetLogger()
		c := GetConfig()

		if c.Spotify.ClientID == "" {
			return fmt.Errorf("client_id is not set; add it to %s or set SPOTIFY_CLIENT_ID", "spotify-to-markdown.yaml")
		}

		db, err := database.New(c.Database)
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer func() { _ = db.Close() }()

		auth := spotifyauth.New(spotifyauth.Config{
			ClientID:     c.Spotify.ClientID,
			RedirectPort: c.Spotify.RedirectPort,
			Scopes:       c.Spotify.Scopes,
		}, db.Conn(), nil)

		log.Infof("Starting auth flow; redirect URI is %s", auth.RedirectURI())
		log.Infof("Make sure this URI is registered in your Spotify app at https://developer.spotify.com/dashboard")

		if err := auth.RunInteractiveFlow(cmd.Context()); err != nil {
			return fmt.Errorf("auth flow failed: %w", err)
		}

		fmt.Println("✅ Authorization complete; tokens persisted to", c.Database)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
}
