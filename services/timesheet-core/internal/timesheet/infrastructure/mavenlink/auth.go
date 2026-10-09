package mavenlink

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"timesheet-bot/internal/timesheet/application"
)

// Auth is one way of signing in to the Mavenlink API. The endpoint code in
// [Client] is shared; only the credentials differ.
type Auth interface {
	// Authorize adds the credentials to req. Before a request that changes
	// data it may first fetch what the mode needs, such as a CSRF token.
	Authorize(ctx context.Context, req *http.Request) error
	// Rejected returns an error wrapping [application.ErrUnauthorized] if
	// resp, the answer to req, shows the credentials were rejected, or nil.
	Rejected(req *http.Request, resp *http.Response) error
	// Secrets returns the values that must never appear in an error.
	Secrets() []string
}

// Bearer signs in with an OAuth token from an app an account administrator
// registered (Settings > API). It is the mode Kantata documents.
type Bearer struct{ token string }

// NewBearer returns a Bearer that sends token.
func NewBearer(token string) *Bearer { return &Bearer{token: token} }

var _ Auth = (*Bearer)(nil)

// Authorize sets the Authorization header.
func (b *Bearer) Authorize(_ context.Context, req *http.Request) error {
	req.Header.Set("Authorization", "Bearer "+b.token)
	return nil
}

// Rejected maps 401 to [application.ErrUnauthorized].
func (b *Bearer) Rejected(_ *http.Request, resp *http.Response) error {
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("%w: mavenlink answered 401: check MAVENLINK_API_TOKEN", application.ErrUnauthorized)
	}
	return nil
}

// Secrets returns the token.
func (b *Bearer) Secrets() []string { return []string{b.token} }

// ErrSessionExpired is wrapped in the error a [SessionCookie] returns when
// Mavenlink no longer accepts the cookie.
var ErrSessionExpired = fmt.Errorf("%w: mavenlink session cookie expired: copy a fresh Cookie header into MAVENLINK_SESSION_COOKIE",
	application.ErrUnauthorized)

// SessionCookie signs in with the Cookie header of a logged-in browser
// session, for users who cannot get an OAuth token. Kantata does not document
// this mode: the session expires and must be copied again. Like the web
// client, every request carries the page's CSRF token and the page as
// Referer.
type SessionCookie struct {
	cookie      string
	csrfPageURL string
	http        *http.Client

	mu      sync.Mutex
	fetched bool
	csrf    string
}

// NewSessionCookie returns a SessionCookie that sends cookie and reads the
// CSRF token from the page at csrfPageURL through hc. It never follows a
// redirect, so a redirect to the login page is seen as an expired session.
func NewSessionCookie(cookie, csrfPageURL string, hc *http.Client) *SessionCookie {
	return &SessionCookie{cookie: cookie, csrfPageURL: csrfPageURL, http: withoutRedirects(hc)}
}

var _ Auth = (*SessionCookie)(nil)

// Authorize sets the Cookie, Referer, X-Requested-With and X-CSRF-Token
// headers, and the web client's Content-Type on a request with a body. The
// token is fetched once and cached; if the page has none, requests go
// without it.
func (s *SessionCookie) Authorize(ctx context.Context, req *http.Request) error {
	token, err := s.csrfToken(ctx)
	if err != nil {
		return err
	}
	req.Header.Set("Cookie", s.cookie)
	req.Header.Set("Referer", s.csrfPageURL)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	if token != "" {
		req.Header.Set("X-CSRF-Token", token)
	}
	if req.Body != nil {
		req.Header.Set("Content-Type", "application/json;charset=utf-8")
	}
	return nil
}

// Rejected maps an expired session to [ErrSessionExpired]: a 401, a redirect
// to the login page, or an HTML page where the API answers JSON. A 403 or 422
// to a request sent without a CSRF token is reported as a CSRF rejection.
func (s *SessionCookie) Rejected(req *http.Request, resp *http.Response) error {
	switch {
	case loginResponse(resp),
		strings.HasPrefix(strings.ToLower(resp.Header.Get("Content-Type")), "text/html"):
		return ErrSessionExpired
	case req.Header.Get("X-CSRF-Token") == "" &&
		(resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnprocessableEntity):
		return fmt.Errorf("%w: mavenlink answered %s, likely a CSRF rejection: no csrf-token meta tag was found on mavenlink.csrf_page_url",
			application.ErrUnauthorized, resp.Status)
	}
	return nil
}

// Secrets returns the cookie and the cached CSRF token.
func (s *SessionCookie) Secrets() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return []string{s.cookie, s.csrf}
}

func (s *SessionCookie) csrfToken(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fetched {
		return s.csrf, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.csrfPageURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Cookie", s.cookie)
	resp, err := s.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("mavenlink csrf page: %w", err)
	}
	defer resp.Body.Close()
	if loginResponse(resp) {
		return "", ErrSessionExpired
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("mavenlink csrf page: %s", resp.Status)
	}
	page, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", fmt.Errorf("mavenlink csrf page: %w", err)
	}
	s.csrf, s.fetched = ParseCSRFToken(string(page)), true
	return s.csrf, nil
}

// loginResponse reports whether resp sends the browser to sign in: a 401, or
// a redirect to Microsoft Entra ID or to a /login path.
func loginResponse(resp *http.Response) bool {
	if resp.StatusCode == http.StatusUnauthorized {
		return true
	}
	if resp.StatusCode < 300 || resp.StatusCode >= 400 {
		return false
	}
	loc, err := resp.Location()
	if err != nil {
		return false
	}
	return strings.EqualFold(loc.Hostname(), "login.microsoftonline.com") ||
		strings.Contains(strings.ToLower(loc.Path), "/login")
}

var (
	csrfNameFirst    = regexp.MustCompile(`(?i)<meta\s[^>]*name\s*=\s*["']csrf-token["'][^>]*content\s*=\s*["']([^"']*)["']`)
	csrfContentFirst = regexp.MustCompile(`(?i)<meta\s[^>]*content\s*=\s*["']([^"']*)["'][^>]*name\s*=\s*["']csrf-token["']`)
)

// ParseCSRFToken returns the content of the page's
// <meta name="csrf-token" content="..."> tag, unescaped, or "" if there is
// none.
func ParseCSRFToken(page string) string {
	for _, re := range []*regexp.Regexp{csrfNameFirst, csrfContentFirst} {
		if m := re.FindStringSubmatch(page); m != nil {
			return html.UnescapeString(m[1])
		}
	}
	return ""
}

// withoutRedirects returns a copy of hc that hands a redirect back to the
// caller instead of following it.
func withoutRedirects(hc *http.Client) *http.Client {
	c := *hc
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &c
}
