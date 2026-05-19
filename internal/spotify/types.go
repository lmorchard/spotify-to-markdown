package spotify

// ExternalURLs maps url-kind to URL. For Spotify objects, the "spotify" key
// holds the open.spotify.com link.
type ExternalURLs struct {
	Spotify string `json:"spotify"`
}

// SimpleArtist is the small artist object returned inside track and album payloads.
type SimpleArtist struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	URI          string       `json:"uri"`
	ExternalURLs ExternalURLs `json:"external_urls"`
}

// SimpleAlbum is the album object returned inside a track payload.
type SimpleAlbum struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	URI          string         `json:"uri"`
	ExternalURLs ExternalURLs   `json:"external_urls"`
	AlbumType    string         `json:"album_type"`
	ReleaseDate  string         `json:"release_date"`
	TotalTracks  int            `json:"total_tracks"`
	Artists      []SimpleArtist `json:"artists"`
}

// Track is the full track object returned by /me/player/recently-played etc.
type Track struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	URI          string         `json:"uri"`
	ExternalURLs ExternalURLs   `json:"external_urls"`
	DurationMs   int            `json:"duration_ms"`
	TrackNumber  int            `json:"track_number"`
	DiscNumber   int            `json:"disc_number"`
	Explicit     bool           `json:"explicit"`
	Popularity   int            `json:"popularity"`
	PreviewURL   string         `json:"preview_url"`
	Album        SimpleAlbum    `json:"album"`
	Artists      []SimpleArtist `json:"artists"`
}

// Context is the playback context attached to a play (playlist, album, etc.).
// Nil-equivalent in API responses when context is unknown.
type Context struct {
	Type         string       `json:"type"`
	URI          string       `json:"uri"`
	ExternalURLs ExternalURLs `json:"external_urls"`
}

// PlayHistoryItem is a single entry in the recently-played response.
type PlayHistoryItem struct {
	Track    Track    `json:"track"`
	PlayedAt string   `json:"played_at"`
	Context  *Context `json:"context"`
}

// RecentlyPlayedResponse is the envelope for GET /me/player/recently-played.
type RecentlyPlayedResponse struct {
	Items   []PlayHistoryItem `json:"items"`
	Next    string            `json:"next"`
	Limit   int               `json:"limit"`
	Cursors struct {
		After  string `json:"after"`
		Before string `json:"before"`
	} `json:"cursors"`
	Href string `json:"href"`
}
