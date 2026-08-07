package dbtest

import "testing"

// TestCheckConfig covers section 12 of 5B7B6C: the guard must reject
// everything except a well-formed URL, an explicit destructive opt-in, and
// a database name starting with cart_integration_ — all without making a
// network connection (checkConfig never dials) and without leaking the
// URL or a password into any message.
func TestCheckConfig(t *testing.T) {
	const validURL = "postgres://user:s3cr3t-password@db.internal:5432/cart_integration_local?sslmode=disable"

	for _, testCase := range []struct {
		name             string
		rawURL           string
		allowDestructive string
		wantSkip         bool
		wantFatal        bool
	}{
		{name: "variable ausente", rawURL: "", allowDestructive: "true", wantSkip: true},
		{name: "allow destructive ausente", rawURL: validURL, allowDestructive: "", wantSkip: true},
		{name: "allow destructive false", rawURL: validURL, allowDestructive: "false", wantSkip: true},
		{name: "url invalida", rawURL: "postgres://user:pass@%zz/cart_integration_local", allowDestructive: "true", wantFatal: true},
		{name: "nombre vacio", rawURL: "postgres://user:pass@db.internal:5432/", allowDestructive: "true", wantFatal: true},
		{name: "postgres", rawURL: "postgres://user:pass@db.internal:5432/postgres", allowDestructive: "true", wantFatal: true},
		{name: "template0", rawURL: "postgres://user:pass@db.internal:5432/template0", allowDestructive: "true", wantFatal: true},
		{name: "template1", rawURL: "postgres://user:pass@db.internal:5432/template1", allowDestructive: "true", wantFatal: true},
		{name: "salon_catalog", rawURL: "postgres://user:pass@db.internal:5432/salon_catalog", allowDestructive: "true", wantFatal: true},
		{name: "staging", rawURL: "postgres://user:pass@db.internal:5432/staging", allowDestructive: "true", wantFatal: true},
		{name: "production", rawURL: "postgres://user:pass@db.internal:5432/production", allowDestructive: "true", wantFatal: true},
		{name: "prefijo valido", rawURL: validURL, allowDestructive: "true", wantSkip: false, wantFatal: false},
		{name: "prefijo valido variante ci", rawURL: "postgres://user:pass@db.internal:5432/cart_integration_ci", allowDestructive: "true"},
		{name: "prefijo valido variante numerica", rawURL: "postgres://user:pass@db.internal:5432/cart_integration_2026", allowDestructive: "true"},
		{name: "nombre sin prefijo exacto", rawURL: "postgres://user:pass@db.internal:5432/not_cart_integration_local", allowDestructive: "true", wantFatal: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			result := checkConfig(testCase.rawURL, testCase.allowDestructive)

			if testCase.wantSkip {
				if result.skipReason == "" {
					t.Fatalf("expected a skip reason, got dsn=%q fatal=%q", result.dsn, result.fatal)
				}
				return
			}
			if testCase.wantFatal {
				if result.fatal == "" {
					t.Fatalf("expected a fatal reason, got dsn=%q skip=%q", result.dsn, result.skipReason)
				}
				return
			}
			if result.skipReason != "" || result.fatal != "" {
				t.Fatalf("expected a validated dsn, got skip=%q fatal=%q", result.skipReason, result.fatal)
			}
			if result.dsn != testCase.rawURL {
				t.Fatalf("expected the dsn to pass through unchanged, got %q", result.dsn)
			}
		})
	}
}

func TestCheckConfigNeverLeaksPasswordOrFullURL(t *testing.T) {
	const secretURL = "postgres://user:s3cr3t-password@db.internal:5432/postgres?sslmode=disable"
	result := checkConfig(secretURL, "true")
	if result.fatal == "" {
		t.Fatal("expected the postgres database name to be rejected")
	}
	for _, secret := range []string{"s3cr3t-password", secretURL, "db.internal:5432"} {
		if containsSubstring(result.fatal, secret) {
			t.Fatalf("fatal message leaked a secret or the full URL: %q contains %q", result.fatal, secret)
		}
	}
}

func TestCheckConfigNeverLeaksURLOnSkip(t *testing.T) {
	const secretURL = "postgres://user:s3cr3t-password@db.internal:5432/cart_integration_local"
	result := checkConfig(secretURL, "")
	if result.skipReason == "" {
		t.Fatal("expected a skip when the destructive opt-in is missing")
	}
	if containsSubstring(result.skipReason, "s3cr3t-password") {
		t.Fatalf("skip message leaked the password: %q", result.skipReason)
	}
}

func containsSubstring(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
