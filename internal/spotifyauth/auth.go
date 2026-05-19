package spotifyauth

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	authorizeURL = "https://accounts.spotify.com/authorize"
	tokenURL     = "https://accounts.spotify.com/api/token"
	callbackPath = "/callback"
)

// DefaultScopes covers all of phase 1+2 fetchers, so a one-time auth grants
// permission for recently-played, currently-playing, top tracks/artists and
// saved tracks. Re-auth is only needed if this list ever changes.
var DefaultScopes = []string{
	"user-read-recently-played",
	"user-read-currently-playing",
	"user-top-read",
	"user-library-read",
}

// Config holds the parameters needed to authenticate against Spotify.
type Config struct {
	ClientID     string
	RedirectPort int
	Scopes       []string
}

// RedirectURI returns the OAuth redirect URI derived from the configured port.
// This MUST be registered byte-for-byte in the Spotify developer dashboard.
func (c Config) RedirectURI() string {
	return fmt.Sprintf("http://127.0.0.1:%d%s", c.RedirectPort, callbackPath)
}

// Authenticator handles the interactive OAuth dance and the silent refresh flow.
type Authenticator struct {
	cfg  Config
	db   *sql.DB
	http *http.Client
}

// New constructs an Authenticator. db is required and must already have the
// schema applied. http is optional; if nil, http.DefaultClient is used.
func New(cfg Config, db *sql.DB, httpClient *http.Client) *Authenticator {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if len(cfg.Scopes) == 0 {
		cfg.Scopes = DefaultScopes
	}
	return &Authenticator{cfg: cfg, db: db, http: httpClient}
}

// RedirectURI returns the OAuth callback URI this authenticator will listen on.
func (a *Authenticator) RedirectURI() string {
	return a.cfg.RedirectURI()
}

// AccessToken returns a valid access token, refreshing automatically if the
// stored one is expired. Returns ErrNoToken if the user has never authed.
func (a *Authenticator) AccessToken(ctx context.Context) (string, error) {
	tok, err := LoadToken(a.db)
	if err != nil {
		return "", err
	}
	if !tok.IsExpired() {
		return tok.AccessToken, nil
	}

	refreshed, err := a.refresh(ctx, tok.RefreshToken)
	if err != nil {
		return "", fmt.Errorf("refresh access token: %w", err)
	}
	// Spotify may or may not rotate the refresh token; preserve the prior one
	// if a new one wasn't returned.
	if refreshed.RefreshToken == "" {
		refreshed.RefreshToken = tok.RefreshToken
	}
	if err := SaveToken(a.db, refreshed); err != nil {
		return "", fmt.Errorf("save refreshed token: %w", err)
	}
	return refreshed.AccessToken, nil
}

// RunInteractiveFlow performs the full PKCE authorization-code flow:
// opens the user's browser, captures the callback locally, exchanges the
// code for tokens, and persists them. Blocks until the flow completes,
// the user cancels, or ctx is cancelled.
func (a *Authenticator) RunInteractiveFlow(ctx context.Context) error {
	if a.cfg.ClientID == "" {
		return errors.New("spotify.client_id is not set in config")
	}

	verifier, err := generateCodeVerifier(64)
	if err != nil {
		return err
	}
	challenge := codeChallengeS256(verifier)

	state, err := randomState()
	if err != nil {
		return err
	}

	authURL := a.buildAuthorizeURL(challenge, state)

	// Spin up the local callback listener BEFORE opening the browser to avoid
	// a race where the redirect arrives before we're ready.
	codeCh := make(chan callbackResult, 1)
	srv, listenErr := a.startCallbackServer(state, codeCh)
	if listenErr != nil {
		return fmt.Errorf("start callback server: %w", listenErr)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	fmt.Println()
	fmt.Println("Open this URL in your browser to authorize Spotify access:")
	fmt.Println()
	fmt.Println("    " + authURL)
	fmt.Println()
	fmt.Println("(Attempting to open it for you now...)")
	fmt.Println()

	if err := openBrowser(authURL); err != nil {
		// Non-fatal: the URL is printed above.
		fmt.Printf("(Could not auto-launch browser: %v)\n\n", err)
	}

	var cb callbackResult
	select {
	case cb = <-codeCh:
	case <-ctx.Done():
		return ctx.Err()
	}
	if cb.err != nil {
		return cb.err
	}

	tok, err := a.exchangeCode(ctx, cb.code, verifier)
	if err != nil {
		return fmt.Errorf("exchange code for token: %w", err)
	}
	if err := SaveToken(a.db, tok); err != nil {
		return fmt.Errorf("save token: %w", err)
	}
	return nil
}

func (a *Authenticator) buildAuthorizeURL(challenge, state string) string {
	q := url.Values{}
	q.Set("client_id", a.cfg.ClientID)
	q.Set("response_type", "code")
	q.Set("redirect_uri", a.cfg.RedirectURI())
	q.Set("scope", strings.Join(a.cfg.Scopes, " "))
	q.Set("state", state)
	q.Set("code_challenge_method", "S256")
	q.Set("code_challenge", challenge)
	return authorizeURL + "?" + q.Encode()
}

type callbackResult struct {
	code string
	err  error
}

func (a *Authenticator) startCallbackServer(expectedState string, out chan<- callbackResult) (*http.Server, error) {
	addr := fmt.Sprintf("127.0.0.1:%d", a.cfg.RedirectPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", addr, err)
	}

	mux := http.NewServeMux()
	var once sync.Once
	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		// Always respond, even on error, so the user's browser doesn't hang.
		if errCode := q.Get("error"); errCode != "" {
			writeCallbackHTML(w, false, "Authorization failed: "+errCode)
			once.Do(func() { out <- callbackResult{err: fmt.Errorf("authorization error: %s", errCode)} })
			return
		}

		if q.Get("state") != expectedState {
			writeCallbackHTML(w, false, "Invalid state parameter (possible CSRF).")
			once.Do(func() { out <- callbackResult{err: errors.New("state mismatch in callback")} })
			return
		}

		code := q.Get("code")
		if code == "" {
			writeCallbackHTML(w, false, "Missing authorization code.")
			once.Do(func() { out <- callbackResult{err: errors.New("missing code in callback")} })
			return
		}

		writeCallbackHTML(w, true, "Authorization complete — you can close this tab and return to the terminal.")
		once.Do(func() { out <- callbackResult{code: code} })
	})

	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() { _ = srv.Serve(ln) }()
	return srv, nil
}

func writeCallbackHTML(w http.ResponseWriter, ok bool, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	status := "OK"
	if !ok {
		status = "Error"
	}
	fmt.Fprintf(w, `<!doctype html><html><head><title>spotify-to-markdown — %s</title>
<style>body{font-family:system-ui,sans-serif;max-width:36em;margin:4em auto;padding:0 1em;line-height:1.5}</style>
</head><body><h1>spotify-to-markdown</h1><p>%s</p></body></html>`, status, msg)
}

func (a *Authenticator) exchangeCode(ctx context.Context, code, verifier string) (*Token, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", a.cfg.RedirectURI())
	form.Set("client_id", a.cfg.ClientID)
	form.Set("code_verifier", verifier)
	return a.postToken(ctx, form)
}

func (a *Authenticator) refresh(ctx context.Context, refreshToken string) (*Token, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", a.cfg.ClientID)
	return a.postToken(ctx, form)
}

func (a *Authenticator) postToken(ctx context.Context, form url.Values) (*Token, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("post to token endpoint: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var er errorResponse
		_ = json.Unmarshal(body, &er)
		desc := er.ErrorDescription
		if desc == "" {
			desc = strings.TrimSpace(string(body))
		}
		return nil, fmt.Errorf("token endpoint returned %d: %s: %s", resp.StatusCode, er.Error, desc)
	}

	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}
	return &Token{
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
		TokenType:    tr.TokenType,
		Scope:        tr.Scope,
		ExpiresAt:    time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second),
	}, nil
}

// openBrowser tries to launch the platform's default browser pointed at url.
// Failure is non-fatal; the caller should always also print the URL.
func openBrowser(u string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", u)
	case "darwin":
		cmd = exec.Command("open", u)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", u)
	default:
		return fmt.Errorf("unsupported OS %q", runtime.GOOS)
	}
	return cmd.Start()
}
