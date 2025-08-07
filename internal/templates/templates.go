// Package templates contains the template files for the application
//
// This file also contains some helper functions for the templates
package templates

import (
	"context"
	"strings"

	"github.com/vladwithcode/salon_catalog/internal/auth"
)

func GetUserInitals(ctx context.Context) string {
	user, err := auth.ExtractAuthFromCtx(ctx)
	if err != nil || user == nil {
		return "?"
	}
	nameParts := strings.SplitN(user.Fullname, " ", 2)
	return nameParts[0][0:1]
}
