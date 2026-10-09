package mavenlink_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"timesheet-bot/internal/timesheet/application"
	"timesheet-bot/internal/timesheet/infrastructure/mavenlink"
)

const (
	fakeCookie = "_app_session_9=fake-session; user_id_9=1"
	fakeCSRF   = "csrf-fake+token=="
	csrfPage   = `<html><head><meta name="csrf-param" content="authenticity_token" />
<meta name="csrf-token" content="csrf-fake+token==" /></head><body></body></html>`
)

func TestParseCSRFToken(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, page, want string
	}{
		{name: "name first", page: csrfPage, want: fakeCSRF},
		{name: "content first", page: `<meta content="abc" name="csrf-token">`, want: "abc"},
		{name: "single quotes and case", page: `<META NAME='csrf-token' CONTENT='abc'>`, want: "abc"},
		{name: "html escaped", page: `<meta name="csrf-token" content="a&amp;b&#43;c">`, want: "a&b+c"},
		{name: "other meta only", page: `<meta name="csrf-param" content="authenticity_token">`, want: ""},
		{name: "empty page", page: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := mavenlink.ParseCSRFToken(tt.page); got != tt.want {
				t.Errorf("ParseCSRFToken(%q) = %q, want %q", tt.page, got, tt.want)
			}
		})
	}
}

// fakeInstance serves the CSRF page at /timesheets and hands every /api/v1
// request to api.
type fakeInstance struct {
	pageHits atomic.Int32
	page     string
	api      http.HandlerFunc
}

func (f *fakeInstance) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/timesheets":
		f.pageHits.Add(1)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(f.page))
	case strings.HasPrefix(r.URL.Path, "/api/v1/"):
		f.api(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func cookieClient(srv *httptest.Server) *mavenlink.Client {
	auth := mavenlink.NewSessionCookie(fakeCookie, srv.URL+"/timesheets", srv.Client())
	return mavenlink.NewClient(srv.URL+"/api/v1", auth, srv.Client())
}

func TestSessionCookieSendsBrowserHeaders(t *testing.T) {
	t.Parallel()
	seen := make(chan http.Header, 2)
	inst := &fakeInstance{page: csrfPage, api: func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Clone()
		h.Set("X-Method", r.Method)
		seen <- h
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			_, _ = w.Write([]byte(`{"results":[{"key":"time_entries","id":"55"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"meta":{"page_count":1},"results":[]}`))
	}}
	srv := httptest.NewServer(inst)
	t.Cleanup(srv.Close)
	c := cookieClient(srv)

	if _, err := c.Created(t.Context(), week); err != nil {
		t.Fatalf("Created() error = %v, want nil", err)
	}
	if _, err := c.Create(t.Context(), application.TimeEntry{WorkspaceID: 7, Day: monday(t), Minutes: 45}); err != nil {
		t.Fatalf("Create() error = %v, want nil", err)
	}

	for range 2 {
		h := <-seen
		method := h.Get("X-Method")
		checks := map[string]string{
			"Cookie":           fakeCookie,
			"X-Csrf-Token":     fakeCSRF,
			"Referer":          srv.URL + "/timesheets",
			"X-Requested-With": "XMLHttpRequest",
			"Authorization":    "",
		}
		if method == http.MethodPost {
			checks["Content-Type"] = "application/json;charset=utf-8"
		}
		for k, want := range checks {
			if got := h.Get(k); got != want {
				t.Errorf("%s request header %s = %q, want %q", method, k, got, want)
			}
		}
	}
	if got := inst.pageHits.Load(); got != 1 {
		t.Errorf("CSRF page fetched %d times, want 1 (cached)", got)
	}
}

func TestSessionCookieExpired(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		api  http.HandlerFunc
	}{
		{name: "401", api: func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusUnauthorized) }},
		{name: "redirect to Microsoft Entra ID", api: func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Location", "https://login.microsoftonline.com/tenant/saml2?SAMLRequest=x")
			w.WriteHeader(http.StatusFound)
		}},
		{name: "redirect to /login", api: func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Location", "/login")
			w.WriteHeader(http.StatusFound)
		}},
		{name: "HTML page instead of JSON", api: func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte("<html>Sign in</html>"))
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(&fakeInstance{page: csrfPage, api: tt.api})
			t.Cleanup(srv.Close)

			_, err := cookieClient(srv).Create(t.Context(), application.TimeEntry{WorkspaceID: 7, Day: monday(t), Minutes: 45})
			if !errors.Is(err, mavenlink.ErrSessionExpired) || !errors.Is(err, application.ErrUnauthorized) {
				t.Fatalf("Create() error = %v, want ErrSessionExpired wrapping ErrUnauthorized", err)
			}
			if !strings.Contains(err.Error(), "MAVENLINK_SESSION_COOKIE") {
				t.Errorf("Create() error = %q, want it to say which variable to refresh", err)
			}
		})
	}
}

func TestSessionCookieExpiredOnCSRFPage(t *testing.T) {
	t.Parallel()
	var apiHits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			t.Errorf("client followed the redirect to /login, want it handed back")
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			apiHits.Add(1)
		}
		w.Header().Set("Location", "/login")
		w.WriteHeader(http.StatusFound)
	}))
	t.Cleanup(srv.Close)

	_, err := cookieClient(srv).Created(t.Context(), week)
	if !errors.Is(err, mavenlink.ErrSessionExpired) {
		t.Errorf("Created() error = %v, want ErrSessionExpired", err)
	}
	if got := apiHits.Load(); got != 0 {
		t.Errorf("API called %d times, want none once the CSRF page shows the session expired", got)
	}
}

func TestSessionCookieMissingCSRFIsReported(t *testing.T) {
	t.Parallel()
	inst := &fakeInstance{page: "<html><head></head></html>", api: func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-CSRF-Token"); got != "" {
			t.Errorf("X-CSRF-Token = %q, want none when the page has no meta tag", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"errors":[{"message":"Can't verify CSRF token authenticity."}]}`))
	}}
	srv := httptest.NewServer(inst)
	t.Cleanup(srv.Close)

	_, err := cookieClient(srv).Create(t.Context(), application.TimeEntry{WorkspaceID: 7, Day: monday(t), Minutes: 45})
	if !errors.Is(err, application.ErrUnauthorized) || !strings.Contains(err.Error(), "CSRF") {
		t.Errorf("Create() error = %v, want ErrUnauthorized naming a CSRF rejection", err)
	}
}

func TestSessionCookieNeverInErrors(t *testing.T) {
	t.Parallel()
	inst := &fakeInstance{page: csrfPage, api: func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		// A server that echoes the request back must not leak the cookie or the CSRF token.
		_, _ = w.Write([]byte(`{"echo":"` + r.Header.Get("Cookie") + " " + r.Header.Get("X-CSRF-Token") + `"}`))
	}}
	srv := httptest.NewServer(inst)
	t.Cleanup(srv.Close)

	_, err := cookieClient(srv).Create(t.Context(), application.TimeEntry{WorkspaceID: 7, Day: monday(t), Minutes: 45})
	if err == nil {
		t.Fatal("Create() error = nil, want a 400 error")
	}
	for _, secret := range []string{fakeCookie, fakeCSRF, "fake-session"} {
		if strings.Contains(err.Error(), secret) {
			t.Errorf("Create() error = %q, want it to omit %q", err, secret)
		}
	}
}
