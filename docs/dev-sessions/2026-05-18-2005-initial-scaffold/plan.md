# Plan — Initial scaffold

Phased approach so we can land a working slice early and iterate.

## Phase 1 — Scaffold + auth + recently-played (this session, MVP)

1. Run `scaffold_project.py spotify-to-markdown --templates` (database is default).
2. Update `go.mod` module path to `github.com/lmorchard/spotify-to-markdown`.
3. Define SQLite schema: `tracks`, `plays`, `artists`, `albums`, `auth_tokens` (or similar).
4. Implement OAuth2 PKCE flow in `internal/spotifyauth/` with a local `127.0.0.1` callback.
5. Implement `auth` command (one-time interactive flow).
6. Implement Spotify API client in `internal/spotify/` (auto-refreshing token).
7. Implement `fetch` command for `/me/player/recently-played` only.
8. Implement `render` command + a default embedded markdown template covering recently-played.
9. Wire `run` command (fetch + render).
10. `make format && make lint && make test`.

Stop here, verify end-to-end, commit. Land Phase 1 before expanding scope.

## Phase 2 — Expand fetch coverage (next session)

- Add currently-playing, top tracks/artists (short/medium/long), saved tracks fetchers.
- Extend schema for top-*, saved tracks.
- Extend default template to render those sections.

## Phase 3 — Polish (later)

- Improve template (more metadata, links to Spotify URIs).
- Optional: stats summary (most-played artists this week, etc.).
- Release workflow (tag a v0.1.0 when stable).

## Key architectural choices

- **Token storage:** SQLite `auth_tokens` table keyed by some constant ("default") so re-auth simply UPSERTs. Keeps secrets in one file the user already manages.
- **Client ID:** in `spotify-to-markdown.yaml`, not embedded — user registers their own app.
- **Dedup strategy:** recently-played has a `played_at` timestamp per play; primary key `(track_id, played_at)`.
- **Markdown template:** embedded default via `//go:embed`, overridable via `--template-file` flag or config (per the skill's template pattern).
