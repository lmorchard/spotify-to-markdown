-- Initial database schema for spotify-to-markdown
-- This is version 1 of the schema

-- Migration tracking table
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Insert initial version
INSERT OR IGNORE INTO schema_migrations (version) VALUES (1);

-- Spotify OAuth2 tokens (single-row table; id always = 1)
CREATE TABLE IF NOT EXISTS auth_tokens (
    id            INTEGER PRIMARY KEY CHECK (id = 1),
    access_token  TEXT NOT NULL,
    refresh_token TEXT NOT NULL,
    token_type    TEXT NOT NULL,
    scope         TEXT NOT NULL,
    expires_at    DATETIME NOT NULL,
    updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS artists (
    spotify_id TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    uri        TEXT,
    url        TEXT,
    fetched_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS albums (
    spotify_id   TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    album_type   TEXT,
    release_date TEXT,
    total_tracks INTEGER,
    uri          TEXT,
    url          TEXT,
    fetched_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS tracks (
    spotify_id   TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    album_id     TEXT REFERENCES albums(spotify_id),
    duration_ms  INTEGER,
    track_number INTEGER,
    disc_number  INTEGER,
    explicit     INTEGER,
    popularity   INTEGER,
    preview_url  TEXT,
    uri          TEXT,
    url          TEXT,
    fetched_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Many-to-many: a track may have multiple artists
CREATE TABLE IF NOT EXISTS track_artists (
    track_id  TEXT NOT NULL REFERENCES tracks(spotify_id),
    artist_id TEXT NOT NULL REFERENCES artists(spotify_id),
    position  INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (track_id, artist_id)
);

CREATE INDEX IF NOT EXISTS idx_track_artists_artist ON track_artists(artist_id);

-- Recently-played history. played_at (ISO 8601 from Spotify, millisecond precision)
-- is unique per user and serves as the natural primary key for dedup.
CREATE TABLE IF NOT EXISTS plays (
    played_at    TEXT PRIMARY KEY,
    track_id     TEXT NOT NULL REFERENCES tracks(spotify_id),
    context_type TEXT,
    context_uri  TEXT,
    fetched_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_plays_track ON plays(track_id);
CREATE INDEX IF NOT EXISTS idx_plays_played_at_desc ON plays(played_at DESC);
