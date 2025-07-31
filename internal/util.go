// Package internal contains utility functions for the application
package internal

import (
	"crypto/rand"
	"net/http"
	"regexp"
	"strings"
	"unsafe"
)

func Slugify(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ToLower(s)

	s = string(
		regexp.MustCompile("[^a-z0-9-]+").
			ReplaceAll(
				[]byte(s),
				[]byte(""),
			),
	)

	return s
}

func HandleRedirect(toRoute string, code int, w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Add("HX-Redirect", toRoute)
		w.WriteHeader(code)
	} else {
		http.Redirect(w, r, toRoute, code)
	}
}

func PtrSliceToPlainSlice[T any](s []*T) []T {
	var res = make([]T, len(s))
	for i, v := range s {
		res[i] = *v
	}
	return res
}

const (
	// HTML ID must start with a letter, then can contain letters, digits, hyphens, underscores, colons, periods
	// We'll use only letters and digits for simplicity and compatibility
	idChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	// First character must be a letter (HTML spec requirement)
	letterChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

// GenerateHTMLID generates a cryptographically secure alphanumeric ID suitable for HTML elements.
// The ID will always start with a letter (as required by HTML spec) followed by random alphanumeric characters.
// Default length is 8 characters, but can be customized.
func GenerateHTMLID(length ...int) string {
	idLength := 8 // Default length
	if len(length) > 0 && length[0] > 1 {
		idLength = length[0]
	}

	// Pre-allocate byte slice for performance
	result := make([]byte, idLength)

	// Generate random bytes for the entire ID
	randomBytes := make([]byte, idLength)
	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback to a simple deterministic method if crypto/rand fails
		// This should never happen in normal circumstances
		return generateFallbackID(idLength)
	}

	// First character must be a letter
	result[0] = letterChars[int(randomBytes[0])%len(letterChars)]

	// Remaining characters can be letters or digits
	for i := 1; i < idLength; i++ {
		result[i] = idChars[int(randomBytes[i])%len(idChars)]
	}

	// Convert bytes to string without allocation using unsafe
	return *(*string)(unsafe.Pointer(&result))
}

// generateFallbackID provides a fallback method if crypto/rand fails
// Uses a simple but deterministic approach
func generateFallbackID(length int) string {
	// This is a very rare fallback - in production you might want to log this
	// For now, return a simple valid HTML ID
	if length < 1 {
		length = 8
	}

	result := make([]byte, length)
	result[0] = 'a' // Ensure it starts with a letter

	for i := 1; i < length; i++ {
		result[i] = 'x' // Simple fallback
	}

	return string(result)
}
