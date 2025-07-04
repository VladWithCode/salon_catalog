package internal

import (
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
