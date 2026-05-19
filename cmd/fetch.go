package cmd

import (
	"fmt"

	"github.com/lmorchard/spotify-to-markdown/internal/database"
	"github.com/lmorchard/spotify-to-markdown/internal/spotify"
	"github.com/lmorchard/spotify-to-markdown/internal/spotifyauth"
	"github.com/lmorchard/spotify-to-markdown/internal/store"
	"github.com/spf13/cobra"
)

var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch recent Spotify activity into the local database",
	Long: `Calls Spotify's Web API for your recent activity and upserts the
results into the local SQLite database. Safe to re-run; duplicate plays are
ignored on insert (played_at is the natural primary key).

Currently fetches: recently-played tracks (last 50).
Future phases will also fetch: currently-playing, top tracks/artists, saved
tracks.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log := GetLogger()
		c := GetConfig()

		if c.Spotify.ClientID == "" {
			return fmt.Errorf("spotify.client_id is not set; add it to %s", "spotify-to-markdown.yaml")
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
		st := store.New(db.Conn())

		ctx := cmd.Context()
		log.Info("Fetching recently-played tracks from Spotify…")
		resp, err := client.GetRecentlyPlayed(ctx, 50, 0)
		if err != nil {
			return fmt.Errorf("get recently played: %w", err)
		}

		var seen, newCount int
		for _, item := range resp.Items {
			seen++
			inserted, err := st.SavePlay(item)
			if err != nil {
				return fmt.Errorf("save play %s: %w", item.PlayedAt, err)
			}
			if inserted {
				newCount++
			}
		}

		log.Infof("Fetched %d plays (%d new, %d already known).", seen, newCount, seen-newCount)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(fetchCmd)
}
