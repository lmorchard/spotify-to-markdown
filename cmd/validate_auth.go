package cmd

import (
	"fmt"

	"github.com/lmorchard/spotify-to-markdown/internal/database"
	"github.com/lmorchard/spotify-to-markdown/internal/spotify"
	"github.com/lmorchard/spotify-to-markdown/internal/spotifyauth"
	"github.com/spf13/cobra"
)

// validateAuthCmd answers "do my credentials work?" with a single-line
// stdout and a clean exit code. Suitable for orchestrators or scripts
// gating behavior on auth health.
var validateAuthCmd = &cobra.Command{
	Use:   "validate-auth",
	Short: "Check whether the cached Spotify OAuth token is accepted",
	Long: `Use the cached OAuth tokens (acquired via ` + "`auth`" + `) to make a
minimal authenticated request (GET /me) and exit 0 if the token is
accepted, non-zero otherwise.

If no token is cached, fails with a hint to run ` + "`auth`" + `.`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := GetConfig()
		if c.Spotify.ClientID == "" {
			return fmt.Errorf("client_id is not set; add it to spotify-to-markdown.yaml or set SPOTIFY_CLIENT_ID")
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

		client := spotify.New(auth, nil)
		user, err := client.CurrentUser(cmd.Context())
		if err != nil {
			return fmt.Errorf("spotify auth check: %w", err)
		}

		identity := user.DisplayName
		if identity == "" {
			identity = user.ID
		}
		fmt.Printf("validate-auth: ok (authenticated as %s)\n", identity)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(validateAuthCmd)
}
