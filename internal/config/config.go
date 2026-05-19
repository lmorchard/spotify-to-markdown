package config

// Config holds application configuration.
type Config struct {
	// Core settings
	Database string
	Verbose  bool
	Debug    bool
	LogJSON  bool

	// Spotify API + auth settings
	Spotify SpotifyConfig

	// Markdown output settings
	Output OutputConfig
}

// SpotifyConfig holds OAuth client + scope settings.
type SpotifyConfig struct {
	ClientID     string
	RedirectPort int
	Scopes       []string
}

// OutputConfig configures the rendered markdown file.
type OutputConfig struct {
	File     string
	Template string // optional path to a custom Go template; empty = use embedded default
}
