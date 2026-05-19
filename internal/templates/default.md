# Recent Spotify Activity

_Generated: {{ formatTime .Generated "2006-01-02 15:04 MST" }}_

## Recently Played

{{ if not .Plays -}}
_No plays yet — run `spotify-to-markdown fetch` first._
{{- else -}}
{{ range .Plays }}
- **{{ formatTime .PlayedAt "2006-01-02 15:04" }}** — [{{ .Track.Name }}]({{ .Track.URL }}) by {{ artistLinks .Track.Artists }}{{ if .Track.Album.Name }} _(from [{{ .Track.Album.Name }}]({{ .Track.Album.URL }}))_{{ end }}{{ if .Track.DurationMs }} · {{ formatDuration .Track.DurationMs }}{{ end }}
{{- end }}
{{- end }}
