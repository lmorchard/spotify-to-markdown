package store

import "time"

// PlayView is a denormalized play+track+album+artists row, shaped for rendering.
type PlayView struct {
	PlayedAt time.Time
	Track    TrackView
	Context  *ContextView
}

// TrackView is a single track ready for display.
type TrackView struct {
	ID         string
	Name       string
	URL        string
	URI        string
	DurationMs int
	Album      AlbumView
	Artists    []ArtistView
}

// AlbumView is a single album ready for display.
type AlbumView struct {
	ID   string
	Name string
	URL  string
}

// ArtistView is a single artist ready for display.
type ArtistView struct {
	ID   string
	Name string
	URL  string
}

// ContextView is the optional playback context for a play (playlist, album, etc.).
type ContextView struct {
	Type string
	URI  string
}
