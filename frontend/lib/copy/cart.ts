// Fixed, safe copy only — mirrors internal/routes/cart.go's
// cartStatusMessages/cartErrorMessages exactly. Never renders raw query
// values; a cart_status/cart_error code that isn't one of these keys shows
// nothing.

export const cartStatusMessages: Record<string, string> = {
  added: "Se añadió el producto a tu selección.",
  updated: "Se actualizó tu selección.",
  removed: "Se eliminó el producto de tu selección.",
  cleared: "Se limpió tu selección.",
};

export const cartErrorMessages: Record<string, string> = {
  invalid_request: "Revisa los datos e intenta de nuevo.",
  request_too_large: "La solicitud es demasiado grande.",
  unsupported_media_type: "Ocurrió un error inesperado.",
  product_not_found: "No encontramos ese producto.",
  product_unavailable: "Ese producto ya no está disponible.",
  insufficient_stock: "No hay stock suficiente disponible.",
  cart_item_not_found: "Ese elemento ya no está en tu selección.",
  idempotency_key_required: "Ocurrió un error inesperado.",
  invalid_idempotency_key: "Ocurrió un error inesperado.",
  idempotency_conflict: "Esa solicitud ya se procesó con otros datos. Intenta de nuevo.",
  cart_unavailable: "Error al actualizar el carrito.",
  backend_unavailable: "El carrito no está disponible en este momento.",
  invalid_response: "El carrito no está disponible en este momento.",
};

export function cartStatusMessageFor(code: string | undefined): string | null {
  if (!code) return null;
  return cartStatusMessages[code] ?? null;
}

export function cartErrorMessageFor(code: string | undefined): string | null {
  if (!code) return null;
  return cartErrorMessages[code] ?? null;
}
