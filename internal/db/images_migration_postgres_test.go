package db

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vladwithcode/salon_catalog/internal/dbtest"
)

// TestPostgresImagesTableUpdatesMigrationDownAndReup covers Fase 11
// section 2's 10 required checks for
// sql/migrations/20250901230135_images_table_updates.sql, whose Down block
// was fixed (DROP TABLE images -> targeted column/index drops) in Fase 10.
// Up was never touched; this test only proves Down/re-up correctness.
func TestPostgresImagesTableUpdatesMigrationDownAndReup(t *testing.T) {
	// 1. Up of the full chain.
	dsn := setupDedicatedDatabase(t)
	pool := dbtest.NewPool(t, dsn)
	ctx := context.Background()

	// 2. State of images before Down: base columns plus the columns this
	// migration's Up added, and unrelated test data that must survive.
	imgID := insertTestImage(t, pool, "before-down.jpg")
	assertTableExists(t, dsn, "images")
	assertColumnExists(t, dsn, "images", "file_type")
	assertColumnExists(t, dsn, "images", "updated_at")
	assertIndexExists(t, dsn, "images_filetype_idx")

	prodID := insertTestProduct(t, pool, true, 5)
	linkImageToProduct(t, pool, imgID, prodID)

	// 3. Down of exactly this migration (down-to the migration right
	// before it — does not touch anything earlier in the chain).
	dbtest.ApplyImagesTableUpdatesMigrationDown(t, dsn)

	// 4. images table still exists (Down never drops the table).
	assertTableExists(t, dsn, "images")

	// 5. Dependent tables (FK to images.id) still exist.
	assertTableExists(t, dsn, "images_products")

	// 6. Prior constraints (base image row, from before this migration's
	// Up ever ran) still exist — the original image row and its FK link
	// survive because Down only drops columns/index, never rows.
	var stillThere string
	if err := pool.QueryRow(ctx, `SELECT id FROM images WHERE id = $1`, imgID).Scan(&stillThere); err != nil {
		t.Fatalf("images row lost after this migration's down: %v", err)
	}
	var linkCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM images_products WHERE image_id = $1 AND product_id = $2`, imgID, prodID).Scan(&linkCount); err != nil {
		t.Fatalf("check images_products survival: %v", err)
	}
	if linkCount != 1 {
		t.Fatalf("expected images_products link to survive down, got count=%d", linkCount)
	}

	// 7. Elements added by Up were actually reverted.
	assertColumnMissing(t, dsn, "images", "file_type")
	assertColumnMissing(t, dsn, "images", "updated_at")
	assertIndexMissing(t, dsn, "images_filetype_idx")

	// 8. Re-up.
	dbtest.ApplyImagesTableUpdatesMigrationUp(t, dsn)
	assertColumnExists(t, dsn, "images", "file_type")
	assertColumnExists(t, dsn, "images", "updated_at")
	assertIndexExists(t, dsn, "images_filetype_idx")

	// 9. Unrelated test data (the pre-existing image/product/link rows)
	// survives the down+up round trip, and the table accepts a real insert
	// again under the reapplied NOT NULL DEFAULT columns.
	if err := pool.QueryRow(ctx, `SELECT id FROM images WHERE id = $1`, imgID).Scan(&stillThere); err != nil {
		t.Fatalf("images row lost after re-up: %v", err)
	}
	newImgID := insertTestImage(t, pool, "after-reup.jpg")
	var fileType string
	if err := pool.QueryRow(ctx, `SELECT file_type FROM images WHERE id = $1`, newImgID).Scan(&fileType); err != nil {
		t.Fatalf("insert after reapplied up failed to read back file_type: %v", err)
	}
	if fileType != "image/jpeg" {
		t.Fatalf("expected default file_type 'image/jpeg', got %q", fileType)
	}

	// 10. goose ends on the correct (last) version — same assertion as the
	// full-chain test, confirming down-to + up left the chain intact.
	if version := dbtest.LastAppliedMigrationVersion(t, dsn); version != 20251001000000 {
		t.Fatalf("expected chain to end at 20251001000000 after down/re-up, goose reports %d", version)
	}
}

func insertTestImage(t *testing.T, pool *pgxpool.Pool, filename string) string {
	t.Helper()
	id := uuid.New().String()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO images (id, filename, name, size) VALUES ($1, $2, $2, 1)`,
		id, filename,
	)
	if err != nil {
		t.Fatalf("insert test image: %v", err)
	}
	return id
}

func linkImageToProduct(t *testing.T, pool *pgxpool.Pool, imageID, productID string) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO images_products (image_id, product_id) VALUES ($1, $2)`,
		imageID, productID,
	)
	if err != nil {
		t.Fatalf("link image to product: %v", err)
	}
}

func assertColumnExists(t *testing.T, dsn, table, column string) {
	t.Helper()
	pool := dbtest.NewPool(t, dsn)
	var found string
	err := pool.QueryRow(context.Background(),
		`SELECT column_name FROM information_schema.columns WHERE table_schema = 'public' AND table_name = $1 AND column_name = $2`,
		table, column,
	).Scan(&found)
	if err != nil {
		t.Fatalf("expected column %s.%s to exist: %v", table, column, err)
	}
}

func assertColumnMissing(t *testing.T, dsn, table, column string) {
	t.Helper()
	pool := dbtest.NewPool(t, dsn)
	var found string
	err := pool.QueryRow(context.Background(),
		`SELECT column_name FROM information_schema.columns WHERE table_schema = 'public' AND table_name = $1 AND column_name = $2`,
		table, column,
	).Scan(&found)
	if err == nil {
		t.Fatalf("expected column %s.%s not to exist", table, column)
	}
}

func assertIndexExists(t *testing.T, dsn, index string) {
	t.Helper()
	pool := dbtest.NewPool(t, dsn)
	var found string
	err := pool.QueryRow(context.Background(), `SELECT to_regclass($1)::text`, index).Scan(&found)
	if err != nil || found == "" {
		t.Fatalf("expected index %q to exist: err=%v found=%q", index, err, found)
	}
}

func assertIndexMissing(t *testing.T, dsn, index string) {
	t.Helper()
	pool := dbtest.NewPool(t, dsn)
	var found *string
	err := pool.QueryRow(context.Background(), `SELECT to_regclass($1)::text`, index).Scan(&found)
	if err != nil {
		t.Fatalf("checking absence of index %q: %v", index, err)
	}
	if found != nil {
		t.Fatalf("expected index %q not to exist", index)
	}
}
