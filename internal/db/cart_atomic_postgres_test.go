package db

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vladwithcode/salon_catalog/internal/dbtest"
)

// setupDedicatedDatabase validates the guard, resets the dedicated
// database's public schema (registering its own cleanup), and applies the
// real, complete, unmodified sql/migrations chain. Every Postgres test in
// this file starts from here — no schema subset, no copied SQL.
func setupDedicatedDatabase(t *testing.T) string {
	t.Helper()
	dsn := dbtest.RequireIsolatedDatabase(t)
	dbtest.ResetDedicatedDatabase(t, dsn)
	dbtest.ApplyMigrationsUp(t, dsn)
	return dsn
}

// TestPostgresFullMigrationChainCreatesRequiredTables runs the entire
// ordered sql/migrations history — including the migrations that hardcode
// "public." — against a dedicated database's own public schema, and
// confirms carts, cart_items, products, and cart_idempotency_keys all
// exist afterward, plus that goose's own version table shows the chain
// reached its last entry.
func TestPostgresFullMigrationChainCreatesRequiredTables(t *testing.T) {
	dsn := dbtest.RequireIsolatedDatabase(t)
	dbtest.ResetDedicatedDatabase(t, dsn)

	assertTableMissing(t, dsn, "carts")

	dbtest.ApplyMigrationsUp(t, dsn)

	for _, table := range []string{"carts", "cart_items", "products", "cart_idempotency_keys"} {
		assertTableExists(t, dsn, table)
	}

	// The idempotency migration's timestamp (20251001000000) is the newest
	// in sql/migrations as of this phase; confirm the chain actually
	// reached it rather than stopping early on some other migration.
	if version := dbtest.LastAppliedMigrationVersion(t, dsn); version != 20251001000000 {
		t.Fatalf("expected the full chain to end at 20251001000000, goose reports %d", version)
	}
}

// TestPostgresIdempotencyMigrationDownAndReup exercises section 10: down
// removes only cart_idempotency_keys, every earlier migration's tables
// survive, and a subsequent up leaves the database usable again.
func TestPostgresIdempotencyMigrationDownAndReup(t *testing.T) {
	dsn := setupDedicatedDatabase(t)
	pool := dbtest.NewPool(t, dsn)
	ctx := context.Background()

	cartID := insertTestCart(t, pool)
	assertTableExists(t, dsn, "cart_idempotency_keys")

	dbtest.ApplyIdempotencyMigrationDown(t, dsn)
	assertTableMissing(t, dsn, "cart_idempotency_keys")

	// carts and cart_items must survive the down migration untouched.
	var stillThere string
	if err := pool.QueryRow(ctx, `SELECT id FROM carts WHERE id = $1`, cartID).Scan(&stillThere); err != nil {
		t.Fatalf("carts row lost after idempotency migration down: %v", err)
	}

	dbtest.ApplyIdempotencyMigrationUp(t, dsn)
	assertTableExists(t, dsn, "cart_idempotency_keys")

	// The database must accept a real insert again after the reapplied up.
	if _, err := pool.Exec(ctx,
		`INSERT INTO cart_idempotency_keys (cart_id, key_hash, request_hash, created_at, expires_at)
		 VALUES ($1, $2, $3, NOW(), NOW() + interval '1 hour')`,
		cartID, fixed32Bytes(1), fixed32Bytes(2),
	); err != nil {
		t.Fatalf("insert after reapplied up migration failed: %v", err)
	}
}

// --- Constraints (section 9) ---

func TestPostgresCartsPrimaryKeyRejectsDuplicateUUID(t *testing.T) {
	dsn := setupDedicatedDatabase(t)
	pool := dbtest.NewPool(t, dsn)
	ctx := context.Background()

	id := uuid.New().String()
	if _, err := pool.Exec(ctx, `INSERT INTO carts (id) VALUES ($1)`, id); err != nil {
		t.Fatalf("first insert: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO carts (id) VALUES ($1)`, id); err == nil {
		t.Fatal("expected duplicate carts.id to be rejected by the primary key")
	}
}

func TestPostgresCartItemsPrimaryKeyRejectsDuplicateLine(t *testing.T) {
	dsn := setupDedicatedDatabase(t)
	pool := dbtest.NewPool(t, dsn)
	ctx := context.Background()

	cartID := insertTestCart(t, pool)
	productID := insertTestProduct(t, pool, true, 10)

	if _, err := pool.Exec(ctx,
		`INSERT INTO cart_items (cart_id, product_id, quantity, source, created_at, updated_at) VALUES ($1, $2, 1, 'catalog', NOW(), NOW())`,
		cartID, productID,
	); err != nil {
		t.Fatalf("first cart_items insert: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO cart_items (cart_id, product_id, quantity, source, created_at, updated_at) VALUES ($1, $2, 1, 'catalog', NOW(), NOW())`,
		cartID, productID,
	); err == nil {
		t.Fatal("expected duplicate (cart_id, product_id) to be rejected")
	}
}

func TestPostgresCartItemsForeignKeyRequiresExistingCart(t *testing.T) {
	dsn := setupDedicatedDatabase(t)
	pool := dbtest.NewPool(t, dsn)
	ctx := context.Background()

	productID := insertTestProduct(t, pool, true, 10)
	if _, err := pool.Exec(ctx,
		`INSERT INTO cart_items (cart_id, product_id, quantity, source, created_at, updated_at) VALUES ($1, $2, 1, 'catalog', NOW(), NOW())`,
		uuid.New().String(), productID,
	); err == nil {
		t.Fatal("expected FK violation for a cart_items row with no matching cart")
	}
}

func TestPostgresCartItemsCascadeOnCartDelete(t *testing.T) {
	dsn := setupDedicatedDatabase(t)
	pool := dbtest.NewPool(t, dsn)
	ctx := context.Background()

	cartID := insertTestCart(t, pool)
	productID := insertTestProduct(t, pool, true, 10)
	if _, err := pool.Exec(ctx,
		`INSERT INTO cart_items (cart_id, product_id, quantity, source, created_at, updated_at) VALUES ($1, $2, 1, 'catalog', NOW(), NOW())`,
		cartID, productID,
	); err != nil {
		t.Fatalf("insert cart_items: %v", err)
	}

	if _, err := pool.Exec(ctx, `DELETE FROM carts WHERE id = $1`, cartID); err != nil {
		t.Fatalf("delete cart: %v", err)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM cart_items WHERE cart_id = $1`, cartID).Scan(&count); err != nil {
		t.Fatalf("count cart_items: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected cascade delete of cart_items, found %d rows", count)
	}
}

func TestPostgresIdempotencyKeysConstraints(t *testing.T) {
	dsn := setupDedicatedDatabase(t)
	pool := dbtest.NewPool(t, dsn)
	ctx := context.Background()

	cartID := insertTestCart(t, pool)

	t.Run("primary key on cart_id, key_hash", func(t *testing.T) {
		keyHash := fixed32Bytes(9)
		insertIdempotencyRow(t, pool, cartID, keyHash, fixed32Bytes(1), time.Hour)
		if _, err := pool.Exec(ctx,
			`INSERT INTO cart_idempotency_keys (cart_id, key_hash, request_hash, created_at, expires_at) VALUES ($1, $2, $3, NOW(), NOW() + interval '1 hour')`,
			cartID, keyHash, fixed32Bytes(2),
		); err == nil {
			t.Fatal("expected duplicate (cart_id, key_hash) to be rejected")
		}
	})

	t.Run("cascade on cart delete", func(t *testing.T) {
		cascadeCartID := insertTestCart(t, pool)
		insertIdempotencyRow(t, pool, cascadeCartID, fixed32Bytes(11), fixed32Bytes(12), time.Hour)
		if _, err := pool.Exec(ctx, `DELETE FROM carts WHERE id = $1`, cascadeCartID); err != nil {
			t.Fatalf("delete cart: %v", err)
		}
		var count int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM cart_idempotency_keys WHERE cart_id = $1`, cascadeCartID).Scan(&count); err != nil {
			t.Fatalf("count: %v", err)
		}
		if count != 0 {
			t.Fatalf("expected cascade delete of idempotency claim, found %d rows", count)
		}
	})

	t.Run("key_hash must be 32 bytes", func(t *testing.T) {
		if _, err := pool.Exec(ctx,
			`INSERT INTO cart_idempotency_keys (cart_id, key_hash, request_hash, created_at, expires_at) VALUES ($1, $2, $3, NOW(), NOW() + interval '1 hour')`,
			cartID, []byte{1, 2, 3}, fixed32Bytes(3),
		); err == nil {
			t.Fatal("expected key_hash length check to reject a non-32-byte value")
		}
	})

	t.Run("request_hash must be 32 bytes", func(t *testing.T) {
		if _, err := pool.Exec(ctx,
			`INSERT INTO cart_idempotency_keys (cart_id, key_hash, request_hash, created_at, expires_at) VALUES ($1, $2, $3, NOW(), NOW() + interval '1 hour')`,
			cartID, fixed32Bytes(20), []byte{1, 2, 3},
		); err == nil {
			t.Fatal("expected request_hash length check to reject a non-32-byte value")
		}
	})

	t.Run("expires_at must be after created_at", func(t *testing.T) {
		if _, err := pool.Exec(ctx,
			`INSERT INTO cart_idempotency_keys (cart_id, key_hash, request_hash, created_at, expires_at) VALUES ($1, $2, $3, NOW(), NOW() - interval '1 hour')`,
			cartID, fixed32Bytes(21), fixed32Bytes(22),
		); err == nil {
			t.Fatal("expected expires_at <= created_at to be rejected")
		}
	})

	t.Run("index on expires_at exists", func(t *testing.T) {
		var indexName string
		err := pool.QueryRow(ctx,
			`SELECT indexname FROM pg_indexes WHERE tablename = 'cart_idempotency_keys' AND indexdef ILIKE '%expires_at%'`,
		).Scan(&indexName)
		if err != nil {
			t.Fatalf("expected an index on expires_at, query failed: %v", err)
		}
	})
}

// --- Test fixtures and helpers ---

func assertTableExists(t *testing.T, dsn string, table string) {
	t.Helper()
	pool := dbtest.NewPool(t, dsn)
	var found string
	err := pool.QueryRow(context.Background(), `SELECT to_regclass($1)::text`, table).Scan(&found)
	if err != nil || found == "" {
		t.Fatalf("expected table %q to exist: err=%v found=%q", table, err, found)
	}
}

func assertTableMissing(t *testing.T, dsn string, table string) {
	t.Helper()
	pool := dbtest.NewPool(t, dsn)
	var found *string
	err := pool.QueryRow(context.Background(), `SELECT to_regclass($1)::text`, table).Scan(&found)
	if err != nil {
		t.Fatalf("checking absence of table %q: %v", table, err)
	}
	if found != nil {
		t.Fatalf("expected table %q not to exist yet", table)
	}
}

func insertTestCart(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	id := uuid.New().String()
	if _, err := pool.Exec(context.Background(), `INSERT INTO carts (id) VALUES ($1)`, id); err != nil {
		t.Fatalf("insert test cart: %v", err)
	}
	return id
}

// insertTestProduct creates a minimal product row with no category or
// image (both nullable), so fixtures never depend on data outside the
// tables this suite actually needs.
func insertTestProduct(t *testing.T, pool *pgxpool.Pool, available bool, quantity int) string {
	t.Helper()
	id := uuid.New().String()
	slug := "test-product-" + id
	_, err := pool.Exec(context.Background(),
		`INSERT INTO products (id, name, slug, description, available, quantity) VALUES ($1, 'Test Product', $2, 'test', $3, $4)`,
		id, slug, available, quantity,
	)
	if err != nil {
		t.Fatalf("insert test product: %v", err)
	}
	return id
}

func insertIdempotencyRow(t *testing.T, pool *pgxpool.Pool, cartID string, keyHash []byte, requestHash []byte, ttl time.Duration) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO cart_idempotency_keys (cart_id, key_hash, request_hash, created_at, expires_at) VALUES ($1, $2, $3, NOW(), NOW() + $4)`,
		cartID, keyHash, requestHash, ttl,
	)
	if err != nil {
		t.Fatalf("insert idempotency row: %v", err)
	}
}

func fixed32Bytes(seed byte) []byte {
	buf := make([]byte, 32)
	for i := range buf {
		buf[i] = seed
	}
	return buf
}
