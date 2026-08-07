package security

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vladwithcode/salon_catalog/internal/session"
)

func TestCSRFConfigFromLookup(t *testing.T) {
	testCases := []struct {
		name    string
		value   string
		present bool
		wantErr error
	}{
		{name: "missing", wantErr: ErrCSRFTrustedOriginsMissing},
		{name: "empty", present: true, wantErr: ErrCSRFTrustedOriginsEmpty},
		{name: "spaces", value: "   ", present: true, wantErr: ErrCSRFTrustedOriginsEmpty},
		{name: "valid", value: "http://localhost:8080", present: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			config, err := csrfConfigFromLookup(func(key string) (string, bool) {
				if key != "CSRF_TRUSTED_ORIGINS" {
					t.Fatalf("unexpected environment lookup %q", key)
				}
				return testCase.value, testCase.present
			})
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("expected error %v, got %v", testCase.wantErr, err)
			}
			if testCase.wantErr == nil && config.TrustedOrigins != testCase.value {
				t.Fatalf("unexpected config: %+v", config)
			}
		})
	}
}

func TestNewCSRFGuardAcceptsAndNormalizesTrustedOrigins(t *testing.T) {
	testCases := []struct {
		name       string
		value      string
		wantOrigin normalizedOrigin
		wantCount  int
	}{
		{name: "HTTP", value: "http://example.com", wantOrigin: normalizedOrigin{scheme: "http", hostname: "example.com", port: "80"}, wantCount: 1},
		{name: "HTTPS", value: "https://example.com", wantOrigin: normalizedOrigin{scheme: "https", hostname: "example.com", port: "443"}, wantCount: 1},
		{name: "multiple with spaces", value: " http://localhost:8080 , https://example.com ", wantOrigin: normalizedOrigin{scheme: "http", hostname: "localhost", port: "8080"}, wantCount: 2},
		{name: "explicit default HTTP port", value: "http://example.com:80", wantOrigin: normalizedOrigin{scheme: "http", hostname: "example.com", port: "80"}, wantCount: 1},
		{name: "explicit default HTTPS port", value: "https://example.com:443", wantOrigin: normalizedOrigin{scheme: "https", hostname: "example.com", port: "443"}, wantCount: 1},
		{name: "uppercase hostname", value: "https://EXAMPLE.COM", wantOrigin: normalizedOrigin{scheme: "https", hostname: "example.com", port: "443"}, wantCount: 1},
		{name: "uppercase scheme", value: "HTTPS://EXAMPLE.COM", wantOrigin: normalizedOrigin{scheme: "https", hostname: "example.com", port: "443"}, wantCount: 1},
		{name: "localhost", value: "http://localhost:3000", wantOrigin: normalizedOrigin{scheme: "http", hostname: "localhost", port: "3000"}, wantCount: 1},
		{name: "IPv6", value: "http://[::1]:8080", wantOrigin: normalizedOrigin{scheme: "http", hostname: "::1", port: "8080"}, wantCount: 1},
		{name: "duplicates removed after normalization", value: "http://example.com,http://EXAMPLE.COM:80/", wantOrigin: normalizedOrigin{scheme: "http", hostname: "example.com", port: "80"}, wantCount: 1},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			guard, err := NewCSRFGuard(CSRFConfig{TrustedOrigins: testCase.value})
			if err != nil {
				t.Fatalf("create guard: %v", err)
			}
			if len(guard.trustedOrigins) != testCase.wantCount {
				t.Fatalf("expected %d normalized origins, got %d", testCase.wantCount, len(guard.trustedOrigins))
			}
			if _, ok := guard.trustedOrigins[testCase.wantOrigin]; !ok {
				t.Fatalf("normalized origin missing: %+v", testCase.wantOrigin)
			}
		})
	}
}

func TestNewCSRFGuardRejectsInvalidConfigurationWithoutLeakingValues(t *testing.T) {
	testCases := []struct {
		name  string
		value string
	}{
		{name: "empty element", value: "https://example.com,,http://localhost:8080"},
		{name: "trailing empty element", value: "https://example.com,"},
		{name: "wildcard", value: "https://*.example.com"},
		{name: "star", value: "*"},
		{name: "null", value: "null"},
		{name: "scheme not allowed", value: "ftp://example.com"},
		{name: "relative URL", value: "/relative"},
		{name: "host missing", value: "https:///path"},
		{name: "userinfo", value: "https://user:password@example.com"},
		{name: "path", value: "https://example.com/path"},
		{name: "query", value: "https://example.com?secret=value"},
		{name: "empty query", value: "https://example.com?"},
		{name: "fragment", value: "https://example.com#secret-fragment"},
		{name: "empty fragment", value: "https://example.com#"},
		{name: "nonnumeric port", value: "https://example.com:abc"},
		{name: "port zero", value: "https://example.com:0"},
		{name: "port too large", value: "https://example.com:65536"},
		{name: "empty port", value: "https://example.com:"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := NewCSRFGuard(CSRFConfig{TrustedOrigins: testCase.value})
			if !errors.Is(err, ErrCSRFTrustedOriginsInvalid) {
				t.Fatalf("expected invalid-origin error, got %v", err)
			}
			if strings.Contains(err.Error(), testCase.value) || strings.Contains(err.Error(), "password") || strings.Contains(err.Error(), "secret-fragment") {
				t.Fatalf("configuration error leaked input: %v", err)
			}
		})
	}
}

func TestCSRFGuardOriginAndRefererRules(t *testing.T) {
	guard := newTestCSRFGuard(t)

	testCases := []struct {
		name        string
		method      string
		host        string
		headers     http.Header
		wantAllowed bool
	}{
		{name: "GET without Origin", method: http.MethodGet, wantAllowed: true},
		{name: "HEAD without Origin", method: http.MethodHead, wantAllowed: true},
		{name: "OPTIONS without Origin", method: http.MethodOptions, wantAllowed: true},
		{name: "trusted Origin", method: http.MethodPost, headers: header("Origin", "https://example.com"), wantAllowed: true},
		{name: "HTTPS implicit port", method: http.MethodPut, headers: header("Origin", "https://EXAMPLE.COM:443"), wantAllowed: true},
		{name: "HTTP implicit port", method: http.MethodPatch, headers: header("Origin", "http://plain.example:80"), wantAllowed: true},
		{name: "different port", method: http.MethodDelete, headers: header("Origin", "https://example.com:444")},
		{name: "different scheme", method: http.MethodPost, headers: header("Origin", "http://example.com")},
		{name: "subdomain", method: http.MethodPost, headers: header("Origin", "https://sub.example.com")},
		{name: "suffix attack", method: http.MethodPost, headers: header("Origin", "https://example.com.attacker.test")},
		{name: "prefix attack", method: http.MethodPost, headers: header("Origin", "https://attacker-example.com")},
		{name: "null Origin", method: http.MethodPost, headers: header("Origin", "null")},
		{name: "malformed Origin", method: http.MethodPost, headers: header("Origin", "https://%zz")},
		{name: "Origin comma list", method: http.MethodPost, headers: header("Origin", "https://example.com, https://attacker.test")},
		{name: "trusted Referer fallback", method: http.MethodPost, headers: header("Referer", "https://example.com/"), wantAllowed: true},
		{name: "Referer path and query", method: http.MethodPut, headers: header("Referer", "https://example.com/catalogo?item=1#details"), wantAllowed: true},
		{name: "untrusted Referer", method: http.MethodPatch, headers: header("Referer", "https://attacker.test/path")},
		{name: "malformed Referer", method: http.MethodDelete, headers: header("Referer", "://invalid")},
		{name: "both absent", method: http.MethodPost},
		{name: "valid Origin ignores different Referer", method: http.MethodPost, headers: headers(map[string]string{"Origin": "https://example.com", "Referer": "https://attacker.test/path"}), wantAllowed: true},
		{name: "invalid Origin never falls back", method: http.MethodPost, headers: headers(map[string]string{"Origin": "https://attacker.test", "Referer": "https://example.com/path"})},
		{name: "X-Forwarded-Host ignored", method: http.MethodPost, headers: header("X-Forwarded-Host", "example.com")},
		{name: "X-Forwarded-Proto ignored", method: http.MethodPost, headers: header("X-Forwarded-Proto", "https")},
		{name: "Forwarded ignored", method: http.MethodPost, headers: header("Forwarded", "host=example.com;proto=https")},
		{name: "trusted Host alone rejected", method: http.MethodPost, host: "example.com"},
		{name: "cookie alone rejected", method: http.MethodPost, headers: header("Cookie", "cart_id=signed-token")},
		{name: "HTMX header alone rejected", method: http.MethodPatch, headers: header("HX-Request", "true")},
		{name: "Sec-Fetch-Site alone rejected", method: http.MethodPost, headers: header("Sec-Fetch-Site", "same-origin")},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(testCase.method, "http://service.invalid/mutation", nil)
			request.Header = testCase.headers.Clone()
			if testCase.host != "" {
				request.Host = testCase.host
			}
			recorder := httptest.NewRecorder()
			called := false
			guard.Protect(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusNoContent)
			})(recorder, request)

			if called != testCase.wantAllowed {
				t.Fatalf("handler called=%t, expected allowed=%t", called, testCase.wantAllowed)
			}
			if testCase.wantAllowed && recorder.Code != http.StatusNoContent {
				t.Fatalf("expected allowed response, got %d", recorder.Code)
			}
			if !testCase.wantAllowed {
				assertCSRFRejected(t, recorder)
			}
		})
	}
}

func TestCSRFGuardRejectsMultipleOriginOrRefererHeaders(t *testing.T) {
	guard := newTestCSRFGuard(t)
	testCases := []struct {
		name   string
		header string
	}{
		{name: "multiple Origin", header: "Origin"},
		{name: "multiple Referer", header: "Referer"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/mutation", nil)
			request.Header.Add(testCase.header, "https://example.com")
			request.Header.Add(testCase.header, "https://example.com")
			recorder := httptest.NewRecorder()
			guard.Protect(func(http.ResponseWriter, *http.Request) {
				t.Fatal("handler ran with duplicate security headers")
			})(recorder, request)
			assertCSRFRejected(t, recorder)
		})
	}
}

func TestCSRFRejectionIsExactAndDoesNotLeakDetails(t *testing.T) {
	guard := newTestCSRFGuard(t)
	request := httptest.NewRequest(http.MethodPost, "/internal/cart/path", nil)
	request.Host = "internal-host.example"
	request.Header.Set("Origin", "https://received-origin.attacker")
	request.Header.Set("Referer", "https://received-referer.attacker/secret")
	request.Header.Set("Cookie", "cart_id=secret-cookie-value")
	recorder := httptest.NewRecorder()

	guard.Protect(func(http.ResponseWriter, *http.Request) {
		t.Fatal("rejected request reached handler")
	})(recorder, request)

	assertCSRFRejected(t, recorder)
	for _, leak := range []string{"example.com", "received-origin", "received-referer", "internal-host", "secret-cookie", "internal/cart"} {
		if strings.Contains(recorder.Body.String(), leak) {
			t.Fatalf("response leaked %q: %q", leak, recorder.Body.String())
		}
	}
	if len(recorder.Header().Values("Set-Cookie")) != 0 {
		t.Fatalf("rejection emitted a cookie: %v", recorder.Header().Values("Set-Cookie"))
	}
}

func TestCSRFGuardRunsBeforeCartSessionAndLoaders(t *testing.T) {
	guard := newTestCSRFGuard(t)
	generatedUUIDs := 0
	manager, err := session.NewCartManager(session.Config{
		Secret: "test-only-cart-cookie-secret-32-bytes",
		Secure: false,
		Clock:  func() time.Time { return time.Date(2026, time.August, 6, 18, 0, 0, 0, time.UTC) },
		UUIDGenerator: func() (uuid.UUID, error) {
			generatedUUIDs++
			return uuid.MustParse("01890f3a-dc00-7cc2-98c4-dc0c0c07398f"), nil
		},
	})
	if err != nil {
		t.Fatalf("create cart manager: %v", err)
	}
	issueRecorder := httptest.NewRecorder()
	manager.Middleware(func(http.ResponseWriter, *http.Request) {})(issueRecorder, httptest.NewRequest(http.MethodGet, "/carrito", nil))
	issuedCookies := issueRecorder.Result().Cookies()
	if len(issuedCookies) != 1 {
		t.Fatalf("expected one valid cart cookie, got %d", len(issuedCookies))
	}
	generatedUUIDs = 0

	loaderCalls := 0
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/carrito", nil)
	request.AddCookie(issuedCookies[0])
	guard.Protect(manager.Middleware(func(http.ResponseWriter, *http.Request) {
		loaderCalls++
	}))(recorder, request)

	assertCSRFRejected(t, recorder)
	if generatedUUIDs != 0 {
		t.Fatalf("rejected request generated %d UUIDs", generatedUUIDs)
	}
	if loaderCalls != 0 {
		t.Fatalf("rejected request made %d loader calls", loaderCalls)
	}
	if got := recorder.Header().Values("Set-Cookie"); len(got) != 0 {
		t.Fatalf("rejected request emitted cart_id: %v", got)
	}
}

func FuzzCSRFGuardDoesNotPanic(f *testing.F) {
	guard, err := NewCSRFGuard(CSRFConfig{TrustedOrigins: "https://example.com"})
	if err != nil {
		f.Fatalf("create guard: %v", err)
	}
	for _, seed := range []string{"", "null", "https://example.com", "https://example.com,https://attacker.test", "\x00", "https://[::1]:443"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, value string) {
		request := httptest.NewRequest(http.MethodPost, "/mutation", nil)
		request.Header.Set("Origin", value)
		recorder := httptest.NewRecorder()
		guard.Protect(func(http.ResponseWriter, *http.Request) {})(recorder, request)
	})
}

func newTestCSRFGuard(t *testing.T) *CSRFGuard {
	t.Helper()
	guard, err := NewCSRFGuard(CSRFConfig{TrustedOrigins: "https://example.com,http://plain.example,http://localhost:8080"})
	if err != nil {
		t.Fatalf("create CSRF guard: %v", err)
	}
	return guard
}

func assertCSRFRejected(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, recorder.Code)
	}
	if recorder.Body.String() != "Solicitud no permitida." {
		t.Fatalf("unexpected rejection body %q", recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/plain; charset=utf-8" {
		t.Fatalf("unexpected Content-Type %q", got)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("unexpected Cache-Control %q", got)
	}
	if got := recorder.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("unexpected X-Content-Type-Options %q", got)
	}
}

func header(name, value string) http.Header {
	result := make(http.Header)
	result.Set(name, value)
	return result
}

func headers(values map[string]string) http.Header {
	result := make(http.Header)
	for name, value := range values {
		result.Set(name, value)
	}
	return result
}
