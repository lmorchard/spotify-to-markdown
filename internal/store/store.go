package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lmorchard/spotify-to-markdown/internal/spotify"
)

// Store wraps a *sql.DB with domain-specific persistence and query helpers.
type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

// SavePlay persists a single PlayHistoryItem: the track (with album and
// artists) and the play row itself. Idempotent on re-run thanks to UPSERT /
// INSERT OR IGNORE on the play's played_at primary key.
func (s *Store) SavePlay(item spotify.PlayHistoryItem) (inserted bool, err error) {
	tx, err := s.db.Begin()
	if err != nil {
		return false, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := upsertAlbum(tx, item.Track.Album); err != nil {
		return false, err
	}
	for _, ar := range item.Track.Artists {
		if err := upsertArtist(tx, ar); err != nil {
			return false, err
		}
	}
	for _, ar := range item.Track.Album.Artists {
		if err := upsertArtist(tx, ar); err != nil {
			return false, err
		}
	}
	if err := upsertTrack(tx, item.Track); err != nil {
		return false, err
	}
	if err := replaceTrackArtists(tx, item.Track); err != nil {
		return false, err
	}

	inserted, err = insertPlay(tx, item)
	if err != nil {
		return false, err
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit tx: %w", err)
	}
	return inserted, nil
}

func upsertArtist(tx *sql.Tx, a spotify.SimpleArtist) error {
	if a.ID == "" {
		return nil
	}
	_, err := tx.Exec(`
		INSERT INTO artists (spotify_id, name, uri, url, fetched_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(spotify_id) DO UPDATE SET
			name       = excluded.name,
			uri        = excluded.uri,
			url        = excluded.url,
			fetched_at = CURRENT_TIMESTAMP
	`, a.ID, a.Name, a.URI, a.ExternalURLs.Spotify)
	if err != nil {
		return fmt.Errorf("upsert artist %s: %w", a.ID, err)
	}
	return nil
}

func upsertAlbum(tx *sql.Tx, al spotify.SimpleAlbum) error {
	if al.ID == "" {
		return nil
	}
	_, err := tx.Exec(`
		INSERT INTO albums (spotify_id, name, album_type, release_date, total_tracks, uri, url, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(spotify_id) DO UPDATE SET
			name         = excluded.name,
			album_type   = excluded.album_type,
			release_date = excluded.release_date,
			total_tracks = excluded.total_tracks,
			uri          = excluded.uri,
			url          = excluded.url,
			fetched_at   = CURRENT_TIMESTAMP
	`, al.ID, al.Name, al.AlbumType, al.ReleaseDate, al.TotalTracks, al.URI, al.ExternalURLs.Spotify)
	if err != nil {
		return fmt.Errorf("upsert album %s: %w", al.ID, err)
	}
	return nil
}

func upsertTrack(tx *sql.Tx, t spotify.Track) error {
	if t.ID == "" {
		return errors.New("track has no spotify id")
	}
	_, err := tx.Exec(`
		INSERT INTO tracks (spotify_id, name, album_id, duration_ms, track_number, disc_number, explicit, popularity, preview_url, uri, url, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(spotify_id) DO UPDATE SET
			name         = excluded.name,
			album_id     = excluded.album_id,
			duration_ms  = excluded.duration_ms,
			track_number = excluded.track_number,
			disc_number  = excluded.disc_number,
			explicit     = excluded.explicit,
			popularity   = excluded.popularity,
			preview_url  = excluded.preview_url,
			uri          = excluded.uri,
			url          = excluded.url,
			fetched_at   = CURRENT_TIMESTAMP
	`,
		t.ID, t.Name, nullIfEmpty(t.Album.ID), t.DurationMs, t.TrackNumber, t.DiscNumber,
		boolToInt(t.Explicit), t.Popularity, t.PreviewURL, t.URI, t.ExternalURLs.Spotify)
	if err != nil {
		return fmt.Errorf("upsert track %s: %w", t.ID, err)
	}
	return nil
}

// replaceTrackArtists clears and rewrites the track_artists join rows for this
// track, preserving artist position order from the API response.
func replaceTrackArtists(tx *sql.Tx, t spotify.Track) error {
	if _, err := tx.Exec(`DELETE FROM track_artists WHERE track_id = ?`, t.ID); err != nil {
		return fmt.Errorf("clear track_artists for %s: %w", t.ID, err)
	}
	for i, a := range t.Artists {
		if a.ID == "" {
			continue
		}
		if _, err := tx.Exec(`
			INSERT INTO track_artists (track_id, artist_id, position)
			VALUES (?, ?, ?)
		`, t.ID, a.ID, i); err != nil {
			return fmt.Errorf("insert track_artist %s/%s: %w", t.ID, a.ID, err)
		}
	}
	return nil
}

func insertPlay(tx *sql.Tx, item spotify.PlayHistoryItem) (bool, error) {
	contextType := sql.NullString{}
	contextURI := sql.NullString{}
	if item.Context != nil {
		contextType.Valid, contextType.String = true, item.Context.Type
		contextURI.Valid, contextURI.String = true, item.Context.URI
	}

	res, err := tx.Exec(`
		INSERT OR IGNORE INTO plays (played_at, track_id, context_type, context_uri, fetched_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
	`, item.PlayedAt, item.Track.ID, contextType, contextURI)
	if err != nil {
		return false, fmt.Errorf("insert play %s: %w", item.PlayedAt, err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// RecentPlays returns the most recently played items, newest first, up to limit.
func (s *Store) RecentPlays(limit int) ([]PlayView, error) {
	if limit <= 0 {
		limit = 50
	}

	views, trackIDs, err := s.loadPlayRows(limit)
	if err != nil {
		return nil, err
	}
	if len(views) == 0 {
		return views, nil
	}

	artistsByTrack, err := s.loadArtistsForTracks(trackIDs)
	if err != nil {
		return nil, err
	}
	for i := range views {
		views[i].Track.Artists = artistsByTrack[views[i].Track.ID]
	}
	return views, nil
}

// loadPlayRows reads the play/track/album columns for the most recent plays
// and fully drains the rows iterator before returning. Doing this in one pass
// avoids holding the single SQLite connection while we issue follow-up
// queries (which would deadlock under MaxOpenConns=1).
func (s *Store) loadPlayRows(limit int) ([]PlayView, []string, error) {
	rows, err := s.db.Query(`
		SELECT p.played_at, p.context_type, p.context_uri,
		       t.spotify_id, t.name, t.url, t.uri, t.duration_ms,
		       a.spotify_id, a.name, a.url
		FROM plays p
		JOIN tracks t ON t.spotify_id = p.track_id
		LEFT JOIN albums a ON a.spotify_id = t.album_id
		ORDER BY p.played_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, nil, fmt.Errorf("query recent plays: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var (
		views    []PlayView
		trackIDs []string
	)
	for rows.Next() {
		var (
			playedAtStr               string
			contextType, contextURI   sql.NullString
			trackID, trackName        string
			trackURL, trackURI        sql.NullString
			durationMs                sql.NullInt64
			albumID, albumName, alURL sql.NullString
		)
		if err := rows.Scan(
			&playedAtStr, &contextType, &contextURI,
			&trackID, &trackName, &trackURL, &trackURI, &durationMs,
			&albumID, &albumName, &alURL,
		); err != nil {
			return nil, nil, fmt.Errorf("scan play row: %w", err)
		}

		playedAt, err := time.Parse(time.RFC3339Nano, playedAtStr)
		if err != nil {
			if playedAt, err = time.Parse(time.RFC3339, playedAtStr); err != nil {
				return nil, nil, fmt.Errorf("parse played_at %q: %w", playedAtStr, err)
			}
		}

		pv := PlayView{
			PlayedAt: playedAt.UTC(),
			Track: TrackView{
				ID:         trackID,
				Name:       trackName,
				URL:        trackURL.String,
				URI:        trackURI.String,
				DurationMs: int(durationMs.Int64),
				Album: AlbumView{
					ID:   albumID.String,
					Name: albumName.String,
					URL:  alURL.String,
				},
			},
		}
		if contextType.Valid && contextType.String != "" {
			pv.Context = &ContextView{
				Type: contextType.String,
				URI:  contextURI.String,
			}
		}

		views = append(views, pv)
		trackIDs = append(trackIDs, trackID)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate plays: %w", err)
	}
	return views, trackIDs, nil
}

// loadArtistsForTracks fetches all (track_id → artists) in a single query,
// keyed by track_id. Artists per track are returned in position order.
func (s *Store) loadArtistsForTracks(trackIDs []string) (map[string][]ArtistView, error) {
	if len(trackIDs) == 0 {
		return map[string][]ArtistView{}, nil
	}
	placeholders := strings.Repeat("?,", len(trackIDs))
	placeholders = placeholders[:len(placeholders)-1]

	args := make([]interface{}, len(trackIDs))
	for i, id := range trackIDs {
		args[i] = id
	}

	q := fmt.Sprintf(`
		SELECT ta.track_id, a.spotify_id, a.name, a.url
		FROM track_artists ta
		JOIN artists a ON a.spotify_id = ta.artist_id
		WHERE ta.track_id IN (%s)
		ORDER BY ta.track_id, ta.position ASC
	`, placeholders)

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("query artists for tracks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string][]ArtistView, len(trackIDs))
	for rows.Next() {
		var (
			trackID, id, name string
			url               sql.NullString
		)
		if err := rows.Scan(&trackID, &id, &name, &url); err != nil {
			return nil, fmt.Errorf("scan artist row: %w", err)
		}
		out[trackID] = append(out[trackID], ArtistView{ID: id, Name: name, URL: url.String})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate artists: %w", err)
	}
	return out, nil
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
