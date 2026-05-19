# spotify-to-markdown

A small Go CLI that fetches your recent Spotify listening activity via the
Spotify Web API, persists it locally in SQLite, and renders a markdown
document summarizing what you've been listening to.

It's designed for scheduled, unattended use after a one-time interactive
OAuth setup — drop it in a cron job and let it keep a rolling markdown
record of your plays.

> **Status:** Phase 1 (MVP). Currently fetches **recently played tracks**.
> Currently-playing, top tracks/artists, and saved/liked tracks are scoped
> for a later phase. See `docs/dev-sessions/` for the plan.

---

## How it works

1. You register a Spotify Developer app and grab its `client_id`.
2. `spotify-to-markdown auth` runs the OAuth2 Authorization Code + PKCE
   flow once — opens your browser, captures the callback on a local
   `127.0.0.1` port, and stores the resulting refresh + access tokens in
   the local SQLite database.
3. `spotify-to-markdown fetch` pulls your last 50 plays from Spotify and
   upserts them into the database (deduped by play timestamp).
4. `spotify-to-markdown render` reads the database and writes a markdown
   file using an embedded (or user-supplied) Go template.
5. `spotify-to-markdown run` does fetch + render in one shot — meant for
   cron / systemd-timer use.

Spotify refresh tokens don't expire from disuse, so once `auth` has been
completed, the tool can run silently for as long as you don't revoke
access in your Spotify account.

---

## Installation

### From source

Requires Go 1.21+ and CGO (for the SQLite driver).

```sh
git clone https://github.com/lmorchard/spotify-to-markdown.git
cd spotify-to-markdown
make build
```

The binary lands in the project root as `./spotify-to-markdown`. Copy it
somewhere on your `PATH` if you like.

### Pre-built binaries

Tagged releases produce cross-compiled binaries for Linux, macOS, and
Windows via the included GitHub Actions workflow. See the project's
[Releases page](https://github.com/lmorchard/spotify-to-markdown/releases).

---

## Spotify Developer app setup

1. Sign in at <https://developer.spotify.com/dashboard> and create a new
   app (or reuse an existing one).
2. **Copy the Client ID.** You do **not** need the Client Secret — this
   tool uses PKCE, which is designed for public clients with no secret.
3. In the app's settings, add a **Redirect URI**:

       http://127.0.0.1:8888/callback

   The host, path, and port must match byte-for-byte. If you change
   `spotify.redirect_port` in the config below, update the URI here too.

---

## Quick start

In whatever working directory you want the data to live (e.g.
`~/.local/share/spotify-to-markdown/`):

```sh
# 1. Generate config + a copy of the markdown template
spotify-to-markdown init

# 2. Edit spotify-to-markdown.yaml and paste your client_id
#    into the `spotify.client_id` field

# 3. One-time browser auth — opens a Spotify consent page
spotify-to-markdown auth

# 4. Fetch + render
spotify-to-markdown run
```

After step 4 you'll have a `spotify-recent.md` file in the working
directory. Re-running `spotify-to-markdown run` is safe and idempotent —
duplicate plays are ignored on insert.

---

## Configuration

`spotify-to-markdown` looks for `spotify-to-markdown.yaml` in the
current directory by default; pass `--config /path/to/file.yaml` to
override.

> **DB path change (heads-up for existing users):** the default database
> path moved from `./spotify-to-markdown.db` (current directory) to
> `$XDG_STATE_HOME/spotify-to-markdown/state.db` (i.e.
> `~/.local/state/spotify-to-markdown/state.db` on most systems). If you
> have an existing DB you want to keep, either `mv` it to the new
> location or set `--database ./spotify-to-markdown.db` (or the matching
> config key). The new default aligns with `pocketcasts-to-markdown` and
> the `me-to-markdown` orchestrator's expectations.

`init` writes a fully-commented version of the config file with these
keys:

```yaml
# database: "/custom/path/state.db"     # default: $XDG_STATE_HOME/spotify-to-markdown/state.db

verbose: false                          # info-level logs to stderr
debug: false                            # debug-level logs to stderr
log_json: false                         # emit logs as JSON

client_id: ""                           # required; from developer.spotify.com
redirect_port: 8888                     # local OAuth callback port

# scopes (optional; defaults shown below — covers all current + future
# phases so you only auth once)
# scopes:
#   - user-read-recently-played
#   - user-read-currently-playing
#   - user-top-read
#   - user-library-read

output:
  file: "spotify-recent.md"             # written/overwritten on each render
  # template: "spotify-to-markdown.md"  # optional custom template
```

Every config key is reachable via an environment variable with the `SPOTIFY_` prefix; nested keys use `_`:

```bash
export SPOTIFY_CLIENT_ID="..."
export SPOTIFY_REDIRECT_PORT=8888
export SPOTIFY_OUTPUT_FILE="recent.md"
```

---

## Commands

| Command  | What it does |
| -------- | ------------ |
| `init`   | Write a default `spotify-to-markdown.yaml` and a copy of the embedded markdown template. Use `--force` to overwrite. |
| `auth`   | Run the one-time interactive OAuth/PKCE flow and persist tokens. |
| `fetch`  | Pull the last 50 plays from Spotify and upsert into the DB. |
| `render` | Read the DB and write the markdown file. Accepts optional `--since`/`--until` to render a specific window; without them, renders the most recent 50 plays. |
| `run`    | `fetch` followed by `render`. Intended for scheduled use. |
| `export` | Orchestrator-friendly `fetch` + windowed `render` with the canonical `--since/--until/-o` flag shape used by [`me-to-markdown`](https://github.com/lmorchard/me-to-markdown). |
| `version`| Print version, commit, and build date. |

Run any command with `--help` for full usage details.

---

## Customizing the markdown output

`spotify-to-markdown init` drops a copy of the embedded default template
into `spotify-to-markdown.md`. Edit that file and point
`output.template` at it in your config to take over rendering.

Templates use Go's
[`text/template`](https://pkg.go.dev/text/template) syntax. The
top-level data passed in is:

```go
type RenderData struct {
    Generated time.Time     // when the render ran
    Plays     []store.PlayView
}
```

Each `PlayView` has `PlayedAt`, `Track` (with `Name`, `URL`, `URI`,
`DurationMs`, `Album`, `Artists`), and optional `Context` (the playlist
or album the track was played from, if Spotify reported one).

The following helper functions are available in templates:

| Function          | Example                                    | Output                  |
| ----------------- | ------------------------------------------ | ----------------------- |
| `formatTime`      | `{{ formatTime .PlayedAt "2006-01-02" }}` | `2026-05-18`            |
| `formatDuration`  | `{{ formatDuration 215000 }}`              | `3:35`                  |
| `artistLinks`     | `{{ artistLinks .Track.Artists }}`         | `[A](urlA) & [B](urlB)` |
| `artistNames`     | `{{ artistNames .Track.Artists }}`         | `A & B`                 |

---

## Scheduled (unattended) use

Once `auth` has been run once, the tool refreshes its access token
silently and needs no further human interaction. Examples:

**crontab — run every 15 minutes:**

```cron
*/15 * * * * cd /home/me/spotify && /usr/local/bin/spotify-to-markdown run >> spotify.log 2>&1
```

**systemd user timer:**

```ini
# ~/.config/systemd/user/spotify-to-markdown.service
[Unit]
Description=Refresh spotify-recent.md

[Service]
Type=oneshot
WorkingDirectory=%h/spotify
ExecStart=/usr/local/bin/spotify-to-markdown run

# ~/.config/systemd/user/spotify-to-markdown.timer
[Unit]
Description=Run spotify-to-markdown every 15 minutes

[Timer]
OnBootSec=2m
OnUnitActiveSec=15m

[Install]
WantedBy=timers.target
```

```sh
systemctl --user enable --now spotify-to-markdown.timer
```

> **Caveat — track loss under heavy listening:** Spotify's
> `/me/player/recently-played` endpoint only ever returns the most
> recent 50 plays, with no pagination beyond that window. If you listen
> to more than 50 tracks between fetches, the older ones fall off the
> Spotify side before this tool can see them. Run the fetcher
> frequently enough to stay under that ceiling. (Cursor-based catch-up
> using `after=<unix_ms>` is on the roadmap.)

---

## What lives in the database

`spotify-to-markdown.db` is a normal SQLite file you can poke at with
`sqlite3` or any client. Tables of note:

- `auth_tokens` — single row holding your OAuth access + refresh tokens
  and expiry. **Treat this file like a credential.**
- `plays` — one row per play, keyed by `played_at` (Spotify's
  millisecond-precision ISO 8601 timestamp). This is the natural dedup
  key.
- `tracks`, `albums`, `artists`, `track_artists` — normalized metadata
  upserted from each fetch.
- `schema_migrations` — version tracker for the naive migration system.

The schema file lives at `internal/database/schema.sql`.

---

## Development

The project follows the conventions from
[`lmorchard-agent-skills/go-cli-builder`](https://github.com/lmorchard/lmorchard-agent-skills):
Cobra commands in `cmd/`, business logic in `internal/<package>/`.

```sh
make setup     # install gofumpt + golangci-lint
make build     # build ./spotify-to-markdown
make run       # build + run
make format    # go fmt + gofumpt
make lint      # golangci-lint
make test      # go test ./...
make clean     # remove binary + *.db files
```

Package layout:

```
cmd/                     # cobra commands (root, auth, fetch, render, run, init, version)
internal/config/         # typed config struct
internal/database/       # sqlite connection, embedded schema, migrations
internal/spotifyauth/    # OAuth2 PKCE flow + token persistence + auto-refresh
internal/spotify/        # Web API client + response types
internal/store/          # transactional upserts + denormalized query views
internal/templates/      # embedded default template + renderer w/ funcmap
docs/dev-sessions/       # session-by-session spec/plan/todo/notes
```

---

## Security notes

- The local SQLite file contains your Spotify refresh + access tokens.
  Don't commit it, share it, or back it up to anywhere you wouldn't put
  a password.
- The OAuth callback listener binds to `127.0.0.1` only — it isn't
  reachable from other hosts on your network.
- No `client_secret` is used or stored; PKCE is the whole story.

---

## License

TBD — Les hasn't set one yet.
