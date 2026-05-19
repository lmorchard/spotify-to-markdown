package cmd

import (
	"fmt"
	"os"

	"github.com/lmorchard/spotify-to-markdown/internal/templates"
	"github.com/spf13/cobra"
)

const defaultConfigContent = `# Configuration file for spotify-to-markdown

# Path to the SQLite database that stores fetched plays and OAuth tokens.
# Default: $XDG_STATE_HOME/spotify-to-markdown/state.db
# database: "/custom/path/state.db"

# Logging
verbose: false
debug: false
log_json: false

spotify:
  # Required. The client_id from your Spotify Developer app:
  # https://developer.spotify.com/dashboard
  # No client_secret is needed — this tool uses the PKCE flow.
  client_id: ""

  # Port for the local OAuth callback listener. Must match the redirect URI
  # registered on your Spotify app, which should be:
  #     http://127.0.0.1:<redirect_port>/callback
  redirect_port: 8888

  # OAuth scopes to request. The defaults cover all phases of fetching, so a
  # one-time ` + "`auth`" + ` grants permission for every supported endpoint. Only
  # override this if you want to narrow scope.
  # scopes:
  #   - user-read-recently-played
  #   - user-read-currently-playing
  #   - user-top-read
  #   - user-library-read

output:
  # Path to the rendered markdown file. Overwritten on every render / run.
  file: "spotify-recent.md"

  # Optional path to a custom Go text/template file. When empty, the embedded
  # default template is used. ` + "`spotify-to-markdown init`" + ` writes a copy of
  # that default alongside this config; point this at it to customize.
  # template: "spotify-to-markdown.md"
`

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration and template files",
	Long: `Create default configuration file and custom template file for customization.

This command generates:
  - spotify-to-markdown.yaml (configuration file)
  - spotify-to-markdown.md (customizable template, or use --template-file to specify)

Use --force to overwrite existing files.

Example:
  spotify-to-markdown init
  spotify-to-markdown init --template-file my-template.md
  spotify-to-markdown init --force`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log := GetLogger()
		force, _ := cmd.Flags().GetBool("force")
		templateFile, _ := cmd.Flags().GetString("template-file")

		configFile := "spotify-to-markdown.yaml"

		// Check if config file exists
		configExists := fileExists(configFile)
		if configExists && !force {
			return fmt.Errorf("config file %s already exists (use --force to overwrite)", configFile)
		}

		// Check if template file exists
		templateExists := fileExists(templateFile)
		if templateExists && !force {
			return fmt.Errorf("template file %s already exists (use --force to overwrite)", templateFile)
		}

		// Create config file
		if err := os.WriteFile(configFile, []byte(defaultConfigContent), 0o644); err != nil {
			return fmt.Errorf("failed to create config file: %w", err)
		}

		if configExists {
			log.Infof("Overwrote %s", configFile)
		} else {
			log.Infof("Created %s", configFile)
		}

		// Get default template content
		templateContent, err := templates.GetDefaultTemplate()
		if err != nil {
			return fmt.Errorf("failed to get default template: %w", err)
		}

		// Create template file
		if err := os.WriteFile(templateFile, []byte(templateContent), 0o644); err != nil {
			return fmt.Errorf("failed to create template file: %w", err)
		}

		if templateExists {
			log.Infof("Overwrote %s", templateFile)
		} else {
			log.Infof("Created %s", templateFile)
		}

		fmt.Printf("\n✅ Initialization complete!\n\n")
		fmt.Printf("Next steps:\n")
		fmt.Printf("  1. Edit %s and add your configuration\n", configFile)
		fmt.Printf("  2. (Optional) Customize %s for your preferred output format\n", templateFile)
		fmt.Printf("  3. Run: spotify-to-markdown <command> --help for usage information\n\n")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().Bool("force", false, "Overwrite existing files")
	initCmd.Flags().String("template-file", "spotify-to-markdown.md", "Name of custom template file to create")
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
