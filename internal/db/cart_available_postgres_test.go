package db

import (
	"context"
	"testing"

	"github.com/vladwithcode/salon_catalog/internal/dbtest"
)

// TestPostgresLoadItemsReportsAvailableIndependentlyOfQuantity covers Fase
// 12 section 7: CartItem.Available (added in Fase 11 for cotización's
// validateQuoteCart) reflects catalog_products.available exactly, is never
// inferred from quantity, and does not disturb LoadItems' other columns.
func TestPostgresLoadItemsReportsAvailableIndependentlyOfQuantity(t *testing.T) {
	dsn := setupDedicatedDatabase(t)
	pool := dbtest.NewPool(t, dsn)
	ctx := context.Background()

	// Cart.LoadItems acquires from the package-level pool (GetConnWithContext),
	// distinct from the dbtest.NewPool used for direct fixture inserts above —
	// point it at the same dedicated database for this test.
	t.Setenv("DATABASE_URL", dsn)
	if _, err := Connect(); err != nil {
		t.Fatalf("connect package db pool: %v", err)
	}
	t.Cleanup(Close)

	cartID := insertTestCart(t, pool)

	availableZeroStock := insertTestProduct(t, pool, true, 0)
	unavailablePlentyStock := insertTestProduct(t, pool, false, 999)

	if _, err := pool.Exec(ctx,
		`INSERT INTO cart_items (cart_id, product_id, quantity, source, created_at, updated_at) VALUES ($1, $2, 1, 'catálogo', NOW(), NOW())`,
		cartID, availableZeroStock,
	); err != nil {
		t.Fatalf("insert cart item (available, zero stock): %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO cart_items (cart_id, product_id, quantity, source, created_at, updated_at) VALUES ($1, $2, 1, 'catálogo', NOW(), NOW())`,
		cartID, unavailablePlentyStock,
	); err != nil {
		t.Fatalf("insert cart item (unavailable, plenty stock): %v", err)
	}

	cart := &Cart{ID: cartID}
	if err := cart.LoadItems(ctx); err != nil {
		t.Fatalf("LoadItems: %v", err)
	}
	if len(cart.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(cart.Items))
	}

	byProduct := map[string]*CartItem{}
	for _, item := range cart.Items {
		byProduct[item.ProductID] = item
	}

	// available=true survives even with quantity=0 in the DB — Available
	// is never derived from MaxQty/quantity, it comes straight from
	// catalog_products.available.
	if item := byProduct[availableZeroStock]; item == nil || !item.Available {
		t.Fatalf("expected available=true regardless of zero stock, got %+v", item)
	}
	if item := byProduct[availableZeroStock]; item.MaxQty != 0 {
		t.Fatalf("expected max_quantity=0 preserved, got %d", item.MaxQty)
	}

	// available=false survives even with abundant stock.
	if item := byProduct[unavailablePlentyStock]; item == nil || item.Available {
		t.Fatalf("expected available=false regardless of stock, got %+v", item)
	}
	if item := byProduct[unavailablePlentyStock]; item.MaxQty != 999 {
		t.Fatalf("expected max_quantity=999 preserved, got %d", item.MaxQty)
	}
}
