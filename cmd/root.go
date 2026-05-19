package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lmorchard/spotify-to-markdown/internal/config"
	"github.com/lmorchard/spotify-to-markdown/internal/spotifyauth"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	log     = logrus.New()
	cfg     *config.Config
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "spotify-to-markdown",
	Short: "Fetch recent Spotify activity and render it as markdown",
	Long: `spotify-to-markdown fetches your recent Spotify listening activity via the
Spotify Web API, persists it locally in SQLite for deduplication and history,
and renders a markdown document summarizing your recent plays.

A one-time interactive ` + "`auth`" + ` command performs the OAuth2 (PKCE) flow.
After that, ` + "`fetch`" + ` and ` + "`render`" + ` (or the combined ` + "`run`" + `) can be invoked
unattended — ideal for scheduled use (cron, systemd timers, etc.).`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		initConfig()
		setupLogging()
	},
}

// Execute adds all child commands to the root command and sets appropriate flags.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Configuration file flag
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./spotify-to-markdown.yaml)")

	// Logging flags
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().Bool("debug", false, "debug output")
	rootCmd.PersistentFlags().Bool("log-json", false, "output logs in JSON format")

	// Database flag
	rootCmd.PersistentFlags().String("database", "", "database file path (default: $XDG_STATE_HOME/spotify-to-markdown/state.db)")

	// Bind flags to viper
	_ = viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	_ = viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	_ = viper.BindPFlag("log_json", rootCmd.PersistentFlags().Lookup("log-json"))
	_ = viper.BindPFlag("database", rootCmd.PersistentFlags().Lookup("database"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Search for config in current directory
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName("spotify-to-markdown")
	}

	// Set defaults
	viper.SetDefault("database", defaultDatabasePath())
	viper.SetDefault("verbose", false)
	viper.SetDefault("debug", false)
	viper.SetDefault("log_json", false)
	viper.SetDefault("redirect_port", 8888)
	viper.SetDefault("scopes", spotifyauth.DefaultScopes)
	viper.SetDefault("output.file", "spotify-recent.md")

	// Read in environment variables that match. Every config key is reachable
	// via SPOTIFY_<KEY>, with `.` in nested keys becoming `_`
	// (e.g. output.file -> SPOTIFY_OUTPUT_FILE).
	viper.SetEnvPrefix("SPOTIFY")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err != nil {
		if cfgFile != "" {
			// Only error if config was explicitly specified
			fmt.Fprintf(os.Stderr, "Error reading config file: %v\n", err)
			os.Exit(1)
		}
	}
}

// setupLogging configures the logger based on configuration
func setupLogging() {
	if viper.GetBool("log_json") {
		log.SetFormatter(&logrus.JSONFormatter{})
	} else {
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}

	if viper.GetBool("debug") {
		log.SetLevel(logrus.DebugLevel)
	} else if viper.GetBool("verbose") {
		log.SetLevel(logrus.InfoLevel)
	} else {
		log.SetLevel(logrus.WarnLevel)
	}
}

// GetConfig returns the application configuration, loading it if necessary
func GetConfig() *config.Config {
	if cfg == nil {
		cfg = &config.Config{
			Database: viper.GetString("database"),
			Verbose:  viper.GetBool("verbose"),
			Debug:    viper.GetBool("debug"),
			LogJSON:  viper.GetBool("log_json"),
			Spotify: config.SpotifyConfig{
				ClientID:     viper.GetString("client_id"),
				RedirectPort: viper.GetInt("redirect_port"),
				Scopes:       viper.GetStringSlice("scopes"),
			},
			Output: config.OutputConfig{
				File:     viper.GetString("output.file"),
				Template: viper.GetString("output.template"),
			},
		}
	}
	return cfg
}

// GetLogger returns the configured logger
func GetLogger() *logrus.Logger {
	return log
}

// xdgStateDir returns $XDG_STATE_HOME/spotify-to-markdown,
// falling back to ~/.local/state/spotify-to-markdown.
func xdgStateDir() string {
	if v := os.Getenv("XDG_STATE_HOME"); v != "" {
		return filepath.Join(v, "spotify-to-markdown")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".local", "state", "spotify-to-markdown")
}

// defaultDatabasePath returns the canonical default location for the
// SQLite archive: $XDG_STATE_HOME/spotify-to-markdown/state.db.
func defaultDatabasePath() string {
	return filepath.Join(xdgStateDir(), "state.db")
}
