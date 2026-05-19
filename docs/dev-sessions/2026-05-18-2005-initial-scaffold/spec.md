# spotify-to-markdown — Spec

## Purpose

A Go CLI that fetches recent Spotify activity for the authenticated user, persists it locally in SQLite, and emits a single markdown document summarizing that activity.

## Decisions (locked in 2026-05-18)

- **Persistence:** SQLite for dedup + long-term history (Spotify's API only returns the last 50 recently-played tracks, so without local storage we'd lose data between runs).
- **Scope of "recent activity":**
  - Recently played tracks (`GET /me/player/recently-played`)
  - Currently playing (`GET /me/player/currently-playing`)
  - Top tracks & top artists (`GET /me/top/tracks`, `GET /me/top/artists`, short/medium/long time ranges)
  - Recently saved/liked tracks (`GET /me/tracks`)
- **Output:** Single markdown file, fully overwritten each run.
- **Stack:** Cobra + Viper + SQLite + Logrus (per `lmorchard-agent-skills:go-cli-builder`). Templates enabled for markdown output.

## Auth model

Spotify OAuth2 Authorization Code flow with PKCE (no client secret required for a local public client).

- First run: `spotify-to-markdown auth` opens browser, runs a local callback listener on `127.0.0.1:<port>`, exchanges the code for tokens, stores refresh + access tokens in SQLite (or the config dir).
- Subsequent runs: refresh access token as needed.
- Required scopes:
  - `user-read-recently-played`
  - `user-read-currently-playing`
  - `user-top-read`
  - `user-library-read`

User supplies their own Spotify app's `client_id` in config. (Spotify requires registering an app at developer.spotify.com; redirect URI must match the local callback.)

## Commands

- `spotify-to-markdown init` — write `spotify-to-markdown.yaml` + a default markdown template.
- `spotify-to-markdown auth` — run the OAuth flow once to obtain refresh token.
- `spotify-to-markdown fetch` — pull all configured activity types into SQLite (dedup on insert).
- `spotify-to-markdown render` — render the markdown file from current DB state.
- `spotify-to-markdown run` — `fetch` then `render` in one shot (default convenience command).
- `spotify-to-markdown version` (built-in)

## Non-goals (for this session)

- No playlists, episodes, podcasts, audiobook activity.
- No per-day/per-week file splitting (output is a single overwritten file).
- No scheduling/cron integration (user can wire that up externally).
- No multi-user support — single-user local tool.
