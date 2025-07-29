// Package internal contains utility functions for the application
package internal

import (
	"net/http"
	"regexp"
	"strings"
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
