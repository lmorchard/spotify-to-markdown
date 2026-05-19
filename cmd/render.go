package cmd

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"github.com/lmorchard/spotify-to-markdown/internal/database"
	"github.com/lmorchard/spotify-to-markdown/internal/store"
	"github.com/lmorchard/spotify-to-markdown/internal/templates"
	"github.com/spf13/cobra"
)

var renderCmd = &cobra.Command{
	Use:   "render",
	Short: "Render the markdown summary from the local database",
	Long: `Reads recent plays from the local SQLite database and writes a
markdown summary to the configured output file.

Uses the embedded default template unless ` + "`output.template`" + ` points to a
custom template file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log := GetLogger()
		c := GetConfig()

		db, err := database.New(c.Database)
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer func() { _ = db.Close() }()

		st := store.New(db.Conn())

		plays, err := st.RecentPlays(50)
		if err != nil {
			return fmt.Errorf("load recent plays: %w", err)
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
		if err := os.WriteFile(outPath, buf.Bytes(), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", outPath, err)
		}

		log.Infof("Wrote %s (%d plays)", outPath, len(plays))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(renderCmd)
}
