package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// exportCmd is the orchestrator-facing entry point with the canonical
// `--since/--until/-o` flag shape shared across all *-to-markdown tools.
// It composes `fetch` (pulls fresh plays into the local archive) and a
// windowed `render` over the requested window.
//
// Output destination handling: --output is used to override the
// `output.file` config key for this invocation. An empty `-o` is treated
// the same as `-o -` (write to stdout) so the orchestrator can read the
// markdown from the subprocess's stdout.
var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Orchestrator-friendly fetch + render in one shot",
	Long: `Pull recent Spotify plays into the local archive and render markdown
over the given time window in one invocation.

The --since/--until flag shape matches the contract used by me-to-markdown
and the rest of the *-to-markdown tools. Template settings come from the
config file or environment; this subcommand exposes only the
orchestrator-facing flags.

Example usage:
  spotify-to-markdown export --since 168h
  spotify-to-markdown export --since 2026-05-11 --until 2026-05-18 -o spotify.md`,
	RunE: func(cmd *cobra.Command, args []string) error {
		since, _ := cmd.Flags().GetString("since")
		until, _ := cmd.Flags().GetString("until")
		output, _ := cmd.Flags().GetString("output")

		if output == "" {
			output = "-"
		}

		// Override viper keys read by render. fetch ignores these.
		viper.Set("output.file", output)

		// Wire --since/--until into render via its own flags so that
		// render's flag-reading path sees them. Cobra's flag values
		// persist across RunE calls because both invocations share this
		// process-local cobra.Command tree.
		if err := renderCmd.Flags().Set("since", since); err != nil {
			return fmt.Errorf("set render --since: %w", err)
		}
		if err := renderCmd.Flags().Set("until", until); err != nil {
			return fmt.Errorf("set render --until: %w", err)
		}

		if err := fetchCmd.RunE(cmd, args); err != nil {
			return err
		}
		return renderCmd.RunE(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)

	exportCmd.Flags().String("since", "", "Start of time window (YYYY-MM-DD or Go duration like 168h) — required")
	exportCmd.Flags().String("until", "", "End of time window (YYYY-MM-DD, defaults to now)")
	exportCmd.Flags().StringP("output", "o", "", "Output file (default: stdout)")
	_ = exportCmd.MarkFlagRequired("since")
}
