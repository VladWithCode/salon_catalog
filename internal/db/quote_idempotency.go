package db

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// QuoteIdempotencyTTL matches the cart's own idempotency window
// (internal/cart/service.go's idempotencyTTL) — long enough to cover a
// realistic retry (network error, slow response, double click) without
// keeping claims alive indefinitely.
const QuoteIdempotencyTTL = 24 * time.Hour

var (
	// ErrQuoteIdempotencyConflict means the same key was reused with a
	// different request payload — never silently applied.
	ErrQuoteIdempotencyConflict = errors.New("quote idempotency conflict")
	ErrQuoteSubmitUnavailable   = errors.New("quote submission unavailable")
)

type QuoteSubmitOutcome int

const (
	QuoteSubmitApplied QuoteSubmitOutcome = iota
	QuoteSubmitReplayed
)

// SubmitQuoteIdempotent claims (cartID, keyHash) and inserts quote inside a
// single transaction — the same shape as internal/cart.Service's
// AddItemIdempotent (internal/db/cart_atomic.go), adapted for quotes: no
// stock to lock, but the same "claim and mutate together, so a failed
// mutation rolls back its own claim" guarantee.
//
// Concurrency: the transaction locks the owning carts row
// (`SELECT ... FOR UPDATE`) before reading or writing
// quote_idempotency_keys, serializing two concurrent submissions for the
// same cart/session — the second waits for the first's commit or rollback,
// then sees its committed claim (replay) or an empty slot (new claim after
// a rolled-back attempt). This mirrors EnsureAndLockCart's role in the cart
// service exactly.
func SubmitQuoteIdempotent(ctx context.Context, cartID string, keyHash, requestHash []byte, quote *Quote, now time.Time) (QuoteSubmitOutcome, error) {
	if len(keyHash) != 32 || len(requestHash) != 32 {
		return QuoteSubmitApplied, ErrQuoteSubmitUnavailable
	}

	conn, err := GetConnWithContext(ctx)
	if err != nil {
		return QuoteSubmitApplied, ErrQuoteSubmitUnavailable
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return QuoteSubmitApplied, ErrQuoteSubmitUnavailable
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after a successful Commit

	var lockedCartID string
	if err := tx.QueryRow(ctx, `SELECT id FROM carts WHERE id = $1 FOR UPDATE`, cartID).Scan(&lockedCartID); err != nil {
		return QuoteSubmitApplied, ErrQuoteSubmitUnavailable
	}

	var existingRequestHash []byte
	var expiresAt time.Time
	err = tx.QueryRow(ctx,
		`SELECT request_hash, expires_at FROM quote_idempotency_keys WHERE cart_id = $1 AND key_hash = $2 FOR UPDATE`,
		cartID, keyHash,
	).Scan(&existingRequestHash, &expiresAt)
	switch {
	case err == nil:
		if now.Before(expiresAt) {
			if !bytes.Equal(existingRequestHash, requestHash) {
				return QuoteSubmitApplied, ErrQuoteIdempotencyConflict
			}
			// Same key, same payload, still valid: replay. Do not create a
			// second quote — just close out the transaction holding the
			// cart lock acquired above.
			if err := tx.Commit(ctx); err != nil {
				return QuoteSubmitApplied, ErrQuoteSubmitUnavailable
			}
			return QuoteSubmitReplayed, nil
		}
		// Claim expired: delete it so the key can be reclaimed as a new
		// operation, then fall through to claim + create below.
		if _, err := tx.Exec(ctx, `DELETE FROM quote_idempotency_keys WHERE cart_id = $1 AND key_hash = $2`, cartID, keyHash); err != nil {
			return QuoteSubmitApplied, ErrQuoteSubmitUnavailable
		}
	case errors.Is(err, pgx.ErrNoRows):
		// No existing claim — proceed to claim + create below.
	default:
		return QuoteSubmitApplied, ErrQuoteSubmitUnavailable
	}

	expires := now.Add(QuoteIdempotencyTTL)
	if _, err := tx.Exec(ctx,
		`INSERT INTO quote_idempotency_keys (cart_id, key_hash, request_hash, created_at, expires_at) VALUES ($1, $2, $3, $4, $5)`,
		cartID, keyHash, requestHash, now, expires,
	); err != nil {
		return QuoteSubmitApplied, ErrQuoteSubmitUnavailable
	}

	id, err := uuid.NewV7()
	if err != nil {
		return QuoteSubmitApplied, ErrUUIDFail
	}

	var timeStart, timeEnd sql.NullTime
	if quote.TimeStart != nil {
		timeStart = sql.NullTime{Time: *quote.TimeStart, Valid: true}
	}
	if quote.TimeEnd != nil {
		timeEnd = sql.NullTime{Time: *quote.TimeEnd, Valid: true}
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO quotes (
			id, customer_name, customer_phone, time_start, time_end, status, comments, cart_id, request_type, event_kind_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		id.String(),
		quote.CustomerName,
		quote.CustomerPhone,
		timeStart,
		timeEnd,
		string(quote.Status),
		quote.Comments,
		quote.CartID,
		string(quote.RequestType),
		quote.EventKindID,
	); err != nil {
		// A failure here rolls back the whole transaction (deferred
		// tx.Rollback runs since Commit was never reached) — the claim
		// inserted above is undone along with it, so the same key can be
		// retried as a fresh attempt rather than being permanently stuck
		// behind a claim for a quote that was never actually created.
		return QuoteSubmitApplied, ErrQuoteSubmitUnavailable
	}
	quote.ID = id.String()

	if err := tx.Commit(ctx); err != nil {
		return QuoteSubmitApplied, ErrQuoteSubmitUnavailable
	}
	return QuoteSubmitApplied, nil
}
