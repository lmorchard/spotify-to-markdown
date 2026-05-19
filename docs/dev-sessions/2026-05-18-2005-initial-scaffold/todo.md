# Todo — Phase 1

- [x] Scaffold project via `go-cli-builder` skill (`--templates`, default DB)
- [x] Fix `go.mod` module path → `github.com/lmorchard/spotify-to-markdown`
- [x] `go mod tidy`
- [x] Design + write `internal/database/schema.sql`
  - [x] `auth_tokens` (single-row, refresh + access + expiry)
  - [x] `tracks` (track metadata)
  - [x] `artists` (artist metadata)
  - [x] `albums` (album metadata)
  - [x] `plays` (played_at PK, track_id, context info)
  - [x] `track_artists` (many-to-many join with position)
- [x] Implement `internal/spotifyauth/` — PKCE OAuth2 flow
  - [x] Generate code verifier/challenge
  - [x] Spin up local `http.Server` on 127.0.0.1:<port>
  - [x] Open browser (`open`/`xdg-open`/`rundll32`)
  - [x] Exchange code → tokens
  - [x] Persist tokens to SQLite (upsert)
  - [x] Silent refresh of access tokens
- [x] Implement `internal/spotify/` — API client
  - [x] Auto-injects Bearer token
  - [x] `GetRecentlyPlayed(limit, after)` method
- [x] Implement `internal/store/` — DB upserts + query views
  - [x] `SavePlay` (track + album + artists + play, in a tx)
  - [x] `RecentPlays(limit)` for rendering
- [x] `cmd/auth.go` — one-shot interactive auth
- [x] `cmd/fetch.go` — pull recently-played, upsert into DB
- [x] `cmd/render.go` — read DB, render template, write markdown file
- [x] `cmd/run.go` — fetch + render
- [x] Embedded default markdown template (`internal/templates/default.md`)
  - [x] Helper funcs: `formatTime`, `formatDuration`, `artistLinks`, `artistNames`
- [x] Update `spotify-to-markdown.yaml.example` and `init` command's embedded config
- [x] `make format && make lint && make test` clean
- [ ] Manual end-to-end test with Les's real Spotify client_id (requires Les)
- [ ] Commit (deferred until Les approves)

## Deferred to Phase 2

- Pagination beyond Spotify's 50-item limit on `recently-played`
- Cursor-based `after=<unix_ms>` fetching using max(played_at)
- Currently-playing endpoint
- Top tracks/artists (short/medium/long term)
- Saved/liked tracks
- Unit tests
