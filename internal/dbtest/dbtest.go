// Package dbtest provides real-PostgreSQL test infrastructure shared by
// internal/db's and internal/cart's integration suites (5B7B6C). It is
// never imported by production code paths — only by _test.go files in
// those two packages — and every destructive operation it performs is
// gated behind three independent checks (see RequireIsolatedDatabase and
// ResetDedicatedDatabase): an explicit URL, an explicit destructive-opt-in,
// and a database name that must start with cart_integration_.
//
// Earlier (5B7B6B) this package isolated tests with a temporary schema and
// a search_path override, and applied only a hand-picked subset of
// sql/migrations to avoid migrations that hardcode "public.". That subset
// approach is gone: some migrations (20250728173855_add_product_fulltext_search.sql
// onward) reference "public." explicitly, including inside a trigger
// function, so a non-public search_path either breaks those migrations
// outright or, worse, silently touches the real public schema. The fix is
// not to patch those migrations (out of scope, and forbidden this phase)
// but to run the full chain the way it was written: against a dedicated,
// disposable database's own public schema, reset with
// DROP SCHEMA public CASCADE / CREATE SCHEMA public before every run.
package dbtest

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// urlEnvVar and allowDestructiveEnvVar are deliberately distinct from
// DATABASE_URL and any production variable: a missing or accidental value
// here must never fall back to a real database.
const (
	urlEnvVar              = "CART_INTEGRATION_TEST_DATABASE_URL"
	allowDestructiveEnvVar = "CART_INTEGRATION_TEST_ALLOW_DESTRUCTIVE"

	// requiredDatabaseNamePrefix is a positive allowlist, not a blocklist:
	// only a database whose name starts with this prefix is ever touched,
	// which rejects "postgres", "template0", "template1", "salon_catalog",
	// "staging", "production", and anything else by construction — no
	// separate list of forbidden names to keep in sync.
	requiredDatabaseNamePrefix = "cart_integration_"
)

// configResult is checkConfig's pure, side-effect-free result: which env
// var (if any) is missing, or which guard the supplied value failed, or a
// validated dsn ready to use. Keeping this pure and separate from
// RequireIsolatedDatabase lets the guard logic itself be unit tested
// (section 12) without touching *testing.T or a network.
type configResult struct {
	dsn        string
	skipReason string // non-empty: caller should t.Skip with this message
	fatal      string // non-empty: caller should t.Fatal with this message
}

// checkConfig validates the two integration-test env vars and the target
// database name, in the order section 5 requires: URL present, then
// destructive opt-in present, then URL well-formed, then database name
// present and prefixed with cart_integration_. It never touches the
// network and its messages never include the raw URL or a password.
func checkConfig(rawURL string, allowDestructive string) configResult {
	if rawURL == "" {
		return configResult{skipReason: fmt.Sprintf("%s not set: skipping PostgreSQL integration test", urlEnvVar)}
	}
	if allowDestructive != "true" {
		return configResult{skipReason: fmt.Sprintf("%s not set to true: skipping PostgreSQL integration test", allowDestructiveEnvVar)}
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return configResult{fatal: fmt.Sprintf("%s is not a valid URL", urlEnvVar)}
	}

	dbName := trimLeadingSlash(parsed.Path)
	if dbName == "" {
		return configResult{fatal: fmt.Sprintf("%s must name a database, not the server root", urlEnvVar)}
	}
	if !strings.HasPrefix(dbName, requiredDatabaseNamePrefix) {
		return configResult{fatal: fmt.Sprintf("%s must target a database whose name starts with %q, got a name starting with %q", urlEnvVar, requiredDatabaseNamePrefix, firstRune(dbName))}
	}

	return configResult{dsn: rawURL}
}

func trimLeadingSlash(path string) string {
	if len(path) > 0 && path[0] == '/' {
		return path[1:]
	}
	return path
}

// firstRune returns a short, safe-to-print prefix of a database name for
// an error message — enough to diagnose a typo, never the full URL.
func firstRune(name string) string {
	if len(name) > 24 {
		return name[:24] + "…"
	}
	return name
}

// RequireIsolatedDatabase returns the configured integration-test database
// URL, or calls t.Skip (missing env var) / t.Fatal (present but invalid)
// with a message that never includes the URL or a password. Both
// CART_INTEGRATION_TEST_DATABASE_URL and
// CART_INTEGRATION_TEST_ALLOW_DESTRUCTIVE=true are required; DATABASE_URL
// and any other production variable are never consulted; and the target
// database name must start with cart_integration_.
func RequireIsolatedDatabase(t *testing.T) string {
	t.Helper()
	result := checkConfig(os.Getenv(urlEnvVar), os.Getenv(allowDestructiveEnvVar))
	if result.skipReason != "" {
		t.Skip(result.skipReason)
	}
	if result.fatal != "" {
		t.Fatal(result.fatal)
	}
	return result.dsn
}

// ResetDedicatedDatabase drops and recreates the public schema of the
// database addressed by dsn, and registers the same reset as t.Cleanup so
// the database is left empty afterward, including after a failing test.
// It re-validates the env-level guards itself — the same three checks
// RequireIsolatedDatabase already ran — because this is the function that
// actually performs the destructive operation, and it must never trust a
// caller to have checked first.
func ResetDedicatedDatabase(t *testing.T, dsn string) {
	t.Helper()

	result := checkConfig(os.Getenv(urlEnvVar), os.Getenv(allowDestructiveEnvVar))
	if result.skipReason != "" || result.fatal != "" || result.dsn != dsn {
		t.Fatal("dbtest: ResetDedicatedDatabase called with a dsn that does not match the validated guard; refusing to run a destructive reset")
	}

	resetPublicSchema(t, dsn)
	t.Cleanup(func() { resetPublicSchema(t, dsn) })
}

func resetPublicSchema(t *testing.T, dsn string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("dbtest: connect to reset public schema: %v", err)
	}
	defer pool.Close()

	warnIfOtherSessions(t, ctx, pool)

	if _, err := pool.Exec(ctx, `DROP SCHEMA IF EXISTS public CASCADE`); err != nil {
		t.Fatalf("dbtest: drop public schema: %v", err)
	}
	if _, err := pool.Exec(ctx, `CREATE SCHEMA public`); err != nil {
		t.Fatalf("dbtest: recreate public schema: %v", err)
	}
}

// warnIfOtherSessions is a best-effort check (section 5.3: "cuando sea
// posible"): it cannot be a hard gate, because the very pool used to check
// necessarily holds its own backend, and a CI runner may briefly show a
// second idle connection during pool warmup. A high count is logged, not
// fatal — it is a signal for a human to notice the database is not as
// dedicated as its name claims, not a flaky test failure.
func warnIfOtherSessions(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	var otherSessions int
	err := pool.QueryRow(ctx,
		`SELECT count(*) FROM pg_stat_activity WHERE datname = current_database() AND pid <> pg_backend_pid()`,
	).Scan(&otherSessions)
	if err != nil {
		t.Logf("dbtest: could not check for other sessions on this database: %v", err)
		return
	}
	if otherSessions > 0 {
		t.Logf("dbtest: %d other session(s) connected to this database besides this test run — confirm it is truly dedicated to integration testing", otherSessions)
	}
}

// ApplyMigrationsUp runs the real, complete, unmodified sql/migrations
// chain against dsn with the goose binary the repository's normal
// migration workflow already depends on (see AGENTS.md) — no subset, no
// copy, no new Go dependency.
func ApplyMigrationsUp(t *testing.T, dsn string) {
	t.Helper()
	runGoose(t, dsn, "up")
}

// ApplyIdempotencyMigrationDown reverts exactly
// 20250902000000_add_cart_idempotency_keys_table.sql (goose down-to the
// migration immediately before it), leaving every earlier migration
// (including carts and cart_items) intact.
func ApplyIdempotencyMigrationDown(t *testing.T, dsn string) {
	t.Helper()
	runGoose(t, dsn, "down-to", "20250809194425")
}

// ApplyIdempotencyMigrationUp reapplies the idempotency migration after
// ApplyIdempotencyMigrationDown.
func ApplyIdempotencyMigrationUp(t *testing.T, dsn string) {
	t.Helper()
	runGoose(t, dsn, "up")
}

// ApplyImagesTableUpdatesMigrationDown reverts exactly
// 20250901230135_images_table_updates.sql (goose down-to the migration
// immediately before it, 20250901203644_add_social_media_table.sql),
// leaving every earlier migration — including the base images table itself
// from 20250703195451_add_images_table.sql and its FK dependents — intact.
func ApplyImagesTableUpdatesMigrationDown(t *testing.T, dsn string) {
	t.Helper()
	runGoose(t, dsn, "down-to", "20250901203644")
}

// ApplyImagesTableUpdatesMigrationUp reapplies migrations after
// ApplyImagesTableUpdatesMigrationDown, back up through the full chain.
func ApplyImagesTableUpdatesMigrationUp(t *testing.T, dsn string) {
	t.Helper()
	runGoose(t, dsn, "up")
}

// LastAppliedMigrationVersion reads goose's own version table and returns
// the highest applied migration id, so a test can confirm the full chain
// really reached the end rather than stopping partway.
func LastAppliedMigrationVersion(t *testing.T, dsn string) int64 {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("dbtest: connect to read goose version: %v", err)
	}
	defer pool.Close()

	var version int64
	err = pool.QueryRow(ctx, `SELECT version_id FROM goose_db_version ORDER BY id DESC LIMIT 1`).Scan(&version)
	if err != nil {
		t.Fatalf("dbtest: read goose_db_version: %v", err)
	}
	return version
}

func runGoose(t *testing.T, dsn string, args ...string) {
	t.Helper()
	fullArgs := append([]string{"-dir", repoMigrationsDir(t), "postgres", dsn}, args...)
	cmd := exec.Command("goose", fullArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("dbtest: goose %v failed: %v\n%s", args, err, output)
	}
}

// repoMigrationsDir locates the real sql/migrations directory from the
// calling test package's working directory (either internal/db or
// internal/cart, both two levels below the repository root). Nothing is
// copied out of it; goose reads it in place.
func repoMigrationsDir(t *testing.T) string {
	t.Helper()
	for _, candidate := range []string{
		filepath.Join("..", "..", "sql", "migrations"),
		filepath.Join("..", "sql", "migrations"),
		filepath.Join("sql", "migrations"),
	} {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			abs, err := filepath.Abs(candidate)
			if err != nil {
				t.Fatalf("dbtest: resolve migrations dir: %v", err)
			}
			return abs
		}
	}
	t.Fatal("dbtest: could not locate sql/migrations from the test working directory")
	return ""
}

// NewPool opens a pgxpool.Pool against dsn with a short connect timeout,
// and registers its Close with t.Cleanup. Each call returns an independent
// pool — nothing here is a shared or global connection.
func NewPool(t *testing.T, dsn string) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("dbtest: open pool: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("dbtest: ping pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// DeadlockDetected reports whether err is PostgreSQL's deadlock_detected
// (SQLSTATE 40P01), so a test can assert a real deadlock was surfaced
// rather than hidden by a retry loop — this suite implements no such
// retries.
func DeadlockDetected(err error) bool {
	type sqlStater interface{ SQLState() string }
	var state sqlStater
	if errors.As(err, &state) {
		return state.SQLState() == "40P01"
	}
	return false
}
