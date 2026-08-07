package session

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

var (
	testNow     = time.Date(2026, time.August, 6, 12, 30, 0, 0, time.UTC)
	testCartID  = uuid.MustParse("01890f3a-dc00-7cc2-98c4-dc0c0c07398f")
	testCartID2 = uuid.MustParse("01890f3a-dc01-7a31-a7b1-1b57a85c5412")
)

const testSecret = "test-only-cart-cookie-secret-32-bytes"

func TestCartConfigFromLookup(t *testing.T) {
	testCases := []struct {
		name       string
		values     map[string]string
		wantErr    error
		wantSecure bool
	}{
		{name: "secret missing", values: map[string]string{"CART_COOKIE_SECURE": "false"}, wantErr: ErrCartCookieSecretMissing},
		{name: "secret too short", values: map[string]string{"CART_COOKIE_SECRET": "too-short", "CART_COOKIE_SECURE": "false"}, wantErr: ErrCartCookieSecretTooShort},
		{name: "secure missing", values: map[string]string{"CART_COOKIE_SECRET": testSecret}, wantErr: ErrCartCookieSecureMissing},
		{name: "secure invalid", values: map[string]string{"CART_COOKIE_SECRET": testSecret, "CART_COOKIE_SECURE": "TRUE"}, wantErr: ErrCartCookieSecureInvalid},
		{name: "secure false", values: map[string]string{"CART_COOKIE_SECRET": testSecret, "CART_COOKIE_SECURE": "false"}, wantSecure: false},
		{name: "secure true", values: map[string]string{"CART_COOKIE_SECRET": testSecret, "CART_COOKIE_SECURE": "true"}, wantSecure: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			lookup := func(key string) (string, bool) {
				value, ok := testCase.values[key]
				return value, ok
			}
			config, err := cartConfigFromLookup(lookup)
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("expected error %v, got %v", testCase.wantErr, err)
			}
			if testCase.wantErr != nil {
				if strings.Contains(err.Error(), testSecret) || strings.Contains(err.Error(), "too-short") {
					t.Fatalf("configuration error leaked the supplied secret: %v", err)
				}
				return
			}
			if config.Secret != testSecret || config.Secure != testCase.wantSecure {
				t.Fatalf("unexpected valid config: %+v", config)
			}
		})
	}
}

func TestNewCartManagerRejectsInsecureSecrets(t *testing.T) {
	for _, config := range []Config{{}, {Secret: "short"}} {
		if _, err := NewCartManager(config); err == nil {
			t.Fatalf("expected configuration %+v to be rejected", config)
		}
	}
}

func TestCartTokenUsesExactSignedPayload(t *testing.T) {
	manager := newTestManager(t, false, testCartID)
	token := manager.signToken(testCartID, testNow)
	segments := strings.Split(token, ".")
	if len(segments) != 4 {
		t.Fatalf("expected four token segments, got %q", token)
	}
	payload := "v1." + testCartID.String() + "." + strings.TrimSpace(segments[2])
	if strings.Join(segments[:3], ".") != payload {
		t.Fatalf("unexpected signed payload %q", strings.Join(segments[:3], "."))
	}

	mac := hmac.New(sha256.New, []byte(testSecret))
	_, _ = mac.Write([]byte(payload))
	wantSignature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if segments[3] != wantSignature {
		t.Fatalf("unexpected signature")
	}
	if strings.Contains(segments[3], "=") {
		t.Fatal("signature contains Base64 padding")
	}

	parsedID, err := manager.verifyToken(token, testNow)
	if err != nil {
		t.Fatalf("verify valid token: %v", err)
	}
	if parsedID != testCartID || parsedID.Version() != uuid.Version(7) {
		t.Fatalf("unexpected UUIDv7 %s", parsedID)
	}
}

func TestCartTokenRejectsMalformedOrUnauthenticatedValues(t *testing.T) {
	manager := newTestManager(t, false, testCartID)
	valid := manager.signToken(testCartID, testNow)
	segments := strings.Split(valid, ".")
	v4 := uuid.MustParse("18e48710-3274-4dc2-9d42-3e612061e324")

	otherManager, err := NewCartManager(Config{Secret: "different-test-cart-secret-at-least-32-bytes"})
	if err != nil {
		t.Fatalf("create manager with different secret: %v", err)
	}

	testCases := []struct {
		name    string
		token   string
		manager *CartManager
	}{
		{name: "raw legacy UUID", token: testCartID.String()},
		{name: "tampered token", token: valid[:len(valid)-1] + replacementFor(valid[len(valid)-1])},
		{name: "modified UUID", token: strings.Replace(valid, testCartID.String(), testCartID2.String(), 1)},
		{name: "modified date", token: strings.Replace(valid, segments[2], "1786019401", 1)},
		{name: "truncated signature", token: strings.Join(append(segments[:3], segments[3][:20]), ".")},
		{name: "padded signature", token: valid + "="},
		{name: "missing segment", token: strings.Join(segments[:3], ".")},
		{name: "extra segment", token: valid + ".extra"},
		{name: "unknown version", token: "v2." + strings.Join(segments[1:], ".")},
		{name: "UUID not v7", token: manager.signToken(v4, testNow)},
		{name: "noncanonical UUID", token: strings.Replace(valid, testCartID.String(), strings.ToUpper(testCartID.String()), 1)},
		{name: "issued at nonnumeric", token: strings.Replace(valid, segments[2], "not-a-number", 1)},
		{name: "issued at noncanonical", token: strings.Replace(valid, segments[2], "+"+segments[2], 1)},
		{name: "too long", token: strings.Repeat("a", maxCartTokenLength+1)},
		{name: "leading space", token: " " + valid},
		{name: "interior space", token: strings.Replace(valid, ".", ". ", 1)},
		{name: "null byte", token: valid + "\x00"},
		{name: "different secret", token: valid, manager: otherManager},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			verifier := manager
			if testCase.manager != nil {
				verifier = testCase.manager
			}
			if parsedID, err := verifier.verifyToken(testCase.token, testNow); err == nil {
				t.Fatalf("expected token to be rejected, got UUID %s", parsedID)
			}
		})
	}
}

func TestCartTokenEnforcesCryptographicLifetime(t *testing.T) {
	manager := newTestManager(t, false, testCartID)
	testCases := []struct {
		name      string
		issuedAt  time.Time
		wantError error
	}{
		{name: "exactly within expiration limit", issuedAt: testNow.Add(-CartTokenLifetime)},
		{name: "expired", issuedAt: testNow.Add(-CartTokenLifetime - time.Second), wantError: ErrExpiredCartToken},
		{name: "future within skew", issuedAt: testNow.Add(maxFutureClockSkew)},
		{name: "future beyond skew", issuedAt: testNow.Add(maxFutureClockSkew + time.Second), wantError: ErrFutureCartToken},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			token := manager.signToken(testCartID, testCase.issuedAt)
			parsedID, err := manager.verifyToken(token, testNow)
			if !errors.Is(err, testCase.wantError) {
				t.Fatalf("expected error %v, got %v", testCase.wantError, err)
			}
			if testCase.wantError == nil && parsedID != testCartID {
				t.Fatalf("expected UUID %s, got %s", testCartID, parsedID)
			}
		})
	}
}

func TestCartTokenRejectsSameLengthIncorrectSignature(t *testing.T) {
	manager := newTestManager(t, false, testCartID)
	token := manager.signToken(testCartID, testNow)
	tampered := token[:len(token)-1] + replacementFor(token[len(token)-1])
	if _, err := manager.verifyToken(tampered, testNow); !errors.Is(err, ErrInvalidCartToken) {
		t.Fatalf("expected a same-length incorrect signature to be rejected, got %v", err)
	}
}

func TestCartMiddlewareCreatesAuthenticatedCookieAndContextOnFirstRequest(t *testing.T) {
	manager := newTestManager(t, false, testCartID)
	request := httptest.NewRequest(http.MethodGet, "/carrito", nil)
	recorder := httptest.NewRecorder()

	var receivedID uuid.UUID
	manager.Middleware(func(w http.ResponseWriter, r *http.Request) {
		var ok bool
		receivedID, ok = CartIDFromContext(r.Context())
		if !ok {
			t.Error("handler did not receive a cart UUID from context")
		}
		_, _ = w.Write([]byte("ok"))
	})(recorder, request)

	if recorder.Code != http.StatusOK || recorder.Body.String() != "ok" {
		t.Fatalf("unexpected handler response: %d %q", recorder.Code, recorder.Body.String())
	}
	if receivedID != testCartID || receivedID.Version() != uuid.Version(7) {
		t.Fatalf("handler received unexpected UUIDv7 %s", receivedID)
	}
	setCookieHeaders := recorder.Header().Values("Set-Cookie")
	if len(setCookieHeaders) != 1 {
		t.Fatalf("expected one Set-Cookie header, got %d", len(setCookieHeaders))
	}

	cookie := responseCartCookie(t, recorder)
	if cookie.Name != CartCookieName || cookie.Path != "/" || cookie.Domain != "" {
		t.Fatalf("unexpected cookie scope: %+v", cookie)
	}
	if !cookie.HttpOnly || cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("unexpected cookie security attributes: %+v", cookie)
	}
	if cookie.MaxAge != int(CartTokenLifetime/time.Second) {
		t.Fatalf("unexpected MaxAge %d", cookie.MaxAge)
	}
	if !cookie.Expires.Equal(testNow.Add(CartTokenLifetime)) {
		t.Fatalf("unexpected Expires %s", cookie.Expires)
	}
	if strings.Contains(recorder.Body.String(), testSecret) || strings.Contains(recorder.Body.String(), strings.Split(cookie.Value, ".")[3]) {
		t.Fatalf("response body leaked session material: %q", recorder.Body.String())
	}

	parsedID, err := manager.verifyToken(cookie.Value, testNow)
	if err != nil || parsedID != testCartID {
		t.Fatalf("new cookie is not a valid authenticated token: id=%s err=%v", parsedID, err)
	}
}

func TestCartMiddlewarePreservesValidCookieWithoutRenewal(t *testing.T) {
	manager := newTestManager(t, false, testCartID2)
	token := manager.signToken(testCartID, testNow.Add(-time.Hour))
	request := httptest.NewRequest(http.MethodGet, "/carrito", nil)
	request.AddCookie(&http.Cookie{Name: CartCookieName, Value: token})
	recorder := httptest.NewRecorder()

	var receivedID uuid.UUID
	manager.Middleware(func(w http.ResponseWriter, r *http.Request) {
		receivedID, _ = CartIDFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})(recorder, request)

	if receivedID != testCartID {
		t.Fatalf("expected existing UUID %s, got %s", testCartID, receivedID)
	}
	if got := recorder.Header().Values("Set-Cookie"); len(got) != 0 {
		t.Fatalf("valid session was renewed unexpectedly: %v", got)
	}
}

func TestCartMiddlewareReplacesInvalidCookies(t *testing.T) {
	issuer := newTestManager(t, false, testCartID)
	expired := issuer.signToken(testCartID, testNow.Add(-CartTokenLifetime-time.Second))
	valid := issuer.signToken(testCartID, testNow)
	segments := strings.Split(valid, ".")

	testCases := []struct {
		name  string
		value string
	}{
		{name: "legacy raw UUID", value: testCartID.String()},
		{name: "tampered", value: valid[:len(valid)-1] + replacementFor(valid[len(valid)-1])},
		{name: "expired", value: expired},
		{name: "unknown version", value: "v2." + strings.Join(segments[1:], ".")},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			manager := newTestManager(t, false, testCartID2)
			request := httptest.NewRequest(http.MethodGet, "/carrito", nil)
			request.AddCookie(&http.Cookie{Name: CartCookieName, Value: testCase.value})
			recorder := httptest.NewRecorder()

			var receivedID uuid.UUID
			manager.Middleware(func(w http.ResponseWriter, r *http.Request) {
				receivedID, _ = CartIDFromContext(r.Context())
			})(recorder, request)

			if receivedID != testCartID2 || receivedID == testCartID {
				t.Fatalf("invalid client identity reached the handler: %s", receivedID)
			}
			if got := len(recorder.Header().Values("Set-Cookie")); got != 1 {
				t.Fatalf("expected one replacement cookie, got %d", got)
			}
		})
	}
}

func TestCartMiddlewareSecureConfiguration(t *testing.T) {
	for _, secure := range []bool{false, true} {
		t.Run(map[bool]string{false: "false", true: "true"}[secure], func(t *testing.T) {
			manager := newTestManager(t, secure, testCartID)
			recorder := httptest.NewRecorder()
			manager.Middleware(func(http.ResponseWriter, *http.Request) {})(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
			if cookie := responseCartCookie(t, recorder); cookie.Secure != secure {
				t.Fatalf("expected Secure=%t, got %+v", secure, cookie)
			}
		})
	}
}

func TestCartMiddlewareRejectsAmbiguousDuplicateCookies(t *testing.T) {
	manager := newTestManager(t, false, testCartID2)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Cookie", CartCookieName+"="+manager.signToken(testCartID, testNow)+"; "+CartCookieName+"="+manager.signToken(testCartID2, testNow))
	recorder := httptest.NewRecorder()

	var receivedID uuid.UUID
	manager.Middleware(func(w http.ResponseWriter, r *http.Request) {
		receivedID, _ = CartIDFromContext(r.Context())
	})(recorder, request)

	if receivedID != testCartID2 {
		t.Fatalf("expected a fresh identity for duplicate cookies, got %s", receivedID)
	}
	if got := len(recorder.Header().Values("Set-Cookie")); got != 1 {
		t.Fatalf("expected exactly one replacement cookie, got %d", got)
	}
}

func TestCartMiddlewareHandlesGeneratorFailureWithoutCallingHandler(t *testing.T) {
	manager, err := NewCartManager(Config{
		Secret: testSecret,
		Clock:  func() time.Time { return testNow },
		UUIDGenerator: func() (uuid.UUID, error) {
			return uuid.Nil, errors.New("entropy unavailable")
		},
	})
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}
	called := false
	recorder := httptest.NewRecorder()
	manager.Middleware(func(http.ResponseWriter, *http.Request) { called = true })(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if called || recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected controlled failure without handler call: called=%t status=%d", called, recorder.Code)
	}
	if strings.Contains(recorder.Body.String(), "entropy") || strings.Contains(recorder.Body.String(), testSecret) {
		t.Fatalf("internal detail leaked: %q", recorder.Body.String())
	}
}

func FuzzVerifyTokenDoesNotPanic(f *testing.F) {
	manager, err := NewCartManager(Config{Secret: testSecret, Clock: func() time.Time { return testNow }})
	if err != nil {
		f.Fatalf("create manager: %v", err)
	}
	f.Add("")
	f.Add("v1.not-a-uuid.0.invalid")
	f.Add(manager.signToken(testCartID, testNow))
	f.Add("\x00 . . . ")

	f.Fuzz(func(t *testing.T, value string) {
		_, _ = manager.verifyToken(value, testNow)
	})
}

func newTestManager(t *testing.T, secure bool, generatedID uuid.UUID) *CartManager {
	t.Helper()

	manager, err := NewCartManager(Config{
		Secret: testSecret,
		Secure: secure,
		Clock:  func() time.Time { return testNow },
		UUIDGenerator: func() (uuid.UUID, error) {
			return generatedID, nil
		},
	})
	if err != nil {
		t.Fatalf("create cart manager: %v", err)
	}
	return manager
}

func responseCartCookie(t *testing.T, recorder *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()

	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one response cookie, got %d", len(cookies))
	}
	return cookies[0]
}

func replacementFor(value byte) string {
	if value == 'A' {
		return "B"
	}
	return "A"
}
