package templates

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/lmorchard/spotify-to-markdown/internal/store"
)

//go:embed default.md
var defaultTemplate string

// GetDefaultTemplate returns the embedded default template content.
func GetDefaultTemplate() (string, error) {
	return defaultTemplate, nil
}

// RenderData is the value passed to the template at render time.
type RenderData struct {
	Generated time.Time
	Plays     []store.PlayView
}

// Renderer is a parsed template ready to render against RenderData.
type Renderer struct {
	tmpl *template.Template
}

// NewRenderer parses the embedded default template.
func NewRenderer() (*Renderer, error) {
	return newRendererFromString("default", defaultTemplate)
}

// NewRendererFromFile parses a user-supplied template file.
func NewRendererFromFile(path string) (*Renderer, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read template %s: %w", path, err)
	}
	return newRendererFromString(path, string(b))
}

func newRendererFromString(name, body string) (*Renderer, error) {
	t, err := template.New(name).Funcs(funcMap()).Parse(body)
	if err != nil {
		return nil, fmt.Errorf("parse template %s: %w", name, err)
	}
	return &Renderer{tmpl: t}, nil
}

// Render executes the template against data and writes to w.
func (r *Renderer) Render(w io.Writer, data RenderData) error {
	if err := r.tmpl.Execute(w, data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}
	return nil
}

func funcMap() template.FuncMap {
	return template.FuncMap{
		"formatTime":     formatTime,
		"formatDuration": formatDuration,
		"artistLinks":    artistLinks,
		"artistNames":    artistNames,
	}
}

// formatTime renders t using the given Go reference layout, in the user's
// local timezone.
func formatTime(t time.Time, layout string) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format(layout)
}

// formatDuration converts a duration in milliseconds to "M:SS" (or "H:MM:SS"
// if >= 1 hour).
func formatDuration(ms int) string {
	if ms <= 0 {
		return ""
	}
	totalSec := ms / 1000
	h := totalSec / 3600
	m := (totalSec % 3600) / 60
	s := totalSec % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

// artistLinks renders a slice of artists as "[A](urlA), [B](urlB) & [C](urlC)".
// If an artist has no URL, just the name is emitted.
func artistLinks(artists []store.ArtistView) string {
	parts := make([]string, 0, len(artists))
	for _, a := range artists {
		if a.URL != "" {
			parts = append(parts, fmt.Sprintf("[%s](%s)", a.Name, a.URL))
		} else {
			parts = append(parts, a.Name)
		}
	}
	return joinNames(parts)
}

// artistNames renders a slice of artists as plain names: "A, B & C".
func artistNames(artists []store.ArtistView) string {
	parts := make([]string, 0, len(artists))
	for _, a := range artists {
		parts = append(parts, a.Name)
	}
	return joinNames(parts)
}

func joinNames(parts []string) string {
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	case 2:
		return parts[0] + " & " + parts[1]
	default:
		return strings.Join(parts[:len(parts)-1], ", ") + " & " + parts[len(parts)-1]
	}
}
