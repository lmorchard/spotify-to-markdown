# Notes — Initial scaffold session

## Open questions / things to confirm with Les

- Spotify Developer app — Les has one but needs to grab credentials. The tool
  needs the **client_id** from https://developer.spotify.com/dashboard, set as
  `spotify.client_id` in the config. The app also needs
  `http://127.0.0.1:8888/callback` (or whatever port is configured) registered
  in its Redirect URIs list.
- End-to-end test: not run yet because it needs Les's real client_id. Once
  Les drops that into `spotify-to-markdown.yaml` and runs `spotify-to-markdown
  auth`, the rest should follow.

## Decisions log

- 2026-05-18: PKCE (Authorization Code with PKCE) — no client secret, ideal
  for a local CLI. Refresh tokens don't expire on disuse so scheduled cron
  works fine after one interactive auth.
- 2026-05-18: All four scopes requested up front (recently-played,
  currently-playing, top-read, library-read) so future fetcher additions
  don't require re-auth.
- 2026-05-18: Single-row `auth_tokens` table (id always = 1, CHECK
  constraint) for OAuth tokens. Keeps secrets in the same SQLite file the
  user already manages.
- 2026-05-18: `played_at` is the natural primary key for `plays` (Spotify
  returns millisecond-precision ISO 8601 timestamps that are unique
  per-user). `INSERT OR IGNORE` handles dedup on re-runs.
- 2026-05-18: Template helpers (`formatTime`, `formatDuration`,
  `artistLinks`) live in `internal/templates` package alongside the embedded
  default. Custom user templates inherit the same funcmap.

## Final summary

Phase 1 MVP complete and building/linting/testing clean.

### Package layout

- `cmd/` — root + auth/fetch/render/run/init/version commands
- `internal/config/` — typed config struct
- `internal/database/` — SQLite connection + migrations + schema
- `internal/spotifyauth/` — PKCE OAuth flow, token persistence,
  silent-refresh access-token source
- `internal/spotify/` — Web API client + response types; depends on a
  `TokenSource` interface (satisfied by spotifyauth.Authenticator)
- `internal/store/` — persistence layer: upserts a PlayHistoryItem into
  artists/albums/tracks/track_artists/plays in a single tx, and queries
  denormalized PlayView rows for rendering
- `internal/templates/` — embedded default + renderer with helper funcs

### Smoke tests done

- `--help` shows all commands
- `init` writes a rich `spotify-to-markdown.yaml` + a copy of the default
  template
- `render` against an empty DB produces the "no plays yet" placeholder
- `make format`, `make lint`, `make test` all clean

### What's missing before this is useful in production

1. Les drops his `client_id` into `spotify-to-markdown.yaml`
2. Adds `http://127.0.0.1:8888/callback` as a Redirect URI on his Spotify app
3. Runs `spotify-to-markdown auth` once, completes browser consent
4. Runs `spotify-to-markdown run` (or `fetch` then `render`)
5. Verify the produced `spotify-recent.md` looks reasonable
