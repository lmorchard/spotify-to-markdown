package cmd

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"github.com/lmorchard/spotify-to-markdown/internal/database"
	"github.com/lmorchard/spotify-to-markdown/internal/store"
	"github.com/lmorchard/spotify-to-markdown/internal/templates"
	"github.com/lmorchard/spotify-to-markdown/internal/timewindow"
	"github.com/spf13/cobra"
)

var renderCmd = &cobra.Command{
	Use:   "render",
	Short: "Render the markdown summary from the local database",
	Long: `Reads plays from the local SQLite database and writes a markdown
summary to the configured output file.

By default (no --since/--until), renders the most recent 50 plays. When
--since is set, renders all plays in the [--since, --until] window
instead.

--since accepts a Go duration (e.g. 168h), a date (YYYY-MM-DD), or an
RFC3339 timestamp. --until accepts a date (treated as end-of-day,
inclusive) or RFC3339, and defaults to now when --since is given without
it.

Uses the embedded default template unless ` + "`output.template`" + ` points to a
custom template file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log := GetLogger()
		c := GetConfig()

		sinceStr, _ := cmd.Flags().GetString("since")
		untilStr, _ := cmd.Flags().GetString("until")

		db, err := database.New(c.Database)
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer func() { _ = db.Close() }()

		st := store.New(db.Conn())

		var plays []store.PlayView
		if sinceStr != "" {
			now := time.Now()
			since, err := timewindow.Parse(sinceStr, now, false)
			if err != nil {
				return fmt.Errorf("--since: %w", err)
			}
			until := now
			if untilStr != "" {
				until, err = timewindow.Parse(untilStr, now, true)
				if err != nil {
					return fmt.Errorf("--until: %w", err)
				}
			}
			plays, err = st.PlaysBetween(since, until)
			if err != nil {
				return fmt.Errorf("load plays between: %w", err)
			}
			log.Infof("Loaded %d plays between %s and %s", len(plays),
				since.Format(time.RFC3339), until.Format(time.RFC3339))
		} else {
			plays, err = st.RecentPlays(50)
			if err != nil {
				return fmt.Errorf("load recent plays: %w", err)
			}
		}

		var renderer *templates.Renderer
		if c.Output.Template != "" {
			renderer, err = templates.NewRendererFromFile(c.Output.Template)
		} else {
			renderer, err = templates.NewRenderer()
		}
		if err != nil {
			return fmt.Errorf("init template: %w", err)
		}

		var buf bytes.Buffer
		if err := renderer.Render(&buf, templates.RenderData{
			Generated: time.Now(),
			Plays:     plays,
		}); err != nil {
			return fmt.Errorf("render: %w", err)
		}

		outPath := c.Output.File
		if outPath == "" {
			outPath = "spotify-recent.md"
		}
		// "-" means stdout
		if outPath == "-" {
			if _, err := os.Stdout.Write(buf.Bytes()); err != nil {
				return fmt.Errorf("write stdout: %w", err)
			}
			return nil
		}
		if err := os.WriteFile(outPath, buf.Bytes(), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", outPath, err)
		}
		log.Infof("Wrote %s (%d plays)", outPath, len(plays))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(renderCmd)

	renderCmd.Flags().String("since", "", "Optional start of time window (Go duration like 168h, YYYY-MM-DD, or RFC3339). When unset, render the most recent 50 plays.")
	renderCmd.Flags().String("until", "", "Optional end of time window (YYYY-MM-DD or RFC3339, defaults to now)")
}
